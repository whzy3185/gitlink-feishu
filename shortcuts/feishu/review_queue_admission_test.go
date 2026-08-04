package feishu

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func reviewAdmissionJob(id, action, chat, user string, pr int, now time.Time) ReviewGatewayJob {
	return ReviewGatewayJob{SchemaVersion: reviewGatewayJobSchema, JobID: "job-" + id, DedupeKey: "dedupe-" + id, Status: "queued", Action: action, InstallationID: "installation-a", Repository: "owner/repo", PRNumber: pr, ChatID: chat, RequestedBy: user, SourceEventID: "event-" + id, CreatedAt: reviewGatewayTimestamp(now), MaxAttempts: 3, NextAttemptAt: reviewGatewayTimestamp(now)}
}

func assertAdmissionCode(t *testing.T, err error, code string) {
	t.Helper()
	var admission *ReviewQueueAdmissionError
	if !errors.As(err, &admission) || admission.Code != code {
		t.Fatalf("err=%v code=%v, want %s", err, admission, code)
	}
}

func TestGlobalJobCapacityRejectsExcessWork(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.MaxJobs = 1
	first := reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime)
	second := reviewAdmissionJob("two", "read_review_context", "chat-b", "user-b", 2, reviewOperationTestTime)
	if result, err := store.AdmitReviewGatewayJob(context.Background(), first, "event_route", reviewOperationTestTime, limits); err != nil || !result.Created {
		t.Fatal(result, err)
	}
	_, err := store.AdmitReviewGatewayJob(context.Background(), second, "event_route", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "global_job_capacity")
}

func TestGlobalOperationCapacityRejectsExcessWork(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.MaxOperations = 1
	operation, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "resource", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "resource", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{operation})
	_, err := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "global_operation_capacity")
}

func TestPerChatCapacityRejectsExcessWork(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.MaxJobsPerChat = 1
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	_, err := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "read_review_context", "chat-a", "user-b", 2, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "chat_job_capacity")
}

func TestPerUserCapacityRejectsExcessWork(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.MaxJobsPerUser = 1
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	_, err := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "read_review_context", "chat-b", "user-a", 2, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "user_job_capacity")
}

