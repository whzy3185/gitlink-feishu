package feishu

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func seedUnknownReviewOperation(t *testing.T, store *SQLiteReviewGatewayStore, resourceType string, desired interface{}) ReviewOperation {
	t.Helper()
	kind := ReviewOperationBitableUpsert
	if resourceType == ReviewResourceTask {
		kind = ReviewOperationTaskUpsert
	}
	op, err := NewReviewOperation(kind, reviewOperationTestJob(), "resource-key", resourceType, desired, ReviewRetryReconcilable, reviewOperationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	op.Status = ReviewOperationUnknown
	op.MutationStatus = ReviewMutationRemoteUnknown
	op.RequiresReconciliation = true
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`UPDATE review_operations SET status='unknown',mutation_status='remote_unknown',requires_reconciliation=1,error_code='request_timeout' WHERE operation_id=?`, op.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := store.GetReviewOperation(context.Background(), op.OperationID)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := createReviewOperationReconciliationTx(context.Background(), tx, updated, "request_timeout", reviewOperationTestTime); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return updated
}

func TestUnknownOperationCreatesReconciliationTask(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["task_create"] = context.DeadlineExceeded
	op, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Review"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, op)
	tasks, err := store.ListReviewOperationReconciliations(context.Background(), "pending", 10)
	if err != nil || len(tasks) != 1 || tasks[0].OperationID != op.OperationID {
		t.Fatalf("tasks=%#v err=%v", tasks, err)
	}
}

func TestExhaustedTransientOperationCreatesDeadLetter(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 503, Detail: "unavailable"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	current := reviewOperationTestTime.Add(time.Minute)
	worker := NewReviewOperationWorker(store, handler, "resource", "worker-exhausted")
	worker.Now = func() time.Time { return current }
	for i := 0; i < 3; i++ {
		ran, err := worker.RunOnce(context.Background())
		if err != nil || !ran {
			t.Fatalf("attempt %d run=%t err=%v", i+1, ran, err)
		}
		current = current.Add(time.Minute)
	}
	stored, _ := store.GetReviewOperation(context.Background(), op.OperationID)
	letters, _ := store.ListReviewDeadLetters(context.Background(), "open", "operation", 10)
	if stored.Status != ReviewOperationDeadLetter || len(letters) != 1 {
		t.Fatalf("status=%s letters=%#v", stored.Status, letters)
	}
}

func TestDeadLetterIsIdempotent(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, op)
	stored, _ := store.GetReviewOperation(context.Background(), op.OperationID)
	tx, _ := store.db.Begin()
	_ = createReviewOperationDeadLetterTx(context.Background(), tx, stored, reviewOperationTestTime.Add(time.Minute))
	_ = tx.Commit()
	letters, _ := store.ListReviewDeadLetters(context.Background(), "open", "operation", 10)
	if len(letters) != 1 {
		t.Fatalf("letters=%d", len(letters))
	}
}

func TestDeadLetterRetryRequiresExplicitConfirmation(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, op)
	letters, _ := store.ListReviewDeadLetters(context.Background(), "open", "operation", 10)
	if err := store.RetryReviewDeadLetter(context.Background(), letters[0].DeadLetterID, "", false, "operator", reviewOperationTestTime); err == nil {
		t.Fatal("retry did not require confirmation")
	}
}

