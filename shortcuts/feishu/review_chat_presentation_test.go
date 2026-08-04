package feishu

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

func TestCanonicalCardIsScopedByInstallationAndChat(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	jobs := []ReviewGatewayJob{
		testReviewGatewayJob(reviewProductInvariantTime, "scope-a"),
		testReviewGatewayJob(reviewProductInvariantTime, "scope-b"),
		testReviewGatewayJob(reviewProductInvariantTime, "scope-c"),
	}
	jobs[1].ChatID = "oc_other"
	jobs[2].InstallationID = "other-installation"
	keys := map[string]bool{}
	for index, job := range jobs {
		operationID := "operation-scope-" + string(rune('a'+index))
		state, acquired, err := store.AcquireChatPRPresentationCreate(context.Background(), job, operationID, reviewProductInvariantTime)
		if err != nil || !acquired {
			t.Fatalf("acquire scope %d: acquired=%t state=%#v err=%v", index, acquired, state, err)
		}
		keys[state.PresentationKey] = true
	}
	if len(keys) != 3 {
		t.Fatalf("canonical card scopes collapsed: %#v", keys)
	}
}

func TestCanonicalCardCreateStoresMessageID(t *testing.T) {
	store, _, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "canonical-create", reviewProductInvariantTime, result, item)
	job := testReviewGatewayJob(reviewProductInvariantTime, "canonical-create")
	state, err := store.GetChatPRPresentation(context.Background(), job)
	if err != nil || state.CanonicalMessageID != "om_reply" || state.CardStatus != "active" || state.PresentationVersion != 1 {
		t.Fatalf("canonical create state=%#v err=%v", state, err)
	}
}

func TestCanonicalCardUnchangedSkipsPatch(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "canonical-unchanged-a", reviewProductInvariantTime, result, item)
	deliverProductInvariantReply(t, store, dispatcher, "canonical-unchanged-b", reviewProductInvariantTime.Add(time.Minute), result, item)
	_, textNotices, updates := productInvariantSenderCounts(sender)
	if updates != 0 || textNotices != 1 {
		t.Fatalf("unchanged canonical card patched: updates=%d notices=%d", updates, textNotices)
	}
}

func TestCanonicalCardChangedPatchesExactlyOnce(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "canonical-change-a", reviewProductInvariantTime, result, item)
	result.HeadSHA = "changed-head"
	result.SourceFingerprint = "changed-source"
	deliverProductInvariantReply(t, store, dispatcher, "canonical-change-b", reviewProductInvariantTime.Add(time.Minute), result, item)
	deliverProductInvariantReply(t, store, dispatcher, "canonical-change-c", reviewProductInvariantTime.Add(2*time.Minute), result, item)
	_, textNotices, updates := productInvariantSenderCounts(sender)
	if updates != 1 || textNotices != 2 {
		t.Fatalf("changed canonical card operation count: updates=%d notices=%d", updates, textNotices)
	}
}

func TestCanonicalCardCollaborationActionKeepsFullContent(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "canonical-action-read", reviewProductInvariantTime, result, item)
	job := testReviewGatewayJob(reviewProductInvariantTime.Add(time.Minute), "canonical-action-claim")
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, result, reviewProductInvariantTime); err != nil {
		t.Fatalf("seed presentation: %v", err)
	}
	claimResult := executeProductInvariantAction(t, store, job, "claim_review", "", reviewProductInvariantTime.Add(time.Minute))
	completeReviewPresentationReply(t, store, job, claimResult, reviewProductInvariantTime.Add(time.Minute))
	dispatcher.deliverPendingReplies(context.Background())
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.updates) != 1 {
		t.Fatalf("collaboration action updates=%d", len(sender.updates))
	}
	assertFullPRCardFacts(t, sender.updates[0])
}

type sequenceReviewGatewaySender struct {
	mu      sync.Mutex
	inputs  []larktypes.SendInput
	updates []Card
	nextID  int
}

func (s *sequenceReviewGatewaySender) Send(_ context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, *input)
	s.nextID++
	return &larktypes.SendResult{MessageID: "om_sequence_" + string(rune('0'+s.nextID)), ChatID: input.ChatID}, nil
}

func (s *sequenceReviewGatewaySender) UpdateInteractiveMessage(_ context.Context, _ string, card Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, card)
	return nil
}

