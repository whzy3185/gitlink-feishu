package feishu

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func seedReviewReconciliationCandidate(t *testing.T, store *SQLiteReviewGatewayStore, cardStatus, messageID string, enableSubscription bool) {
	t.Helper()
	if enableSubscription {
		if _, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
			InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
			AddGroups: []string{ReviewSubscriptionGroupPulls},
		}, reviewEventTestTime); err != nil {
			t.Fatal(err)
		}
	}
	nowText := reviewGatewayTimestamp(reviewEventTestTime)
	if _, err := store.db.Exec(`INSERT INTO review_pr_presentations (
		presentation_key, installation_id, repository, pr_number, schema_version,
		gitlink_state, collection_status, partial, source_fingerprint,
		content_fingerprint, source_completed_at, updated_at
	) VALUES (?, 'installation-a', 'owner/repo', 42, ?, 'open', 'complete', 0, 'source-42', 'content-42', ?, ?)`,
		reviewPRPresentationKey("installation-a", "owner/repo", 42), reviewPRPresentationSchema, nowText, nowText); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO chat_pr_presentations (
		presentation_key, app_scope, installation_id, chat_id, repository, pr_number,
		canonical_message_id, content_fingerprint, card_status, created_at, updated_at
	) VALUES (?, ?, 'installation-a', 'chat-a', 'owner/repo', 42, ?, 'content-42', ?, ?, ?)`,
		stableKey("test-chat-presentation", cardStatus, messageID), reviewGatewayAppScope,
		messageID, cardStatus, nowText, nowText); err != nil {
		t.Fatal(err)
	}
}

func newReviewReconciliationHarness(t *testing.T, cardStatus, messageID string, subscription bool) (*SQLiteReviewGatewayStore, *recordingReviewEventQueue, *ReviewReconciliationScheduler) {
	t.Helper()
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	seedReviewReconciliationCandidate(t, store, cardStatus, messageID, subscription)
	queue := &recordingReviewEventQueue{}
	scheduler := &ReviewReconciliationScheduler{
		Store: store, Queue: queue, Interval: 20 * time.Minute,
		Now: func() time.Time { return reviewEventTestTime.Add(2 * time.Hour) },
	}
	return store, queue, scheduler
}

func TestReconciliationSelectsActiveCanonicalCards(t *testing.T) {
	store, _, _ := newReviewReconciliationHarness(t, "active", "om_active", true)
	candidates, err := store.ListReviewReconciliationCandidates(context.Background())
	if err != nil || len(candidates) != 1 || candidates[0].CanonicalMessageHash == "" {
		t.Fatalf("candidates=%+v err=%v", candidates, err)
	}
}

func TestReconciliationSkipsArchivedCards(t *testing.T) {
	store, _, _ := newReviewReconciliationHarness(t, "archived", "om_archived", true)
	candidates, err := store.ListReviewReconciliationCandidates(context.Background())
	if err != nil || len(candidates) != 0 {
		t.Fatalf("archived candidates=%+v err=%v", candidates, err)
	}
}

func TestReconciliationSkipsDisabledSubscriptions(t *testing.T) {
	store, _, _ := newReviewReconciliationHarness(t, "active", "om_active", false)
	candidates, err := store.ListReviewReconciliationCandidates(context.Background())
	if err != nil || len(candidates) != 0 {
		t.Fatalf("disabled-subscription candidates=%+v err=%v", candidates, err)
	}
}

func TestReconciliationSkipsMissingCanonicalMessage(t *testing.T) {
	store, _, _ := newReviewReconciliationHarness(t, "active", "", true)
	candidates, err := store.ListReviewReconciliationCandidates(context.Background())
	if err != nil || len(candidates) != 0 {
		t.Fatalf("missing-message candidates=%+v err=%v", candidates, err)
	}
}

func TestReconciliationDryRunCreatesNoJob(t *testing.T) {
	store, queue, scheduler := newReviewReconciliationHarness(t, "active", "om_active", true)
	result, err := scheduler.RunOnce(context.Background(), true)
	var cursors int
	_ = store.db.QueryRow(`SELECT COUNT(*) FROM review_reconciliation_cursors`).Scan(&cursors)
	if err != nil || len(result.Candidates) != 1 || result.Enqueued != 0 || len(queue.jobs) != 0 || cursors != 0 {
		t.Fatalf("dry run=%+v jobs=%d cursors=%d err=%v", result, len(queue.jobs), cursors, err)
	}
}

func TestReconciliationTimeBucketIsIdempotent(t *testing.T) {
	_, queue, scheduler := newReviewReconciliationHarness(t, "active", "om_active", true)
	first, err := scheduler.RunOnce(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := scheduler.RunOnce(context.Background(), false)
	if err != nil || first.Enqueued != 1 || second.Enqueued != 0 || second.Duplicates != 1 || len(queue.jobs) != 1 {
		t.Fatalf("bucket results: first=%+v second=%+v jobs=%d err=%v", first, second, len(queue.jobs), err)
	}
}

func TestReconciliationJobDoesNotNotifyChat(t *testing.T) {
	_, queue, scheduler := newReviewReconciliationHarness(t, "active", "om_active", true)
	_, _ = scheduler.RunOnce(context.Background(), false)
	if len(queue.jobs) != 1 || queue.jobs[0].NotifyChat || queue.jobs[0].SourceMessageID != "" || queue.jobs[0].MutatesGitLink {
		t.Fatalf("reconciliation job=%+v", queue.jobs)
	}
}

func deliverReconciliationResult(
	t *testing.T, store *SQLiteReviewGatewayStore, dispatcher *ReviewGatewayReplyDispatcher,
	id string, at time.Time, result ReviewGatewayExecutionResult, item ReviewCollaborationItem,
) {
	t.Helper()
	job := testReviewGatewayJob(at, id)
	job.SourceMessageID = ""
	job.SourceEventID = reviewReconciliationCursorKey(job.InstallationID, job.ChatID, job.Repository, job.PRNumber)
	job.NotificationMode = ReviewNotificationCanonicalOnly
	job.NotifyChat = false
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "reconciliation-worker", Now: at, LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim reconciliation job=%+v err=%v", claimed, err)
	}
	bundle := productInvariantBundle(result, item)
	result.JobID, result.Action, result.Collaboration, result.ResultCard = job.JobID, job.Action, &bundle, bundle.Card
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatal(err)
	}
	dispatcher.deliverPendingReplies(context.Background())
}

func TestUnchangedReconciliationDoesNotPatchCard(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result, item := fullReviewGatewayResultFixture(), fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "reconcile-seed", reviewProductInvariantTime, result, item)
	beforeCards, beforeNotices, beforePatches := productInvariantSenderCounts(sender)
	deliverReconciliationResult(t, store, dispatcher, "reconcile-unchanged", reviewProductInvariantTime.Add(time.Minute), result, item)
	afterCards, afterNotices, afterPatches := productInvariantSenderCounts(sender)
	if beforeCards != afterCards || beforeNotices != afterNotices || beforePatches != afterPatches {
		t.Fatalf("unchanged reconciliation wrote remotely: before=%d/%d/%d after=%d/%d/%d", beforeCards, beforeNotices, beforePatches, afterCards, afterNotices, afterPatches)
	}
}

func TestChangedReconciliationUsesCanonicalCardPatch(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result, item := fullReviewGatewayResultFixture(), fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "reconcile-change-seed", reviewProductInvariantTime, result, item)
	result.PullRequest.Title = "Preserve complete PR context after reconciliation"
	result.SourceFingerprint = "source-after-reconciliation"
	deliverReconciliationResult(t, store, dispatcher, "reconcile-changed", reviewProductInvariantTime.Add(time.Minute), result, item)
	fullCards, notices, patches := productInvariantSenderCounts(sender)
	if fullCards != 1 || notices != 0 || patches != 1 {
		t.Fatalf("changed reconciliation lifecycle: cards=%d notices=%d patches=%d", fullCards, notices, patches)
	}
}

func TestReconciliationCursorSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reconciliation-restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	configuration := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{InstallationID: "installation-a", OperationMode: "collaborate", AllowedRepositories: []string{"owner/repo"}, Enabled: true}},
		Bindings: []ReviewChatBinding{{
			ChatID: "chat-a", InstallationID: "installation-a", Repositories: []string{"owner/repo"},
			DefaultRepository: "owner/repo", Enabled: true, AdminUserIDs: []string{"admin-a"},
		}},
	}
	if err := store.SyncReviewGatewayConfiguration(context.Background(), configuration, "restart-test", reviewEventTestTime); err != nil {
		t.Fatal(err)
	}
	seedReviewReconciliationCandidate(t, store, "active", "om_restart", true)
	queue := &recordingReviewEventQueue{}
	scheduler := &ReviewReconciliationScheduler{Store: store, Queue: queue, Interval: 20 * time.Minute, Now: func() time.Time { return reviewEventTestTime.Add(time.Hour) }}
	if _, err := scheduler.RunOnce(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cursors, err := store.ListReviewReconciliationCursors(context.Background())
	if err != nil || len(cursors) != 1 || cursors[0].LastEnqueuedAt == "" {
		t.Fatalf("reopened cursors=%+v err=%v", cursors, err)
	}
}
