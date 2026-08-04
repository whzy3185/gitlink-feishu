package feishu

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUnknownReplyIsNotAutomaticallyResent(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	sender.err = context.DeadlineExceeded
	operation, _ := NewReviewOperation(ReviewOperationReplySend, reviewOperationTestJob(), "consumer", "feishu_reply", reviewReplyDesired{Text: "done"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, operation)
	worker := NewReviewOperationWorker(store, handler, "reply", "reply-worker-again")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Hour) }
	ran, err := worker.RunOnce(context.Background())
	if err != nil || ran || sender.sends != 1 {
		t.Fatalf("run=%t sends=%d err=%v", ran, sender.sends, err)
	}
}

func TestReplyWaitsForCanonicalCardSuccess(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	card, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, "card", ReviewResourceCard, Card{"value": "card"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	reply, _ := NewReviewOperation(ReviewOperationReplySend, job, "consumer", "feishu_reply", reviewReplyDesired{Text: "done"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	reply.DependencyOperationID = card.OperationID
	reply.DependencyPolicy = ReviewDependencySuccess
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{card, reply}); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "reply", LeaseOwner: "reply-worker", Now: reviewOperationTestTime, LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 0 {
		t.Fatalf("reply ran before card success: claimed=%d err=%v", len(claimed), err)
	}
	if _, err := store.db.Exec(`UPDATE review_operations SET status='succeeded' WHERE operation_id=?`, card.OperationID); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "reply", LeaseOwner: "reply-worker", Now: reviewOperationTestTime, LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("reply unavailable after card success: claimed=%d err=%v", len(claimed), err)
	}
}

func TestBaseCreateSuccessPersistFailureBecomesUnknown(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	if _, err := store.db.Exec(`CREATE TRIGGER reject_base_state BEFORE INSERT ON review_collaboration_resources
		WHEN NEW.resource_type='feishu_bitable' BEGIN SELECT RAISE(FAIL, 'synthetic local persist failure'); END`); err != nil {
		t.Fatal(err)
	}
	operation, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, operation)
	if stored.Status != ReviewOperationUnknown || stored.RemoteID == "" || client.baseCreateCalls != 1 {
		t.Fatalf("status=%s remote=%q creates=%d", stored.Status, stored.RemoteID, client.baseCreateCalls)
	}
}

func TestBaseUpdateIsIdempotent(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	operation, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat", Fields: map[string]interface{}{"value": "same"}}, ReviewRetryReconcilable, reviewOperationTestTime)
	first := executeReviewOperationForTest(t, store, handler, operation)
	if first.Status != ReviewOperationSucceeded {
		t.Fatalf("first status=%s", first.Status)
	}
	result, err := handler.Execute(context.Background(), operation)
	if err != nil || !result.Unchanged || client.baseCreateCalls != 1 || client.baseUpdateCalls != 0 {
		t.Fatalf("result=%+v creates=%d updates=%d err=%v", result, client.baseCreateCalls, client.baseUpdateCalls, err)
	}
}

func TestDocCreateUnknownDoesNotCreateAgain(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["doc_create"] = context.DeadlineExceeded
	operation, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, reviewOperationTestJob(), "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Title: "Review", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, operation)
	worker := NewReviewOperationWorker(store, handler, "resource", "doc-worker-again")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Hour) }
	ran, err := worker.RunOnce(context.Background())
	if err != nil || ran || client.docCreateCalls != 1 {
		t.Fatalf("run=%t creates=%d err=%v", ran, client.docCreateCalls, err)
	}
}

func TestDocFingerprintPreventsDuplicateSnapshot(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	operation, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, reviewOperationTestJob(), "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Title: "Review", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, operation)
	result, err := handler.Execute(context.Background(), operation)
	if err != nil || !result.Unchanged || client.docCreateCalls != 1 || client.docAppendCalls != 1 {
		t.Fatalf("result=%+v creates=%d appends=%d err=%v", result, client.docCreateCalls, client.docAppendCalls, err)
	}
}

func TestTaskCreateIsNotBlindlyRetried(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["task_create"] = context.DeadlineExceeded
	operation, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Review"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, operation)
	worker := NewReviewOperationWorker(store, handler, "resource", "task-worker-again")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Hour) }
	ran, err := worker.RunOnce(context.Background())
	if err != nil || ran || client.taskCreateCalls != 1 {
		t.Fatalf("run=%t creates=%d err=%v", ran, client.taskCreateCalls, err)
	}
}

