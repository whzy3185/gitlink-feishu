package feishu

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var reviewOperationTestTime = time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)

func openReviewOperationTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "operation.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func reviewOperationTestJob() ReviewGatewayJob {
	return ReviewGatewayJob{
		JobID:           "job-operation",
		InstallationID:  "installation-a",
		ChatID:          "chat-a",
		Repository:      "Gitlink/gitlink-cli",
		PRNumber:        431,
		RequestedBy:     "user-a",
		SourceMessageID: "message-a",
		MaxAttempts:     3,
	}
}

func reviewOperationTestResult() ReviewGatewayExecutionResult {
	return ReviewGatewayExecutionResult{
		Status:      "completed",
		CompletedAt: reviewGatewayTimestamp(reviewOperationTestTime),
		ResultCard:  Card{"header": map[string]interface{}{"title": "PR #431"}},
		Message:     "review ready",
	}
}

func TestOperationPlannerCreatesDurableOperations(t *testing.T) {
	store := openReviewOperationTestStore(t)
	planner := ReviewOperationPlanner{Store: store, Now: func() time.Time { return reviewOperationTestTime }}
	operations, err := planner.Plan(context.Background(), reviewOperationTestJob(), reviewOperationTestResult())
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 2 {
		t.Fatalf("operations=%d, want card and reply", len(operations))
	}
	stored, err := store.ListReviewOperations(context.Background(), "", "", 10)
	if err != nil || len(stored) != 2 {
		t.Fatalf("stored=%d err=%v", len(stored), err)
	}
	if stored[1].DependencyOperationID != stored[0].OperationID || stored[1].DependencyPolicy != ReviewDependencySuccess {
		t.Fatalf("reply dependency=%#v, want card success", stored[1])
	}
}

func TestOperationPlannerIsIdempotent(t *testing.T) {
	store := openReviewOperationTestStore(t)
	planner := ReviewOperationPlanner{Store: store, Now: func() time.Time { return reviewOperationTestTime }}
	for i := 0; i < 2; i++ {
		if _, err := planner.Plan(context.Background(), reviewOperationTestJob(), reviewOperationTestResult()); err != nil {
			t.Fatal(err)
		}
	}
	stored, _ := store.ListReviewOperations(context.Background(), "", "", 10)
	if len(stored) != 2 {
		t.Fatalf("stored=%d, want 2", len(stored))
	}
}

func TestOperationPlannerPerformsNoExternalWrites(t *testing.T) {
	store := openReviewOperationTestStore(t)
	// The planner has no OpenAPI or GitLink client by construction. Successful
	// planning therefore proves the phase boundary without a fake that could
	// accidentally hide a network call.
	planner := ReviewOperationPlanner{Store: store, Now: func() time.Time { return reviewOperationTestTime }}
	if _, err := planner.Plan(context.Background(), reviewOperationTestJob(), reviewOperationTestResult()); err != nil {
		t.Fatal(err)
	}
}

func TestSameDesiredStateCreatesOneOperation(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	op, err := NewReviewOperation(ReviewOperationBitableUpsert, job, "resource-a", ReviewResourceBitable, map[string]string{"value": "same"}, ReviewRetryReconcilable, reviewOperationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{op, op}); err != nil {
		t.Fatal(err)
	}
	stored, _ := store.ListReviewOperations(context.Background(), "", "", 10)
	if len(stored) != 1 {
		t.Fatalf("stored=%d, want 1", len(stored))
	}
}

func TestNewerDesiredStateMakesOlderOperationStale(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	oldOperation, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "resource-a", ReviewResourceBitable, map[string]string{"value": "old"}, ReviewRetryReconcilable, reviewOperationTestTime)
	newOperation, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "resource-a", ReviewResourceBitable, map[string]string{"value": "new"}, ReviewRetryReconcilable, reviewOperationTestTime.Add(time.Minute))
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{oldOperation}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{newOperation}); err != nil {
		t.Fatal(err)
	}
	oldStored, _ := store.GetReviewOperation(context.Background(), oldOperation.OperationID)
	if oldStored.Status != ReviewOperationStale {
		t.Fatalf("old status=%s, want stale", oldStored.Status)
	}
}

func TestOperationPayloadIsBounded(t *testing.T) {
	_, err := NewReviewOperation(ReviewOperationReplySend, reviewOperationTestJob(), "consumer-a", "feishu_reply", map[string]string{
		"text": strings.Repeat("x", reviewOperationDesiredJSONMaxBytes),
	}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err == nil || !strings.Contains(err.Error(), "desired_payload_too_large") {
		t.Fatalf("err=%v, want payload too large", err)
	}
}

func TestOperationPayloadContainsNoSecrets(t *testing.T) {
	_, err := NewReviewOperation(ReviewOperationReplySend, reviewOperationTestJob(), "consumer-a", "feishu_reply", map[string]string{
		"authorization": "Bearer secret",
	}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err == nil {
		t.Fatal("secret payload accepted")
	}
}

func TestOperationLeaseSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	op, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, reviewOperationTestJob(), "card-a", ReviewResourceCard, Card{"x": "y"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{op}); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "canonical_card", LeaseOwner: "worker-a", Now: reviewOperationTestTime, LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim=%d err=%v", len(claimed), err)
	}
	_ = store.Close()
	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	persisted, err := reopened.GetReviewOperation(context.Background(), op.OperationID)
	if err != nil || persisted.LeaseOwner != "worker-a" || persisted.Status != ReviewOperationLeased {
		t.Fatalf("persisted=%#v err=%v", persisted, err)
	}
}

func TestStaleOperationWorkerCannotCompleteNewLease(t *testing.T) {
	store := openReviewOperationTestStore(t)
	op, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, reviewOperationTestJob(), "card-a", ReviewResourceCard, Card{"x": "y"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{op})
	first, _ := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "canonical_card", LeaseOwner: "worker-a", Now: reviewOperationTestTime, LeaseDuration: time.Second})
	second, _ := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "canonical_card", LeaseOwner: "worker-b", Now: reviewOperationTestTime.Add(2 * time.Second), LeaseDuration: time.Minute})
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("claims first=%d second=%d", len(first), len(second))
	}
	if _, err := store.StartReviewOperationAttempt(context.Background(), first[0], reviewOperationTestTime.Add(3*time.Second)); err == nil {
		t.Fatal("stale worker started a new attempt")
	}
	if _, err := store.StartReviewOperationAttempt(context.Background(), second[0], reviewOperationTestTime.Add(3*time.Second)); err != nil {
		t.Fatalf("current worker failed: %v", err)
	}
}

func TestOperationDependencyRequiresSuccess(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	parent, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, "card-a", ReviewResourceCard, Card{"x": "y"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	child, _ := NewReviewOperation(ReviewOperationReplySend, job, "consumer-a", "feishu_reply", map[string]string{"text": "done"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	child.DependencyOperationID = parent.OperationID
	child.DependencyPolicy = ReviewDependencySuccess
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{parent, child}); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "reply", LeaseOwner: "reply-worker", Now: reviewOperationTestTime, LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 0 {
		t.Fatalf("child claim=%d err=%v, want blocked", len(claimed), err)
	}
	_, err = store.db.Exec(`UPDATE review_operations SET status='succeeded' WHERE operation_id=?`, parent.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err = store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "reply", LeaseOwner: "reply-worker", Now: reviewOperationTestTime, LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("child claim=%d err=%v, want 1", len(claimed), err)
	}
}
