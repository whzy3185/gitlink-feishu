package feishu

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

type fakeReviewOperationClient struct {
	mu                                                              sync.Mutex
	tokenCalls, searchCalls, baseCreateCalls, baseUpdateCalls       int
	docCreateCalls, docAppendCalls, taskCreateCalls, taskPatchCalls int
	search                                                          BitableSearchResult
	baseID, docID, taskID                                           string
	err                                                             map[string]error
}

func (f *fakeReviewOperationClient) failure(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err[name]
}
func (f *fakeReviewOperationClient) TenantAccessToken(context.Context, string, string) (TenantToken, error) {
	f.mu.Lock()
	f.tokenCalls++
	err := f.err["token"]
	f.mu.Unlock()
	return TenantToken{Value: "tenant-token", Expire: 7200}, err
}
func (f *fakeReviewOperationClient) SearchBitableRecord(context.Context, string, string, string, string) (BitableSearchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searchCalls++
	return f.search, f.err["search"]
}
func (f *fakeReviewOperationClient) CreateBitableRecord(context.Context, string, string, string, map[string]interface{}) (BitableWriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.baseCreateCalls++
	return BitableWriteResult{RecordID: firstNonEmpty(f.baseID, "record-created")}, f.err["base_create"]
}
func (f *fakeReviewOperationClient) UpdateBitableRecord(context.Context, string, string, string, string, map[string]interface{}) (BitableWriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.baseUpdateCalls++
	return BitableWriteResult{RecordID: firstNonEmpty(f.baseID, "record-updated")}, f.err["base_update"]
}
func (f *fakeReviewOperationClient) CreateDocument(context.Context, string, string, string) (CreatedDocument, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docCreateCalls++
	return CreatedDocument{DocumentID: firstNonEmpty(f.docID, "document-created")}, f.err["doc_create"]
}
func (f *fakeReviewOperationClient) CreateBlocks(context.Context, string, string, string, []DocBlock) (CreatedBlocks, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docAppendCalls++
	return CreatedBlocks{RevisionID: 1}, f.err["doc_append"]
}
func (f *fakeReviewOperationClient) CreateTask(context.Context, string, TaskCandidate) (CreatedTask, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.taskCreateCalls++
	return CreatedTask{TaskID: firstNonEmpty(f.taskID, "task-created")}, f.err["task_create"]
}
func (f *fakeReviewOperationClient) PatchTask(context.Context, string, string, TaskCandidate, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.taskPatchCalls++
	return f.err["task_patch"]
}

type fakeReviewOperationSender struct {
	mu             sync.Mutex
	sends, patches int
	err            error
}

func (f *fakeReviewOperationSender) Send(context.Context, *larktypes.SendInput) (*larktypes.SendResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends++
	if f.err != nil {
		return nil, f.err
	}
	return &larktypes.SendResult{MessageID: "message-created", ChatID: "chat-a"}, nil
}
func (f *fakeReviewOperationSender) UpdateInteractiveMessage(context.Context, string, Card) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.patches++
	return f.err
}

func newReviewOperationResourceHarness(t *testing.T) (*SQLiteReviewGatewayStore, *fakeReviewOperationClient, *fakeReviewOperationSender, *ReviewOperationHandler) {
	t.Helper()
	store := openReviewOperationTestStore(t)
	client := &fakeReviewOperationClient{err: map[string]error{}}
	sender := &fakeReviewOperationSender{}
	handler := &ReviewOperationHandler{
		Store: store, Client: client, Sender: sender,
		Config: ReviewCollaborationPublisherConfig{AppID: "app", AppSecret: "secret", BaseAppToken: "base", ReviewTableID: "table", DocumentFolderToken: "folder", EnableTask: true},
		Now:    func() time.Time { return reviewOperationTestTime },
	}
	handler.TokenProvider = NewReviewTenantTokenProvider(client, handler.Now)
	return store, client, sender, handler
}

