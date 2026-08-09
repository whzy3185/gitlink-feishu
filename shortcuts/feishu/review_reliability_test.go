package feishu

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNormalizeGitLinkReviewEvent(t *testing.T) {
	event, err := NormalizeGitLinkReviewEvent("delivery-1", "pull_request", []byte(`{
		"repository":{"full_name":"owner/repo"},
		"pull_request":{"number":42,"head_sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	}`))
	if err != nil {
		t.Fatalf("NormalizeGitLinkReviewEvent: %v", err)
	}
	if event.Repository != "owner/repo" || event.PRNumber != 42 || event.HeadSHA == "" {
		t.Fatalf("event = %#v", event)
	}
}

func TestReviewEventInboxDeduplicatesRetriesDeadLettersAndReplays(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	event := ReviewRepositoryEvent{SchemaVersion: reviewRepositoryEventSchema, DeliveryID: "delivery-1", EventType: "pull_request", Repository: "owner/repo", PRNumber: 42}
	inserted, err := store.EnqueueReviewEvent(context.Background(), event, now)
	if err != nil || !inserted {
		t.Fatalf("first enqueue = %v, %v", inserted, err)
	}
	inserted, err = store.EnqueueReviewEvent(context.Background(), event, now)
	if err != nil || inserted {
		t.Fatalf("duplicate enqueue = %v, %v", inserted, err)
	}
	claimed, err := store.ClaimReviewEvents(context.Background(), "worker-a", now, now.Add(time.Minute), 1)
	if err != nil || len(claimed) != 1 || claimed[0].AttemptCount != 1 {
		t.Fatalf("claimed = %#v, err=%v", claimed, err)
	}
	status, err := store.FailReviewEvent(context.Background(), claimed[0], "worker-a", errors.New("temporary"), now, 1)
	if err != nil || status != "dead_letter" {
		t.Fatalf("failure = %q, %v", status, err)
	}
	var deadLetters int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_dead_letters WHERE source_id='delivery-1'`).Scan(&deadLetters); err != nil || deadLetters != 1 {
		t.Fatalf("dead letters = %d, err=%v", deadLetters, err)
	}
	if err := store.ReplayReviewEvent(context.Background(), "delivery-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("ReplayReviewEvent: %v", err)
	}
	claimed, err = store.ClaimReviewEvents(context.Background(), "worker-b", now.Add(time.Minute), now.Add(2*time.Minute), 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("replayed claim = %#v, err=%v", claimed, err)
	}
}

func TestReviewOperationDedupeLeaseAndCompletion(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	operation, err := NewReviewOperation("refresh_pr", "owner/repo#42", map[string]interface{}{"pr": 42}, 3, now)
	if err != nil {
		t.Fatalf("NewReviewOperation: %v", err)
	}
	inserted, err := store.EnqueueReviewOperation(context.Background(), operation, now)
	if err != nil || !inserted {
		t.Fatalf("enqueue = %v, %v", inserted, err)
	}
	inserted, err = store.EnqueueReviewOperation(context.Background(), operation, now)
	if err != nil || inserted {
		t.Fatalf("duplicate enqueue = %v, %v", inserted, err)
	}

	type claimResult struct {
		items []ReviewOperation
		err   error
	}
	results := make(chan claimResult, 2)
	var wg sync.WaitGroup
	for _, owner := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			items, err := store.ClaimReviewOperations(context.Background(), owner, now, now.Add(time.Minute), 1)
			results <- claimResult{items: items, err: err}
		}(owner)
	}
	wg.Wait()
	close(results)
	claimed := []ReviewOperation{}
	for result := range results {
		if result.err != nil {
			t.Fatalf("claim error: %v", result.err)
		}
		claimed = append(claimed, result.items...)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed count = %d, want 1", len(claimed))
	}
	if err := store.CompleteReviewOperation(context.Background(), claimed[0], claimed[0].LeaseOwner, now.Add(time.Second)); err != nil {
		t.Fatalf("CompleteReviewOperation: %v", err)
	}
}

func TestReviewOperationUnknownRequiresReconciliationAndNoBlindRetry(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	operation, _ := NewReviewOperation("remote_write", "owner/repo#42:write", map[string]interface{}{"pr": 42}, 3, now)
	if _, err := store.EnqueueReviewOperation(context.Background(), operation, now); err != nil {
		t.Fatalf("EnqueueReviewOperation: %v", err)
	}
	claimed, err := store.ClaimReviewOperations(context.Background(), "worker-a", now, now.Add(time.Minute), 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed = %#v, err=%v", claimed, err)
	}
	status, err := store.FailReviewOperation(context.Background(), claimed[0], "worker-a", &ReviewOperationError{
		Kind: "network", RemoteMayHaveApplied: true, Err: errors.New("connection reset"),
	}, now.Add(time.Second))
	if err != nil || status != "unknown" {
		t.Fatalf("unknown = %q, %v", status, err)
	}
	again, err := store.ClaimReviewOperations(context.Background(), "worker-b", now.Add(2*time.Minute), now.Add(3*time.Minute), 1)
	if err != nil || len(again) != 0 {
		t.Fatalf("unknown operation was retried: %#v, err=%v", again, err)
	}
	if err := store.MarkReviewOperationReconciled(context.Background(), operation.OperationID, "verified", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("MarkReviewOperationReconciled: %v", err)
	}
	var state, mutation, reconciliation string
	if err := store.db.QueryRow(`SELECT status, mutation_status, reconciliation_state FROM review_operations WHERE operation_id=?`, operation.OperationID).Scan(&state, &mutation, &reconciliation); err != nil {
		t.Fatalf("read operation: %v", err)
	}
	if state != "completed" || mutation != "confirmed" || reconciliation != "verified" {
		t.Fatalf("operation state = %s/%s/%s", state, mutation, reconciliation)
	}
}