func TestTaskPatchTransientCanRetry(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	if err := store.SaveReviewResourceState(context.Background(), "task-key", ReviewResourceTask, "task-existing", "old", reviewOperationTestTime); err != nil {
		t.Fatal(err)
	}
	client.err["task_patch"] = &OpenAPIHTTPError{StatusCode: 503, Detail: "unavailable"}
	operation, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Changed"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, operation)
	if stored.Status != ReviewOperationRetryScheduled || client.taskPatchCalls != 1 {
		t.Fatalf("status=%s patches=%d", stored.Status, client.taskPatchCalls)
	}
}

func TestArchivedTaskIsCompleted(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	if err := store.SaveReviewResourceState(context.Background(), "task-key", ReviewResourceTask, "task-existing", "old", reviewOperationTestTime); err != nil {
		t.Fatal(err)
	}
	operation, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Archived: true}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, operation)
	if stored.Status != ReviewOperationSucceeded || client.taskPatchCalls != 1 {
		t.Fatalf("status=%s patches=%d", stored.Status, client.taskPatchCalls)
	}
}

func TestTaskFailureDoesNotBlockCard(t *testing.T) {
	store, client, sender, handler := newReviewOperationResourceHarness(t)
	client.err["task_create"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	job := reviewOperationTestJob()
	task, _ := NewReviewOperation(ReviewOperationTaskUpsert, job, "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Review"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	card, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, "card-key", ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"elements": []interface{}{}}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{task, card}); err != nil {
		t.Fatal(err)
	}
	resourceWorker := NewReviewOperationWorker(store, handler, "resource", "resource-worker")
	cardWorker := NewReviewOperationWorker(store, handler, "canonical_card", "card-worker")
	resourceWorker.Now, cardWorker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Minute) }, func() time.Time { return reviewOperationTestTime.Add(time.Minute) }
	_, _ = resourceWorker.RunOnce(context.Background())
	_, _ = cardWorker.RunOnce(context.Background())
	stored, _ := store.GetReviewOperation(context.Background(), card.OperationID)
	if stored.Status != ReviewOperationSucceeded || sender.sends != 1 {
		t.Fatalf("card status=%s sends=%d", stored.Status, sender.sends)
	}
}

func assertOperationHTTPStatusIsTerminal(t *testing.T, status int) {
	t.Helper()
	classified := ClassifyReviewOperationError(&OpenAPIHTTPError{StatusCode: status}, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, true)
	if classified.Class != ReviewOperationErrorTerminal || classified.HTTPStatus != status {
		t.Fatalf("status=%d classified=%+v", status, classified)
	}
}

func TestOperation400IsTerminal(t *testing.T) { assertOperationHTTPStatusIsTerminal(t, 400) }
func TestOperation401IsTerminal(t *testing.T) { assertOperationHTTPStatusIsTerminal(t, 401) }
func TestOperation404IsTerminal(t *testing.T) { assertOperationHTTPStatusIsTerminal(t, 404) }

func TestOperationRetryAfterIsRespected(t *testing.T) {
	classified := ClassifyReviewOperationError(&OpenAPIHTTPError{StatusCode: 429, RetryAfter: 17 * time.Second}, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, true)
	if classified.Class != ReviewOperationErrorRateLimited || classified.RetryAfter != 17*time.Second {
		t.Fatalf("classified=%+v", classified)
	}
}

func TestStaleFingerprintSkipsRemoteWrite(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	oldOperation, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, map[string]string{"value": "old"}, ReviewRetryReconcilable, reviewOperationTestTime)
	newOperation, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, map[string]string{"value": "new"}, ReviewRetryReconcilable, reviewOperationTestTime.Add(time.Minute))
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{oldOperation})
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{newOperation})
	if _, err := store.db.Exec(`UPDATE review_operations SET status='succeeded' WHERE operation_id=?`, newOperation.OperationID); err != nil {
		t.Fatal(err)
	}
	worker := NewReviewOperationWorker(store, handler, "resource", "resource-worker")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(2 * time.Minute) }
	ran, err := worker.RunOnce(context.Background())
	if err != nil || ran || client.searchCalls+client.baseCreateCalls+client.baseUpdateCalls != 0 {
		t.Fatalf("run=%t external calls=%d err=%v", ran, client.searchCalls+client.baseCreateCalls+client.baseUpdateCalls, err)
	}
}

func TestRetryStopsAtMaxAttempts(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 503, Detail: "unavailable"}
	operation, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	operation.MaxAttempts = 1
	stored := executeReviewOperationForTest(t, store, handler, operation)
	if stored.Status != ReviewOperationDeadLetter || stored.AttemptCount != 1 {
		t.Fatalf("status=%s attempts=%d", stored.Status, stored.AttemptCount)
	}
}