func executeReviewOperationForTest(t *testing.T, store *SQLiteReviewGatewayStore, handler *ReviewOperationHandler, operation ReviewOperation) ReviewOperation {
	t.Helper()
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{operation}); err != nil {
		t.Fatal(err)
	}
	worker := NewReviewOperationWorker(store, handler, operation.QueueClass, "worker-test")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Minute) }
	worker.BatchSize = 10
	ran, err := worker.RunOnce(context.Background())
	if err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	stored, err := store.GetReviewOperation(context.Background(), operation.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func TestCanonicalCardCreateRunsThroughOperation(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	op, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, PresentationKey: reviewChatPRPresentationKey(job), SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime), Card: Card{"elements": []interface{}{map[string]interface{}{"tag": "div"}}}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationSucceeded || sender.sends != 1 {
		t.Fatalf("status=%s sends=%d", stored.Status, sender.sends)
	}
}

func TestCanonicalCardPatchRunsThroughOperation(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	first, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime), Card: Card{"value": "one"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, first)
	second, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime.Add(time.Minute)), Card: Card{"value": "two"}}, ReviewRetryNonIdempotent, reviewOperationTestTime.Add(time.Minute))
	stored := executeReviewOperationForTest(t, store, handler, second)
	if stored.Status != ReviewOperationSucceeded || sender.patches != 1 || sender.sends != 1 {
		t.Fatalf("status=%s sends=%d patches=%d", stored.Status, sender.sends, sender.patches)
	}
}

func TestCardCreateTimeoutBecomesUnknown(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	sender.err = context.DeadlineExceeded
	job := reviewOperationTestJob()
	op, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"value": "one"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationUnknown || !stored.RequiresReconciliation {
		t.Fatalf("status=%s reconcile=%t", stored.Status, stored.RequiresReconciliation)
	}
}

func TestUnknownCardIsNotAutomaticallyCreatedAgain(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	sender.err = context.DeadlineExceeded
	job := reviewOperationTestJob()
	op, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"value": "one"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, op)
	worker := NewReviewOperationWorker(store, handler, "canonical_card", "other")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Hour) }
	ran, err := worker.RunOnce(context.Background())
	if err != nil || ran || sender.sends != 1 {
		t.Fatalf("run=%t sends=%d err=%v", ran, sender.sends, err)
	}
}

func TestCardPatchTransientErrorCanRetry(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	first, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime), Card: Card{"value": "one"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, first)
	sender.err = &OpenAPIHTTPError{StatusCode: 503, Detail: "unavailable"}
	second, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, SourceCompletedAt: reviewGatewayTimestamp(reviewOperationTestTime.Add(time.Minute)), Card: Card{"value": "two"}}, ReviewRetryNonIdempotent, reviewOperationTestTime.Add(time.Minute))
	stored := executeReviewOperationForTest(t, store, handler, second)
	if stored.Status != ReviewOperationRetryScheduled {
		t.Fatalf("status=%s", stored.Status)
	}
}

