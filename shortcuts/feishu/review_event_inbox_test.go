package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func insertReviewEventTest(t *testing.T, store *SQLiteReviewGatewayStore, eventName, action string) ReviewEventInboxRecord {
	t.Helper()
	normalization := normalizeReviewEventTest(t, eventName, action)
	result, err := store.InsertReviewEventInbox(context.Background(), normalization, ReviewEventIngressMetadata{
		SignatureStatus: "verified", TimestampStatus: "valid", DeliveryStatus: "header",
		RawPayload: []byte(`{"authorization":"secret","token":"secret","safe":"diagnostic"}`),
	})
	if err != nil {
		t.Fatalf("InsertReviewEventInbox: %v", err)
	}
	return result.Record
}

func TestInboxInsertIsIdempotentByInstallationAndDelivery(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	normalization := normalizeReviewEventTest(t, "pull_request", "opened")
	first, err := store.InsertReviewEventInbox(context.Background(), normalization, ReviewEventIngressMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.InsertReviewEventInbox(context.Background(), normalization, ReviewEventIngressMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Duplicate || !second.Duplicate || first.Record.Event.EventID != second.Record.Event.EventID {
		t.Fatalf("idempotency failure: first=%+v second=%+v", first, second)
	}
}

func TestDuplicateDeliveryDoesNotOverwriteOriginalEvent(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	first := normalizeReviewEventTest(t, "pull_request", "opened")
	second := normalizeReviewEventTest(t, "pull_request", "closed")
	if _, err := store.InsertReviewEventInbox(context.Background(), first, ReviewEventIngressMetadata{}); err != nil {
		t.Fatal(err)
	}
	result, err := store.InsertReviewEventInbox(context.Background(), second, ReviewEventIngressMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Duplicate || result.Record.Event.Action != "pull_request.opened" {
		t.Fatalf("duplicate delivery overwrote original event: %+v", result.Record)
	}
}

func TestInboxStoresCanonicalEvent(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	record := insertReviewEventTest(t, store, "review", "submitted")
	var canonical NormalizedReviewEvent
	if err := json.Unmarshal([]byte(record.CanonicalEventJSON), &canonical); err != nil {
		t.Fatal(err)
	}
	if canonical.SchemaVersion != normalizedReviewEventSchema || canonical.EventID != record.Event.EventID || canonical.Action != "review.created" {
		t.Fatalf("canonical event mismatch: %+v", canonical)
	}
}

func TestInboxSanitizesSecrets(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	record := insertReviewEventTest(t, store, "pull_request", "opened")
	lower := strings.ToLower(record.SanitizedPayloadJSON)
	if strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") || strings.Contains(lower, "token") || strings.Contains(lower, "alice") {
		t.Fatalf("sanitized payload leaked sensitive data: %s", record.SanitizedPayloadJSON)
	}
}

func TestInboxBoundsSanitizedPayload(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	normalization := normalizeReviewEventTest(t, "pull_request", "opened")
	result, err := store.InsertReviewEventInbox(context.Background(), normalization, ReviewEventIngressMetadata{
		RawPayload: []byte(strings.Repeat("x", reviewEventSanitizedPayloadLimit*2)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Record.SanitizedPayloadJSON) > reviewEventSanitizedPayloadLimit {
		t.Fatalf("sanitized payload is unbounded: %d", len(result.Record.SanitizedPayloadJSON))
	}
}

func TestInboxSurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-inbox-restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	record := insertReviewEventTest(t, store, "pull_request", "opened")
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	reopened, err := store.GetReviewEventInbox(context.Background(), record.Event.EventID)
	if err != nil || reopened.Event.EventID != record.Event.EventID || reopened.CanonicalEventJSON == "" {
		t.Fatalf("reopened event = %+v err=%v", reopened, err)
	}
}

func TestInboxClaimUsesLease(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	insertReviewEventTest(t, store, "pull_request", "opened")
	first, err := store.ClaimReviewEventInbox(context.Background(), ReviewEventClaimOptions{
		LeaseOwner: "worker-a", Now: reviewEventTestTime, LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, secondErr := store.ClaimReviewEventInbox(context.Background(), ReviewEventClaimOptions{
		LeaseOwner: "worker-b", Now: reviewEventTestTime, LeaseDuration: time.Minute,
	})
	if first.LeaseOwner != "worker-a" || !errors.Is(secondErr, sql.ErrNoRows) {
		t.Fatalf("lease result = first:%+v second:%v", first, secondErr)
	}
}

func TestInboxProcessorRoutesOnce(t *testing.T) {
	store, queue, processor, record := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("first process = %v, %v", processed, err)
	}
	if _, err := store.db.Exec(`UPDATE review_event_inbox SET status='normalized', processed_at='' WHERE event_id=?`, record.Event.EventID); err != nil {
		t.Fatal(err)
	}
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("second process = %v, %v", processed, err)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("event routed %d times", len(queue.jobs))
	}
}

func TestInboxNoSubscriptionCompletesWithoutRoute(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	record := insertReviewEventTest(t, store, "pull_request", "opened")
	queue := &recordingReviewEventQueue{}
	processor := &ReviewEventInboxProcessor{Store: store, Queue: queue, LeaseOwner: "worker-a", Now: func() time.Time { return reviewEventTestTime }}
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("process = %v, %v", processed, err)
	}
	completed, _ := store.GetReviewEventInbox(context.Background(), record.Event.EventID)
	if completed.Status != "processed" || completed.ReasonCode != "no_active_subscription" || len(queue.jobs) != 0 {
		t.Fatalf("no-subscription result = %+v jobs=%d", completed, len(queue.jobs))
	}
}

func TestInboxRetryStopsAfterMaxAttempts(t *testing.T) {
	store, queue, processor, record := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	queue.fail = true
	for attempt := 1; attempt <= 3; attempt++ {
		_, err := processor.ProcessOne(context.Background())
		if err == nil {
			t.Fatalf("attempt %d unexpectedly succeeded", attempt)
		}
		if _, err := store.db.Exec(`UPDATE review_event_inbox SET next_attempt_at='' WHERE event_id=?`, record.Event.EventID); err != nil {
			t.Fatal(err)
		}
	}
	failed, _ := store.GetReviewEventInbox(context.Background(), record.Event.EventID)
	if failed.Status != "failed" || failed.AttemptCount != 3 {
		t.Fatalf("retry terminal state = %+v", failed)
	}
}

type recordingReviewEventQueue struct {
	jobs []ReviewGatewayJob
	fail bool
}

func (q *recordingReviewEventQueue) EnqueuePreparedJob(_ context.Context, job ReviewGatewayJob) (bool, error) {
	if q.fail {
		return false, errors.New("temporary queue failure")
	}
	for _, existing := range q.jobs {
		if existing.DedupeKey == job.DedupeKey {
			return false, nil
		}
	}
	q.jobs = append(q.jobs, job)
	return true, nil
}

func newReviewEventProcessorHarness(
	t *testing.T, groups []string, mode, eventName, action string,
) (*SQLiteReviewGatewayStore, *recordingReviewEventQueue, *ReviewEventInboxProcessor, ReviewEventInboxRecord) {
	t.Helper()
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: groups, NotificationMode: mode,
	}, reviewEventTestTime)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	record := insertReviewEventTest(t, store, eventName, action)
	queue := &recordingReviewEventQueue{}
	processor := &ReviewEventInboxProcessor{
		Store: store, Queue: queue, LeaseOwner: "worker-a",
		LeaseDuration: time.Minute, Now: func() time.Time { return reviewEventTestTime },
	}
	return store, queue, processor, record
}