func TestCurrentReplyMessageIDDoesNotReplaceCanonicalMessageID(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	sender := &sequenceReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "canonical-id-a", reviewProductInvariantTime, result, item)
	job := testReviewGatewayJob(reviewProductInvariantTime, "canonical-id-a")
	before, err := store.GetChatPRPresentation(context.Background(), job)
	if err != nil {
		t.Fatalf("read canonical before notice: %v", err)
	}
	deliverProductInvariantReply(t, store, dispatcher, "canonical-id-b", reviewProductInvariantTime.Add(time.Minute), result, item)
	after, err := store.GetChatPRPresentation(context.Background(), job)
	if err != nil || before.CanonicalMessageID == "" || after.CanonicalMessageID != before.CanonicalMessageID {
		t.Fatalf("reply ID replaced canonical ID: before=%#v after=%#v err=%v", before, after, err)
	}
}

type blockingCanonicalCreateSender struct {
	mu          sync.Mutex
	inputs      []larktypes.SendInput
	started     chan struct{}
	release     chan struct{}
	startedOnce sync.Once
}

func (s *blockingCanonicalCreateSender) Send(_ context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error) {
	s.mu.Lock()
	s.inputs = append(s.inputs, *input)
	s.mu.Unlock()
	if input.MsgType == "interactive" {
		s.startedOnce.Do(func() { close(s.started) })
		<-s.release
	}
	return &larktypes.SendResult{MessageID: "om_concurrent", ChatID: input.ChatID}, nil
}

func (s *blockingCanonicalCreateSender) UpdateInteractiveMessage(context.Context, string, Card) error {
	return nil
}

func TestConcurrentCanonicalCardCreateSendsOnce(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	sender := &blockingCanonicalCreateSender{started: make(chan struct{}), release: make(chan struct{})}
	first := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	second := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	clock := time.Now().Add(time.Minute)
	first.now = func() time.Time { return clock }
	second.now = func() time.Time { return clock }
	result := fullReviewGatewayResultFixture()
	firstJob := testReviewGatewayJob(reviewProductInvariantTime, "concurrent-card-a")
	secondJob := testReviewGatewayJob(reviewProductInvariantTime.Add(time.Second), "concurrent-card-b")
	completeReviewPresentationReply(t, store, firstJob, result, reviewProductInvariantTime)
	completeReviewPresentationReply(t, store, secondJob, result, reviewProductInvariantTime.Add(time.Second))
	done := make(chan struct{})
	go func() {
		first.deliverPendingReplies(context.Background())
		close(done)
	}()
	<-sender.started
	second.deliverPendingReplies(context.Background())
	close(sender.release)
	<-done
	sender.mu.Lock()
	interactive := 0
	for _, input := range sender.inputs {
		if input.MsgType == "interactive" {
			interactive++
		}
	}
	sender.mu.Unlock()
	if interactive != 1 {
		t.Fatalf("concurrent canonical creates=%d", interactive)
	}
}

func TestStalePatchWorkerDoesNotOverwriteNewFingerprint(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "stale-patch")
	_, acquired, err := store.AcquireChatPRPresentationCreate(context.Background(), job, "create", reviewProductInvariantTime)
	if err != nil || !acquired {
		t.Fatalf("acquire create: %t %v", acquired, err)
	}
	if err := store.CompleteChatPRPresentationCreate(context.Background(), job, "create", "om_card", "fingerprint-1", reviewProductInvariantTime.Format(time.RFC3339), false, reviewProductInvariantTime); err != nil {
		t.Fatalf("complete create: %v", err)
	}
	_, acquired, err = store.AcquireChatPRPresentationPatch(context.Background(), job, "fingerprint-1", "patch-new", reviewProductInvariantTime.Add(time.Minute).Format(time.RFC3339), reviewProductInvariantTime.Add(time.Minute))
	if err != nil || !acquired {
		t.Fatalf("acquire patch: %t %v", acquired, err)
	}
	if err := store.CompleteChatPRPresentationPatch(context.Background(), job, "patch-stale", "stale-fingerprint", reviewProductInvariantTime.Add(time.Minute).Format(time.RFC3339), false, reviewProductInvariantTime.Add(time.Minute)); err == nil {
		t.Fatal("stale patch worker completed another worker's operation")
	}
	if err := store.CompleteChatPRPresentationPatch(context.Background(), job, "patch-new", "fingerprint-2", reviewProductInvariantTime.Add(time.Minute).Format(time.RFC3339), false, reviewProductInvariantTime.Add(time.Minute)); err != nil {
		t.Fatalf("complete current patch: %v", err)
	}
	state, _ := store.GetChatPRPresentation(context.Background(), job)
	if state.ContentFingerprint != "fingerprint-2" || state.PresentationVersion != 2 {
		t.Fatalf("stale worker overwrote state: %#v", state)
	}
	if _, acquired, err := store.AcquireChatPRPresentationPatch(
		context.Background(), job, "fingerprint-2", "patch-old-source",
		reviewProductInvariantTime.Format(time.RFC3339), reviewProductInvariantTime.Add(2*time.Minute),
	); !errors.Is(err, ErrStaleReviewPRPresentation) || acquired {
		t.Fatalf("old source acquired canonical patch: acquired=%t err=%v", acquired, err)
	}
	afterStale, _ := store.GetChatPRPresentation(context.Background(), job)
	if afterStale.ContentFingerprint != "fingerprint-2" || afterStale.CardStatus != "active" {
		t.Fatalf("old source changed canonical state: %#v", afterStale)
	}
}

func TestCanonicalCardMappingSurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical-restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	job := testReviewGatewayJob(reviewProductInvariantTime, "canonical-restart")
	_, acquired, _ := store.AcquireChatPRPresentationCreate(context.Background(), job, "create", reviewProductInvariantTime)
	if !acquired {
		t.Fatal("canonical create not acquired")
	}
	if err := store.CompleteChatPRPresentationCreate(context.Background(), job, "create", "om_persisted", "fingerprint", reviewProductInvariantTime.Format(time.RFC3339), false, reviewProductInvariantTime); err != nil {
		t.Fatalf("complete create: %v", err)
	}
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer store.Close()
	state, err := store.GetChatPRPresentation(context.Background(), job)
	if err != nil || state.CanonicalMessageID != "om_persisted" || state.CardStatus != "active" {
		t.Fatalf("canonical mapping after restart=%#v err=%v", state, err)
	}
}

func TestRestartDoesNotCreateSecondCanonicalCard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "canonical-no-duplicate.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	deliverProductInvariantReply(t, store, dispatcher, "restart-card-a", reviewProductInvariantTime, result, item)
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer store.Close()
	dispatcher = NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	deliverProductInvariantReply(t, store, dispatcher, "restart-card-b", reviewProductInvariantTime.Add(time.Minute), result, item)
	fullCards, notices, _ := productInvariantSenderCounts(sender)
	if fullCards != 1 || notices != 1 {
		t.Fatalf("restart duplicated canonical card: full=%d notices=%d", fullCards, notices)
	}
}

func TestArchivedPRKeepsExistingCanonicalCard(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "archive-a", reviewProductInvariantTime, result, item)
	result.GitLinkState = "merged"
	item.Archived = true
	deliverProductInvariantReply(t, store, dispatcher, "archive-b", reviewProductInvariantTime.Add(time.Minute), result, item)
	job := testReviewGatewayJob(reviewProductInvariantTime, "archive-a")
	state, err := store.GetChatPRPresentation(context.Background(), job)
	fullCards, _, updates := productInvariantSenderCounts(sender)
	if err != nil || state.CanonicalMessageID != "om_reply" || state.CardStatus != "archived" || fullCards != 1 || updates != 1 {
		t.Fatalf("archived canonical state=%#v full=%d updates=%d err=%v", state, fullCards, updates, err)
	}
}

func TestCanonicalCardUnknownIsNotAutomaticallyRecreated(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "unknown-card")
	_, acquired, _ := store.AcquireChatPRPresentationCreate(context.Background(), job, "unknown-create", reviewProductInvariantTime)
	if !acquired {
		t.Fatal("unknown create not acquired")
	}
	if err := store.MarkChatPRPresentationUnknown(context.Background(), job, "unknown-create", "", "uncertain create", reviewProductInvariantTime); err != nil {
		t.Fatalf("mark unknown: %v", err)
	}
	completeReviewPresentationReply(t, store, job, fullReviewGatewayResultFixture(), reviewProductInvariantTime)
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 1)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	dispatcher.deliverPendingReplies(context.Background())
	sender.mu.Lock()
	sends := len(sender.inputs)
	sender.mu.Unlock()
	if sends != 0 {
		t.Fatalf("unknown canonical card was recreated: sends=%d", sends)
	}
}