func TestReplyRunsThroughOperation(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	op, _ := NewReviewOperation(ReviewOperationReplySend, reviewOperationTestJob(), "consumer", "feishu_reply", reviewReplyDesired{JobID: "job", SourceMessageID: "source", Text: "done"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationSucceeded || sender.sends != 1 {
		t.Fatalf("status=%s sends=%d", stored.Status, sender.sends)
	}
}

func TestReplyTimeoutBecomesUnknown(t *testing.T) {
	store, _, sender, handler := newReviewOperationResourceHarness(t)
	sender.err = context.DeadlineExceeded
	op, _ := NewReviewOperation(ReviewOperationReplySend, reviewOperationTestJob(), "consumer", "feishu_reply", reviewReplyDesired{Text: "done"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationUnknown {
		t.Fatalf("status=%s", stored.Status)
	}
}

func TestBaseSearchFindsExistingRecord(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.search = BitableSearchResult{Found: true, Matches: 1, RecordID: "existing"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat", Fields: map[string]interface{}{"unique_key": "base-key"}}, ReviewRetryReconcilable, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.RemoteID != "existing" || client.baseCreateCalls != 0 || client.baseUpdateCalls != 1 {
		t.Fatalf("stored=%#v creates=%d updates=%d", stored, client.baseCreateCalls, client.baseUpdateCalls)
	}
}

func TestBaseMultipleMatchesRequireReconciliation(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.search = BitableSearchResult{Matches: 2}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationUnknown {
		t.Fatalf("status=%s", stored.Status)
	}
}

func TestBase429UsesRetryAfter(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 429, RetryAfter: 7 * time.Second, Detail: "limited"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationRetryScheduled || stored.RetryAfterAt == "" {
		t.Fatalf("status=%s retry=%q", stored.Status, stored.RetryAfterAt)
	}
}

func TestBase403DoesNotRetry(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	op, _ := NewReviewOperation(ReviewOperationBitableUpsert, reviewOperationTestJob(), "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationFailedTerminal {
		t.Fatalf("status=%s", stored.Status)
	}
}

func TestBaseFailureDoesNotBlockDocAndTask(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["search"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	job := reviewOperationTestJob()
	base, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	doc, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, job, "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	task, _ := NewReviewOperation(ReviewOperationTaskUpsert, job, "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Review"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{base, doc, task}); err != nil {
		t.Fatal(err)
	}
	worker := NewReviewOperationWorker(store, handler, "resource", "worker-resources")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Minute) }
	worker.BatchSize = 10
	if ran, err := worker.RunOnce(context.Background()); err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	docStored, _ := store.GetReviewOperation(context.Background(), doc.OperationID)
	taskStored, _ := store.GetReviewOperation(context.Background(), task.OperationID)
	if docStored.Status != ReviewOperationSucceeded || taskStored.Status != ReviewOperationSucceeded {
		t.Fatalf("doc=%s task=%s", docStored.Status, taskStored.Status)
	}
}

func TestDocCreatePersistsRemoteIDBeforeAppend(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["doc_append"] = context.DeadlineExceeded
	op, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, reviewOperationTestJob(), "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Title: "Review", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	state, _ := store.GetReviewResourceState(context.Background(), "doc-key", ReviewResourceDoc)
	if stored.Status != ReviewOperationUnknown || state.RemoteID == "" || state.ContentFingerprint != "" {
		t.Fatalf("operation=%#v state=%#v", stored, state)
	}
}

func TestDocAppendUnknownDoesNotBlindlyAppendAgain(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["doc_append"] = context.DeadlineExceeded
	op, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, reviewOperationTestJob(), "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Title: "Review", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	executeReviewOperationForTest(t, store, handler, op)
	worker := NewReviewOperationWorker(store, handler, "resource", "worker-again")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Hour) }
	ran, _ := worker.RunOnce(context.Background())
	if ran || client.docAppendCalls != 1 {
		t.Fatalf("run=%t appends=%d", ran, client.docAppendCalls)
	}
}

func TestDocFailureDoesNotBlockBaseAndTask(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["doc_create"] = &OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}
	job := reviewOperationTestJob()
	base, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	doc, _ := NewReviewOperation(ReviewOperationDocSnapshotUpsert, job, "doc-key", ReviewResourceDoc, reviewDocDesired{ResourceKey: "doc-key", TargetScope: "chat", Markdown: "snapshot"}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	task, _ := NewReviewOperation(ReviewOperationTaskUpsert, job, "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: &TaskCandidate{UniqueKey: "task-key", Title: "Review"}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{base, doc, task}); err != nil {
		t.Fatal(err)
	}
	worker := NewReviewOperationWorker(store, handler, "resource", "worker-resources")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Minute) }
	worker.BatchSize = 10
	if ran, err := worker.RunOnce(context.Background()); err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	baseStored, _ := store.GetReviewOperation(context.Background(), base.OperationID)
	taskStored, _ := store.GetReviewOperation(context.Background(), task.OperationID)
	if baseStored.Status != ReviewOperationSucceeded || taskStored.Status != ReviewOperationSucceeded {
		t.Fatalf("base=%s task=%s", baseStored.Status, taskStored.Status)
	}
}