func TestUnknownResourceShowsNeedsReconciliation(t *testing.T) {
	card := reviewCardWithProjectionStatuses(Card{"elements": []interface{}{}}, []ReviewResourceProjectionStatus{{ResourceType: ReviewResourceBitable, Status: ReviewProjectionNeedsReconciliation}})
	encoded, err := encodeReviewGatewayCard(card)
	if err != nil || !strings.Contains(encoded, "Base：待对账") {
		t.Fatalf("card=%s err=%v", encoded, err)
	}
}

func TestWorkerLeaseRecoversAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worker-restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	operation, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, reviewOperationTestJob(), "card", ReviewResourceCard, Card{"value": "card"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{operation})
	claimed, err := store.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "canonical_card", LeaseOwner: "worker-a", Now: reviewOperationTestTime, LeaseDuration: time.Second})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("first claim=%d err=%v", len(claimed), err)
	}
	_ = store.Close()
	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	claimed, err = reopened.ClaimReviewOperations(context.Background(), ReviewOperationClaimOptions{QueueClass: "canonical_card", LeaseOwner: "worker-b", Now: reviewOperationTestTime.Add(2 * time.Second), LeaseDuration: time.Minute})
	if err != nil || len(claimed) != 1 || claimed[0].LeaseOwner != "worker-b" {
		t.Fatalf("recovered claim=%+v err=%v", claimed, err)
	}
}

type coalescedRouteQueue struct{ jobID string }

func (q coalescedRouteQueue) EnqueuePreparedJob(context.Context, ReviewGatewayJob) (bool, error) {
	return false, nil
}
func (q coalescedRouteQueue) EnqueuePreparedJobWithAdmission(context.Context, ReviewGatewayJob, string) (ReviewJobAdmissionResult, error) {
	return ReviewJobAdmissionResult{Coalesced: true, JobID: q.jobID}, nil
}

func TestCoalescedEventRouteUsesExistingJobID(t *testing.T) {
	store, _, processor, record := newReviewEventProcessorHarness(t, []string{ReviewSubscriptionGroupPulls}, ReviewNotificationCanonicalOnly, "pull_request", "opened")
	processor.Queue = coalescedRouteQueue{jobID: "job-existing"}
	if processed, err := processor.ProcessOne(context.Background()); err != nil || !processed {
		t.Fatalf("processed=%t err=%v", processed, err)
	}
	routes, err := store.ListReviewEventRoutes(context.Background(), record.Event.EventID)
	if err != nil || len(routes) != 1 || routes[0].RouteStatus != "coalesced" || routes[0].JobID != "job-existing" {
		t.Fatalf("routes=%+v err=%v", routes, err)
	}
}

func requireReviewStageFourMigration(t *testing.T, version int, name string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stage-four-migration.db")
	for pass := 0; pass < 2; pass++ {
		store, err := OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("open pass %d: %v", pass+1, err)
		}
		var count int
		err = store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name=?`, version, name).Scan(&count)
		_ = store.Close()
		if err != nil || count != 1 {
			t.Fatalf("migration %d/%s count=%d pass=%d err=%v", version, name, count, pass+1, err)
		}
	}
}

func TestOperationMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFourMigration(t, 11, "review_operations_v1")
}
func TestAttemptMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFourMigration(t, 12, "review_operation_attempts_projection_status_v1")
}
func TestDeadLetterMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFourMigration(t, 13, "review_dead_letters_operation_reconciliation_v1")
}
func TestQueueClassMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFourMigration(t, 14, "review_job_queue_classes_coalescing_v1")
}
func TestRateLimitMigrationIsIdempotent(t *testing.T) {
	requireReviewStageFourMigration(t, 15, "review_rate_limit_buckets_v1")
}

func TestStageFourMigrationRollbackLeavesDatabaseUsable(t *testing.T) {
	store := openReviewOperationTestStore(t)
	err := applyReviewGatewaySchemaMigrations(store.db, []reviewGatewaySchemaMigration{{Version: 199, Name: "synthetic_failure", Statements: []string{`CREATE TABLE leaked_stage_four(value TEXT)`, `INSERT INTO missing_stage_four(value) VALUES('x')`}}})
	if err == nil {
		t.Fatal("invalid migration unexpectedly succeeded")
	}
	var versionCount, tableCount, operationCount int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=199`).Scan(&versionCount)
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='leaked_stage_four'`).Scan(&tableCount)
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_operations`).Scan(&operationCount); err != nil {
		t.Fatalf("database unusable after rollback: %v", err)
	}
	if versionCount != 0 || tableCount != 0 || operationCount != 0 {
		t.Fatalf("failed migration leaked state: version=%d table=%d operations=%d", versionCount, tableCount, operationCount)
	}
}