type canonicalPatchPersistFailureStore struct {
	*SQLiteReviewGatewayStore
}

func (s *canonicalPatchPersistFailureStore) CompleteChatPRPresentationPatch(
	context.Context, ReviewGatewayJob, string, string, string, bool, time.Time,
) error {
	return errors.New("simulated canonical patch persistence failure")
}

func TestCanonicalCardPatchSuccessPersistFailureEntersUnknown(t *testing.T) {
	base := openReviewPresentationTestStore(t)
	defer base.Close()
	store := &canonicalPatchPersistFailureStore{SQLiteReviewGatewayStore: base}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	result := fullReviewGatewayResultFixture()
	firstJob := testReviewGatewayJob(reviewProductInvariantTime, "patch-persist-a")
	completeReviewPresentationReply(t, store, firstJob, result, reviewProductInvariantTime)
	dispatcher.deliverPendingReplies(context.Background())
	result.HeadSHA = "patch-persist-changed"
	result.SourceFingerprint = "patch-persist-source"
	secondJob := testReviewGatewayJob(reviewProductInvariantTime.Add(time.Minute), "patch-persist-b")
	result.ResultCard = buildReviewGatewayResultCard(secondJob, result, nil)
	completeReviewPresentationReply(t, store, secondJob, result, reviewProductInvariantTime.Add(time.Minute))
	dispatcher.deliverPendingReplies(context.Background())
	dispatcher.deliverPendingReplies(context.Background())
	state, err := base.GetChatPRPresentation(context.Background(), firstJob)
	sender.mu.Lock()
	updates := len(sender.updates)
	sender.mu.Unlock()
	if err != nil || updates != 1 || state.CardStatus != "unknown" || !state.RequiresReconciliation {
		t.Fatalf("patch persistence uncertainty: updates=%d state=%#v err=%v", updates, state, err)
	}
}

type sqliteReplyPersistFailureStore struct {
	*SQLiteReviewGatewayStore
}

func (s *sqliteReplyPersistFailureStore) MarkReplySent(context.Context, string, string, time.Time) error {
	return errors.New("simulated reply persistence failure")
}

func TestReplyUnknownStatusSurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reply-unknown-restart.db")
	base, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	store := &sqliteReplyPersistFailureStore{SQLiteReviewGatewayStore: base}
	job := testReviewGatewayJob(reviewProductInvariantTime, "reply-unknown-restart")
	completeReviewPresentationReply(t, store, job, fullReviewGatewayResultFixture(), reviewProductInvariantTime)
	dispatcher := NewReviewGatewayReplyDispatcher(&recordingReviewGatewaySender{}, store, &reviewGatewayJSONOutput{writer: io.Discard}, 1)
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }
	dispatcher.deliverPendingReplies(context.Background())
	_ = base.Close()
	base, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer base.Close()
	var status string
	var reconciliation int
	if err := base.db.QueryRow(`SELECT reply_status, reply_requires_reconciliation FROM review_gateway_jobs WHERE job_id=?`, job.JobID).Scan(&status, &reconciliation); err != nil {
		t.Fatalf("read reply unknown state: %v", err)
	}
	if status != "unknown" || reconciliation != 1 {
		t.Fatalf("reply unknown state did not survive: status=%q reconciliation=%d", status, reconciliation)
	}
}

func TestConditionalStateTransitionRejectsStaleWorker(t *testing.T) {
	TestStalePatchWorkerDoesNotOverwriteNewFingerprint(t)
}

func TestChatPresentationMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat-presentation-migration.db")
	for pass := 0; pass < 2; pass++ {
		store, err := OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("open pass %d: %v", pass+1, err)
		}
		_ = store.Close()
	}
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open migration store: %v", err)
	}
	defer store.Close()
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name='chat_pr_presentations_v1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("chat presentation migration count=%d err=%v", count, err)
	}
}

type canonicalPersistFailureStore struct {
	*SQLiteReviewGatewayStore
}

func (s *canonicalPersistFailureStore) CompleteChatPRPresentationCreate(
	context.Context,
	ReviewGatewayJob,
	string,
	string,
	string,
	string,
	bool,
	time.Time,
) error {
	return errors.New("simulated canonical mapping persistence failure")
}