func TestTaskCreateUnknownRequiresReconciliation(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	client.err["task_create"] = context.DeadlineExceeded
	task := &TaskCandidate{UniqueKey: "task-key", Title: "Review"}
	op, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Task: task}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationUnknown || !stored.RequiresReconciliation {
		t.Fatalf("status=%s", stored.Status)
	}
}

func TestArchivedPRWithoutTaskDoesNotCreateTask(t *testing.T) {
	store, client, _, handler := newReviewOperationResourceHarness(t)
	op, _ := NewReviewOperation(ReviewOperationTaskUpsert, reviewOperationTestJob(), "task-key", ReviewResourceTask, reviewTaskDesired{ResourceKey: "task-key", TargetScope: "chat", Archived: true}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	stored := executeReviewOperationForTest(t, store, handler, op)
	if stored.Status != ReviewOperationUnchanged || client.taskCreateCalls != 0 {
		t.Fatalf("status=%s creates=%d", stored.Status, client.taskCreateCalls)
	}
}

func TestOperation403IsTerminal(t *testing.T) {
	err := ClassifyReviewOperationError(&OpenAPIHTTPError{StatusCode: 403, Detail: "forbidden"}, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, true)
	if err.Class != ReviewOperationErrorTerminal {
		t.Fatalf("class=%s", err.Class)
	}
}

func TestOperation429IsRateLimited(t *testing.T) {
	err := ClassifyReviewOperationError(&OpenAPIHTTPError{StatusCode: 429, RetryAfter: time.Second}, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, true)
	if err.Class != ReviewOperationErrorRateLimited || err.RetryAfter != time.Second {
		t.Fatalf("error=%#v", err)
	}
}

func TestOperation503IsTransient(t *testing.T) {
	err := ClassifyReviewOperationError(&OpenAPIHTTPError{StatusCode: 503}, ReviewOperation{RetrySafety: ReviewRetryIdempotent}, true)
	if err.Class != ReviewOperationErrorTransient {
		t.Fatalf("class=%s", err.Class)
	}
}

func TestNonIdempotentTimeoutIsUnknown(t *testing.T) {
	err := ClassifyReviewOperationError(context.DeadlineExceeded, ReviewOperation{RetrySafety: ReviewRetryNonIdempotent}, true)
	if err.Class != ReviewOperationErrorUnknownSideEffect || !err.RemoteSideEffectPossible {
		t.Fatalf("error=%#v", err)
	}
}

func TestReviewTenantTokenProviderCoalescesRefresh(t *testing.T) {
	client := &fakeReviewOperationClient{err: map[string]error{}}
	provider := NewReviewTenantTokenProvider(client, func() time.Time { return reviewOperationTestTime })
	for i := 0; i < 3; i++ {
		if _, err := provider.Token(context.Background(), "app", "secret"); err != nil {
			t.Fatal(err)
		}
	}
	if client.tokenCalls != 1 {
		t.Fatalf("token calls=%d", client.tokenCalls)
	}
}

func TestCardShowsPerResourceSyncState(t *testing.T) {
	card := reviewCardWithProjectionStatuses(Card{"elements": []interface{}{}}, []ReviewResourceProjectionStatus{
		{ResourceType: ReviewResourceBitable, Status: ReviewProjectionSucceeded},
		{ResourceType: ReviewResourceDoc, Status: ReviewProjectionPending},
		{ResourceType: ReviewResourceTask, Status: ReviewProjectionNeedsReconciliation},
	})
	encoded, _ := encodeReviewGatewayCard(card)
	for _, expected := range []string{"Base：已同步", "Doc：同步中", "Task：待对账"} {
		if !strings.Contains(encoded, expected) {
			t.Fatalf("card=%s missing %q", encoded, expected)
		}
	}
}

func TestResourceStatusChangeQueuesCardRefresh(t *testing.T) {
	store, _, _, handler := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	card, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"elements": []interface{}{}}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	base, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{card, base}); err != nil {
		t.Fatal(err)
	}
	worker := NewReviewOperationWorker(store, handler, "resource", "resource-worker")
	worker.Now = func() time.Time { return reviewOperationTestTime.Add(time.Minute) }
	if ran, err := worker.RunOnce(context.Background()); err != nil || !ran {
		t.Fatalf("run=%t err=%v", ran, err)
	}
	operations, _ := store.ListReviewOperations(context.Background(), "canonical_card", "pending", 10)
	if len(operations) != 1 || operations[0].OperationID == card.OperationID {
		t.Fatalf("refresh operations=%#v", operations)
	}
}

