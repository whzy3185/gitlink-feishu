package feishu

import (
	"context"
	"testing"
	"time"
)

func newReviewReplayHarness(t *testing.T) (*SQLiteReviewGatewayStore, ReviewEventInboxRecord) {
	t.Helper()
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	original := insertReviewEventTest(t, store, "pull_request", "opened")
	return store, original
}

func replayReviewEventTest(t *testing.T, store *SQLiteReviewGatewayStore, eventID, requestID string) ReviewEventInboxInsertResult {
	t.Helper()
	result, err := store.ReplayReviewEvent(context.Background(), ReviewEventReplayRequest{
		EventID: eventID, RequestID: requestID, Reason: "operator retry after configuration fix", Confirmed: true,
	}, reviewEventTestTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("ReplayReviewEvent: %v", err)
	}
	return result
}

func TestReplayRequiresExplicitRequestID(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	_, err := store.ReplayReviewEvent(context.Background(), ReviewEventReplayRequest{
		EventID: original.Event.EventID, Reason: "retry", Confirmed: true,
	}, reviewEventTestTime)
	if err == nil {
		t.Fatal("Replay without request ID was accepted")
	}
}

func TestReplayRequestIDIsIdempotent(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	first := replayReviewEventTest(t, store, original.Event.EventID, "replay-001")
	second := replayReviewEventTest(t, store, original.Event.EventID, "replay-001")
	if first.Duplicate || !second.Duplicate || first.Record.Event.EventID != second.Record.Event.EventID {
		t.Fatalf("Replay request idempotency failed: first=%+v second=%+v", first, second)
	}
}

func TestReplayPreservesOriginalEvent(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	replayReviewEventTest(t, store, original.Event.EventID, "replay-preserve")
	after, err := store.GetReviewEventInbox(context.Background(), original.Event.EventID)
	if err != nil || after.CanonicalEventJSON != original.CanonicalEventJSON || after.Status != original.Status {
		t.Fatalf("original changed after Replay: before=%+v after=%+v err=%v", original, after, err)
	}
}

func TestReplayUsesCurrentSubscriptions(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	_, _ = store.db.Exec(`UPDATE review_event_inbox SET status='processed' WHERE event_id=?`, original.Event.EventID)
	if _, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupPulls},
	}, reviewEventTestTime.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	replay := replayReviewEventTest(t, store, original.Event.EventID, "replay-current-subscription")
	queue := &recordingReviewEventQueue{}
	processor := &ReviewEventInboxProcessor{Store: store, Queue: queue, LeaseOwner: "replay-worker", Now: func() time.Time { return reviewEventTestTime.Add(2 * time.Hour) }}
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("process Replay = %v, %v", processed, err)
	}
	routes, _ := store.ListReviewEventRoutes(context.Background(), replay.Record.Event.EventID)
	if len(routes) != 1 || len(queue.jobs) != 1 {
		t.Fatalf("Replay did not use current subscription: routes=%+v jobs=%+v", routes, queue.jobs)
	}
}

func TestReplayCannotBypassDisabledInstallation(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	_, _ = store.db.Exec(`UPDATE gitlink_installations SET enabled=0 WHERE installation_id='installation-a'`)
	_, err := store.ReplayReviewEvent(context.Background(), ReviewEventReplayRequest{
		EventID: original.Event.EventID, RequestID: "replay-disabled-installation", Reason: "retry", Confirmed: true,
	}, reviewEventTestTime)
	if err == nil {
		t.Fatal("Replay bypassed disabled installation")
	}
}

func TestReplayCannotBypassRepositoryBinding(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	_, _ = store.db.Exec(`UPDATE chat_repository_bindings SET enabled=0 WHERE installation_id='installation-a'`)
	_, err := store.ReplayReviewEvent(context.Background(), ReviewEventReplayRequest{
		EventID: original.Event.EventID, RequestID: "replay-disabled-binding", Reason: "retry", Confirmed: true,
	}, reviewEventTestTime)
	if err == nil {
		t.Fatal("Replay bypassed disabled repository binding")
	}
}

func TestReplayDoesNotReverifySignature(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	replay := replayReviewEventTest(t, store, original.Event.EventID, "replay-no-signature")
	if replay.Record.SignatureStatus != "replay_not_reverified" || replay.Record.TimestampStatus != "replay_not_reverified" {
		t.Fatalf("Replay signature metadata = %+v", replay.Record)
	}
}

func TestReplayCreatesNoDirectExternalWrites(t *testing.T) {
	store, original := newReviewReplayHarness(t)
	replayReviewEventTest(t, store, original.Event.EventID, "replay-no-writes")
	var jobs, routes int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_jobs`).Scan(&jobs)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_event_routes`).Scan(&routes)
	if jobs != 0 || routes != 0 {
		t.Fatalf("Replay created direct effects: jobs=%d routes=%d", jobs, routes)
	}
}

func TestUnsupportedIgnoredEventCannotReplay(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	ignored := normalizeReviewEventTest(t, "pull_request", "teleported")
	insert, err := store.InsertReviewEventInbox(context.Background(), ignored, ReviewEventIngressMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ReplayReviewEvent(context.Background(), ReviewEventReplayRequest{
		EventID: insert.Record.Event.EventID, RequestID: "replay-ignored", Reason: "retry", Confirmed: true,
	}, reviewEventTestTime)
	if err == nil {
		t.Fatal("ignored unsupported event was replayed")
	}
}