func TestCanonicalCardCreateSuccessPersistFailureEntersUnknown(t *testing.T) {
	base := openReviewPresentationTestStore(t)
	defer base.Close()
	store := &canonicalPersistFailureStore{SQLiteReviewGatewayStore: base}
	job := testReviewGatewayJob(reviewProductInvariantTime, "canonical-persist-failure")
	result := fullReviewGatewayResultFixture()
	completeReviewPresentationReply(t, store, job, result, reviewProductInvariantTime)
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 1)
	dispatcher.now = func() time.Time { return reviewProductInvariantTime.Add(time.Minute) }
	dispatcher.deliverPendingReplies(context.Background())
	dispatcher.deliverPendingReplies(context.Background())
	sender.mu.Lock()
	sendCount := len(sender.inputs)
	sender.mu.Unlock()
	state, err := base.GetChatPRPresentation(context.Background(), job)
	if err != nil {
		t.Fatalf("GetChatPRPresentation: %v", err)
	}
	var replyStatus string
	if err := base.db.QueryRow(`SELECT reply_status FROM review_gateway_jobs WHERE job_id=?`, job.JobID).Scan(&replyStatus); err != nil {
		t.Fatalf("read reply status: %v", err)
	}
	if sendCount != 1 || state.CardStatus != "unknown" || !state.RequiresReconciliation || replyStatus != "unknown" {
		t.Fatalf("unsafe canonical recovery: sends=%d state=%#v reply=%q", sendCount, state, replyStatus)
	}
}

type patchNoticeFailureSender struct {
	mu             sync.Mutex
	inputs         []larktypes.SendInput
	updates        []Card
	failNextNotice bool
	nextMessageID  string
}

func (s *patchNoticeFailureSender) Send(_ context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, *input)
	if input.MsgType == "text" && s.failNextNotice {
		s.failNextNotice = false
		return nil, errors.New("simulated lightweight notice failure")
	}
	return &larktypes.SendResult{MessageID: firstNonEmpty(s.nextMessageID, "om_notice"), ChatID: input.ChatID}, nil
}

func (s *patchNoticeFailureSender) UpdateInteractiveMessage(_ context.Context, _ string, card Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, card)
	return nil
}

func TestPatchSuccessNoticeFailureDoesNotPatchAgain(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	sender := &patchNoticeFailureSender{nextMessageID: "om_canonical"}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	current := reviewProductInvariantTime
	dispatcher.now = func() time.Time { return current }
	firstJob := testReviewGatewayJob(current, "patch-notice-first")
	firstResult := fullReviewGatewayResultFixture()
	completeReviewPresentationReply(t, store, firstJob, firstResult, current)
	dispatcher.deliverPendingReplies(context.Background())

	current = current.Add(time.Minute)
	secondJob := testReviewGatewayJob(current, "patch-notice-second")
	secondResult := fullReviewGatewayResultFixture()
	secondResult.HeadSHA = "fedcba9876543210fedcba9876543210fedcba98"
	secondResult.SourceFingerprint = "source-fingerprint-8"
	secondResult.ResultCard = buildReviewGatewayResultCard(secondJob, secondResult, nil)
	completeReviewPresentationReply(t, store, secondJob, secondResult, current)
	sender.mu.Lock()
	sender.failNextNotice = true
	sender.mu.Unlock()
	dispatcher.deliverPendingReplies(context.Background())

	current = current.Add(5 * time.Minute)
	dispatcher.deliverPendingReplies(context.Background())
	sender.mu.Lock()
	updates := len(sender.updates)
	fullCards := 0
	textNotices := 0
	for _, input := range sender.inputs {
		if input.MsgType == "interactive" {
			fullCards++
		}
		if input.MsgType == "text" {
			textNotices++
		}
	}
	sender.mu.Unlock()
	if updates != 1 || fullCards != 1 || textNotices != 2 {
		t.Fatalf("patch was repeated after notice failure: updates=%d full_cards=%d text_attempts=%d", updates, fullCards, textNotices)
	}
}

func completeReviewPresentationReply(
	t *testing.T,
	store ReviewGatewayJobStore,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	at time.Time,
) {
	t.Helper()
	if len(result.ResultCard) == 0 {
		result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
	}
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "presentation-worker", Now: at, LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result.JobID = job.JobID
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
}