func TestResourceStatusCardRefreshIsCoalesced(t *testing.T) {
	store, _, _, _ := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	card, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"elements": []interface{}{}}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{card}); err != nil {
		t.Fatal(err)
	}
	source, _ := NewReviewOperation(ReviewOperationBitableUpsert, job, "base-key", ReviewResourceBitable, reviewBitableDesired{ResourceKey: "base-key", TargetScope: "chat"}, ReviewRetryReconcilable, reviewOperationTestTime)
	if err := store.QueueReviewProjectionCardRefresh(context.Background(), source, reviewOperationTestTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.QueueReviewProjectionCardRefresh(context.Background(), source, reviewOperationTestTime.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	operations, _ := store.ListReviewOperations(context.Background(), "canonical_card", "", 10)
	if len(operations) != 2 {
		t.Fatalf("operations=%d, want original plus one refresh", len(operations))
	}
}

func TestCardStatusDoesNotCauseRefreshLoop(t *testing.T) {
	store, _, _, _ := newReviewOperationResourceHarness(t)
	job := reviewOperationTestJob()
	card, _ := NewReviewOperation(ReviewOperationCanonicalCardUpsert, job, reviewChatPRPresentationKey(job), ReviewResourceCard, reviewCanonicalCardDesired{JobID: job.JobID, Card: Card{"elements": []interface{}{}}}, ReviewRetryNonIdempotent, reviewOperationTestTime)
	if err := store.SaveReviewOperations(context.Background(), []ReviewOperation{card}); err != nil {
		t.Fatal(err)
	}
	if err := store.QueueReviewProjectionCardRefresh(context.Background(), card, reviewOperationTestTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	operations, _ := store.ListReviewOperations(context.Background(), "canonical_card", "", 10)
	if len(operations) != 1 {
		t.Fatalf("card status created loop: %d", len(operations))
	}
}

func TestDisabledResourceShowsDisabled(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	planner := &ReviewOperationPlanner{Store: store, Config: ReviewCollaborationPublisherConfig{BaseScope: ReviewResourceScopeDisabled, DocScope: ReviewResourceScopeDisabled}, Now: func() time.Time { return reviewOperationTestTime }}
	if _, err := planner.Plan(context.Background(), job, reviewOperationTestResult()); err != nil {
		t.Fatal(err)
	}
	statuses, _ := store.ListReviewResourceProjectionStatuses(context.Background(), job.InstallationID, job.ChatID, job.Repository, job.PRNumber)
	if len(statuses) != 3 || statuses[0].Status != ReviewProjectionDisabled {
		t.Fatalf("statuses=%#v", statuses)
	}
}

func TestMissingConfigurationShowsNotConfigured(t *testing.T) {
	store := openReviewOperationTestStore(t)
	job := reviewOperationTestJob()
	planner := &ReviewOperationPlanner{Store: store, Now: func() time.Time { return reviewOperationTestTime }}
	if _, err := planner.Plan(context.Background(), job, reviewOperationTestResult()); err != nil {
		t.Fatal(err)
	}
	statuses, _ := store.ListReviewResourceProjectionStatuses(context.Background(), job.InstallationID, job.ChatID, job.Repository, job.PRNumber)
	values := map[string]ReviewResourceProjectionStatusValue{}
	for _, status := range statuses {
		values[status.ResourceType] = status.Status
	}
	if values[ReviewResourceBitable] != ReviewProjectionNotConfigured || values[ReviewResourceDoc] != ReviewProjectionNotConfigured || values[ReviewResourceTask] != ReviewProjectionDisabled {
		t.Fatalf("statuses=%#v", values)
	}
}