func TestBaseReconciliationRepairsRemoteID(t *testing.T) {
	store, client, _, _ := newReviewOperationResourceHarness(t)
	op := seedUnknownReviewOperation(t, store, ReviewResourceBitable, reviewBitableDesired{ResourceKey: "resource-key", TargetScope: "chat"})
	client.search = BitableSearchResult{Found: true, Matches: 1, RecordID: "record-existing"}
	reconciler := &ReviewOperationReconciler{Store: store, Client: client, TokenProvider: NewReviewTenantTokenProvider(client, func() time.Time { return reviewOperationTestTime }), Config: ReviewCollaborationPublisherConfig{AppID: "app", AppSecret: "secret", BaseAppToken: "base", ReviewTableID: "table"}, LeaseOwner: "reconciler", Now: func() time.Time { return reviewOperationTestTime.Add(time.Minute) }}
	ran, err := reconciler.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	stored, _ := store.GetReviewOperation(context.Background(), op.OperationID)
	state, _ := store.GetReviewResourceState(context.Background(), "resource-key", ReviewResourceBitable)
	if stored.Status != ReviewOperationSucceeded || state.RemoteID != "record-existing" {
		t.Fatalf("operation=%#v state=%#v", stored, state)
	}
}

func TestBaseUnknownCanReconcileByUniqueKey(t *testing.T) { TestBaseReconciliationRepairsRemoteID(t) }

func TestReconciliationFailureCanReachDeadLetter(t *testing.T) {
	store, client, _, _ := newReviewOperationResourceHarness(t)
	seedUnknownReviewOperation(t, store, ReviewResourceBitable, reviewBitableDesired{ResourceKey: "resource-key", TargetScope: "chat"})
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 503, Detail: "unavailable"}
	current := reviewOperationTestTime.Add(time.Minute)
	reconciler := &ReviewOperationReconciler{Store: store, Client: client, TokenProvider: NewReviewTenantTokenProvider(client, func() time.Time { return current }), Config: ReviewCollaborationPublisherConfig{AppID: "app", AppSecret: "secret", BaseAppToken: "base", ReviewTableID: "table"}, LeaseOwner: "reconciler", Now: func() time.Time { return current }}
	for i := 0; i < 3; i++ {
		ran, err := reconciler.RunOnce(context.Background())
		if err != nil || !ran {
			t.Fatalf("attempt %d run=%t err=%v", i+1, ran, err)
		}
		current = current.Add(time.Minute)
	}
	letters, _ := store.ListReviewDeadLetters(context.Background(), "open", "reconciliation", 10)
	if len(letters) != 1 {
		t.Fatalf("letters=%#v", letters)
	}
}

func TestUnsupportedAutoReconciliationBecomesManualRequired(t *testing.T) {
	store, client, _, _ := newReviewOperationResourceHarness(t)
	seedUnknownReviewOperation(t, store, ReviewResourceTask, reviewTaskDesired{ResourceKey: "resource-key", TargetScope: "chat"})
	reconciler := &ReviewOperationReconciler{Store: store, Client: client, LeaseOwner: "reconciler", Now: func() time.Time { return reviewOperationTestTime.Add(time.Minute) }}
	ran, err := reconciler.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	tasks, _ := store.ListReviewOperationReconciliations(context.Background(), "manual_required", 10)
	if len(tasks) != 1 {
		t.Fatalf("tasks=%#v", tasks)
	}
}

func TestResolvedReconciliationDoesNotRunAgain(t *testing.T) {
	store, client, _, _ := newReviewOperationResourceHarness(t)
	seedUnknownReviewOperation(t, store, ReviewResourceTask, reviewTaskDesired{ResourceKey: "resource-key", TargetScope: "chat"})
	reconciler := &ReviewOperationReconciler{Store: store, Client: client, LeaseOwner: "reconciler", Now: func() time.Time { return reviewOperationTestTime.Add(time.Minute) }}
	_, _ = reconciler.RunOnce(context.Background())
	ran, err := reconciler.RunOnce(context.Background())
	if err != nil || ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
}

func TestOperationAdminViewContainsNoRawRemoteIdentifiers(t *testing.T) {
	views := reviewOperationAdminViews([]ReviewOperation{{OperationID: "op", RemoteID: "remote-secret", ChatID: "chat-secret"}})
	encoded := fmt.Sprintf("%#v", views)
	if strings.Contains(encoded, "remote-secret") || strings.Contains(encoded, "chat-secret") {
		t.Fatalf("admin view leaked identifiers: %s", encoded)
	}
}