func TestCapacityFailureLeavesNoOrphanReservation(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.MaxJobs = 1
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	second := reviewAdmissionJob("two", "read_review_context", "chat-b", "user-b", 2, reviewOperationTestTime)
	_, _ = store.AdmitReviewGatewayJob(context.Background(), second, "event_route", reviewOperationTestTime, limits)
	var count int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_events WHERE dedupe_key=?`, second.DedupeKey).Scan(&count)
	if count != 0 {
		t.Fatalf("orphan reservations=%d", count)
	}
}

func TestUserRateLimitRejectsExcessCommands(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.UserRequestsPerMinute = 1
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	_, err := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "read_review_context", "chat-b", "user-a", 2, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "user_rate_limited")
}

func TestChatRateLimitRejectsExcessCommands(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.ChatRequestsPerMinute = 1
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	_, err := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "read_review_context", "chat-a", "user-b", 2, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	assertAdmissionCode(t, err, "chat_rate_limited")
}

func TestWebhookIsNotSubjectToUserRateLimit(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.UserRequestsPerMinute = 1
	for i := 0; i < 2; i++ {
		job := reviewAdmissionJob(fmt.Sprint(i), "read_review_context", "chat-a", "webhook", i+1, reviewOperationTestTime)
		if _, err := store.AdmitReviewGatewayJob(context.Background(), job, "event_route", reviewOperationTestTime, limits); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWebhookStillRespectsGlobalCapacity(t *testing.T) { TestGlobalJobCapacityRejectsExcessWork(t) }

func TestSamePRRefreshesAreCoalesced(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	first := reviewAdmissionJob("one", "refresh_review_context", "chat-a", "user-a", 42, reviewOperationTestTime)
	second := reviewAdmissionJob("two", "refresh_review_context", "chat-a", "user-b", 42, reviewOperationTestTime.Add(time.Second))
	a, err := store.AdmitReviewGatewayJob(context.Background(), first, "user_command", reviewOperationTestTime, limits)
	if err != nil || !a.Created {
		t.Fatal(a, err)
	}
	b, err := store.AdmitReviewGatewayJob(context.Background(), second, "user_command", reviewOperationTestTime.Add(time.Second), limits)
	if err != nil || !b.Coalesced || b.JobID != a.JobID {
		t.Fatalf("admission=%#v err=%v", b, err)
	}
}

func TestDifferentChatsAreNotCoalesced(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	a, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "refresh_review_context", "chat-a", "user-a", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "refresh_review_context", "chat-b", "user-b", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	if !a.Created || !b.Created {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

func TestCollaborationActionsAreNotCoalesced(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	a, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "claim_review", "chat-a", "user-a", 42, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "claim_review", "chat-a", "user-b", 42, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	if !a.Created || !b.Created {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

func TestControlledWritesAreNotCoalesced(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	a, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "confirm_common_review", "chat-a", "user-a", 42, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "confirm_common_review", "chat-a", "user-a", 42, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	if !a.Created || !b.Created {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

func TestCoalescedUserRequestsRetainBothConsumers(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	first := reviewAdmissionJob("one", "refresh_review_context", "chat-a", "user-a", 42, reviewOperationTestTime)
	first.SourceMessageID = "message-one"
	second := reviewAdmissionJob("two", "refresh_review_context", "chat-a", "user-b", 42, reviewOperationTestTime.Add(time.Second))
	second.SourceMessageID = "message-two"
	a, _ := store.AdmitReviewGatewayJob(context.Background(), first, "user_command", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), second, "user_command", reviewOperationTestTime.Add(time.Second), limits)
	consumers, _ := store.ListReviewJobConsumers(context.Background(), a.JobID)
	if !b.Coalesced || len(consumers) != 2 {
		t.Fatalf("b=%#v consumers=%#v", b, consumers)
	}
}

func TestCoalescedJobExecutesOneGitLinkRead(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "refresh_review_context", "chat-a", "user-a", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "refresh_review_context", "chat-a", "user-b", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime.Add(time.Second), limits)
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{QueueClass: ReviewQueueGitLinkRead, LeaseOwner: "reader", Now: reviewOperationTestTime.Add(2 * time.Second), LeaseDuration: time.Minute, Limit: 10})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%d err=%v", len(claimed), err)
	}
}

func TestConsumerRepliesAreDeliveredIndependently(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewAdmissionJob("one", "refresh_review_context", "chat-a", "user-a", 42, reviewOperationTestTime)
	job.SourceMessageID = "one"
	first, _ := store.AdmitReviewGatewayJob(context.Background(), job, "user_command", reviewOperationTestTime, DefaultReviewQueueLimits())
	second := job
	second.JobID = "job-two"
	second.DedupeKey = "dedupe-two"
	second.SourceMessageID = "two"
	_, _ = store.AdmitReviewGatewayJob(context.Background(), second, "user_command", reviewOperationTestTime.Add(time.Second), DefaultReviewQueueLimits())
	consumers, _ := store.ListReviewJobConsumers(context.Background(), first.JobID)
	planner := ReviewOperationPlanner{Store: store}
	operations, err := planner.BuildWithConsumers(job, reviewOperationTestResult(), consumers, reviewOperationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	replies := 0
	for _, operation := range operations {
		if operation.OperationKind == ReviewOperationReplySend {
			replies++
		}
	}
	if replies != 2 {
		t.Fatalf("reply operations=%d", replies)
	}
}

func TestReconciliationCoalescesWithEventRefresh(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	a, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("event", "refresh_review_context", "chat-a", "event", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("reconcile", "refresh_review_context", "chat-a", "scheduler", 42, reviewOperationTestTime.Add(time.Second)), "reconciliation", reviewOperationTestTime.Add(time.Second), limits)
	if !a.Created || !b.Coalesced {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

func TestCoalesceWindowExpiryCreatesNewJob(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	limits.RefreshCoalesceWindow = time.Second
	a, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("one", "refresh_review_context", "chat-a", "event", 42, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	b, _ := store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("two", "refresh_review_context", "chat-a", "event", 42, reviewOperationTestTime.Add(2*time.Second)), "event_route", reviewOperationTestTime.Add(2*time.Second), limits)
	if !a.Created || !b.Created {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

type recordingJobClassHandler struct {
	readStarted       chan struct{}
	releaseRead       chan struct{}
	collaborationDone chan struct{}
	calls             atomic.Int32
}

func (h *recordingJobClassHandler) execute(ctx context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
	h.calls.Add(1)
	if job.QueueClass == ReviewQueueGitLinkRead {
		select {
		case h.readStarted <- struct{}{}:
		default:
			{
			}
		}
		select {
		case <-h.releaseRead:
		case <-ctx.Done():
			return ReviewGatewayExecutionResult{}, ctx.Err()
		}
	}
	if job.QueueClass == ReviewQueueCollaboration {
		select {
		case h.collaborationDone <- struct{}{}:
		default:
			{
			}
		}
	}
	return ReviewGatewayExecutionResult{Status: "completed", CompletedAt: reviewGatewayTimestamp(time.Now())}, nil
}

func TestSlowGitLinkReadDoesNotBlockCollaboration(t *testing.T) {
	store := openReviewOperationTestStore(t)
	limits := DefaultReviewQueueLimits()
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("read", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, limits)
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("collab", "claim_review", "chat-a", "user-b", 2, reviewOperationTestTime), "user_command", reviewOperationTestTime, limits)
	gateway, _ := NewReviewGateway(ReviewGatewayBindings{}, ReviewGatewayConfig{}, store)
	queue := NewReviewGatewayQueue(gateway, store, 10, nil)
	handler := &recordingJobClassHandler{readStarted: make(chan struct{}, 1), releaseRead: make(chan struct{}), collaborationDone: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	manager := ReviewWorkerManager{Queue: queue, Concurrency: DefaultReviewWorkerConcurrency(), PollInterval: 10 * time.Millisecond, InstanceID: "test"}
	go func() { _ = manager.Run(ctx, handler.execute) }()
	select {
	case <-handler.readStarted:
	case <-time.After(time.Second):
		t.Fatal("read did not start")
	}
	select {
	case <-handler.collaborationDone:
	case <-time.After(time.Second):
		t.Fatal("collaboration was blocked")
	}
	close(handler.releaseRead)
	cancel()
}

func TestAgentWorkerIsDisabledByDefault(t *testing.T) {
	if DefaultReviewWorkerConcurrency().Agent != 0 {
		t.Fatal("agent worker enabled by default")
	}
}

func TestControlledWriteUsesDedicatedWorker(t *testing.T) {
	class, err := reviewGatewayQueueClassForAction("confirm_common_review")
	if err != nil || class != ReviewQueueControlledWrite {
		t.Fatalf("class=%s err=%v", class, err)
	}
}

func TestWorkerDoesNotHoldSQLiteTransactionDuringNetworkCall(t *testing.T) {
	store := openReviewOperationTestStore(t)
	_, _ = store.AdmitReviewGatewayJob(context.Background(), reviewAdmissionJob("read", "read_review_context", "chat-a", "user-a", 1, reviewOperationTestTime), "event_route", reviewOperationTestTime, DefaultReviewQueueLimits())
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{QueueClass: ReviewQueueGitLinkRead, LeaseOwner: "worker", Now: reviewOperationTestTime, LeaseDuration: time.Minute, Limit: 1})
	if err != nil || len(claimed) != 1 {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		var count int
		done <- store.db.QueryRow(`SELECT COUNT(*) FROM review_gateway_jobs`).Scan(&count)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("SQLite remained locked after claim")
	}
}

type blockingOperationExecutor struct {
	blockKind ReviewOperationKind
	started   chan struct{}
	release   chan struct{}
	completed chan ReviewOperationKind
	mu        sync.Mutex
}

func (e *blockingOperationExecutor) Execute(ctx context.Context, operation ReviewOperation) (ReviewOperationExecutionResult, error) {
	if operation.OperationKind == e.blockKind {
		select {
		case e.started <- struct{}{}:
		default:
			{
			}
		}
		select {
		case <-e.release:
		case <-ctx.Done():
			return ReviewOperationExecutionResult{}, ctx.Err()
		}
	}
	select {
	case e.completed <- operation.OperationKind:
	default:
		{
		}
	}
	return ReviewOperationExecutionResult{AppliedFingerprint: operation.DesiredFingerprint}, nil
}

func TestSlowBaseDoesNotBlockCanonicalCard(t *testing.T) {
	testSlowResourceDoesNotBlockKind(t, ReviewOperationBitableUpsert, ReviewOperationCanonicalCardUpsert)
}
func TestSlowDocDoesNotBlockTask(t *testing.T) {
	testSlowResourceDoesNotBlockKind(t, ReviewOperationDocSnapshotUpsert, ReviewOperationTaskUpsert)
}
func testSlowResourceDoesNotBlockKind(t *testing.T, slow, target ReviewOperationKind) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	slowOp, _ := NewReviewOperation(slow, job, "slow", mapOperationResource(slow), map[string]string{"target_scope": "chat"}, ReviewRetryIdempotent, reviewOperationTestTime)
	targetOp, _ := NewReviewOperation(target, job, "target", mapOperationResource(target), map[string]string{"target_scope": "chat"}, ReviewRetryIdempotent, reviewOperationTestTime)
	_ = store.SaveReviewOperations(context.Background(), []ReviewOperation{slowOp, targetOp})
	executor := &blockingOperationExecutor{blockKind: slow, started: make(chan struct{}, 1), release: make(chan struct{}), completed: make(chan ReviewOperationKind, 4)}
	concurrency := DefaultReviewWorkerConcurrency()
	workers := NewReviewOperationWorkerPool(store, executor, concurrency, "test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, worker := range workers {
		worker.PollInterval = 10 * time.Millisecond
		go func(w *ReviewOperationWorker) { _ = w.Run(ctx) }(worker)
	}
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("slow operation did not start")
	}
	deadline := time.After(time.Second)
	for {
		select {
		case kind := <-executor.completed:
			if kind == target {
				close(executor.release)
				return
			}
		case <-deadline:
			close(executor.release)
			t.Fatalf("target %s blocked by %s", target, slow)
		}
	}
}
func mapOperationResource(kind ReviewOperationKind) string {
	switch kind {
	case ReviewOperationCanonicalCardUpsert:
		return ReviewResourceCard
	case ReviewOperationBitableUpsert:
		return ReviewResourceBitable
	case ReviewOperationDocSnapshotUpsert:
		return ReviewResourceDoc
	case ReviewOperationTaskUpsert:
		return ReviewResourceTask
	default:
		return "feishu_reply"
	}
}
