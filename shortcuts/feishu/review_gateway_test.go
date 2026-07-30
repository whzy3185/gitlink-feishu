package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func TestParseReviewGatewayIntent(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantRepo   string
		wantNumber int
	}{
		{input: "帮助", wantName: "help"},
		{input: "查看绑定", wantName: "show_binding"},
		{input: "查看待 Review", wantName: "read_review_queue"},
		{input: "查看 PR #431", wantName: "read_review_context", wantNumber: 431},
		{input: "领取 PR 431", wantName: "plan_claim_review", wantNumber: 431},
		{input: "生成 PR #431 Review 草稿", wantName: "generate_review_draft", wantNumber: 431},
		{input: "刷新 PR #431", wantName: "refresh_review_context", wantNumber: 431},
		{input: "绑定仓库 gitlink-org/gitlink-cli", wantName: "plan_bind_repository", wantRepo: "gitlink-org/gitlink-cli"},
		{input: "批准 PR #431", wantName: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := parseReviewGatewayIntent(test.input)
			if got.Name != test.wantName || got.Repository != test.wantRepo || got.PRNumber != test.wantNumber {
				t.Fatalf("parseReviewGatewayIntent(%q) = %#v", test.input, got)
			}
		})
	}
}

func TestReviewGatewayPlansBoundReadOnlyJobAndDeduplicatesMessage(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Bindings: []ReviewChatBinding{{
			ChatID:         "oc_review",
			Repository:     "gitlink-org/gitlink-cli",
			Enabled:        true,
			AllowedUserIDs: []string{"ou_owner"},
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	event := ReviewGatewayEvent{
		EventID:      "evt_1",
		MessageID:    "om_1",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看 PR #431",
		CreateTimeMs: now.Add(-time.Minute).UnixMilli(),
	}
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !receipt.Accepted || receipt.Job == nil {
		t.Fatalf("receipt not accepted: %#v", receipt)
	}
	if receipt.Job.MutatesGitLink || receipt.Job.Mode != "preview" || receipt.Job.Repository != "gitlink-org/gitlink-cli" {
		t.Fatalf("unsafe or incomplete job: %#v", receipt.Job)
	}
	if receipt.DedupeKey != "feishu:message:om_1" {
		t.Fatalf("dedupe key = %q", receipt.DedupeKey)
	}

	event.EventID = "evt_retry"
	duplicate, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("duplicate Plan: %v", err)
	}
	if duplicate.Accepted || !duplicate.Duplicate || duplicate.Reason != "duplicate_event" {
		t.Fatalf("duplicate receipt = %#v", duplicate)
	}
}

func TestReviewGatewayRejectsUnsupportedBindingSchema(t *testing.T) {
	_, err := NewReviewGateway(
		ReviewGatewayBindings{SchemaVersion: "feishu.review-bindings/v2"},
		ReviewGatewayConfig{},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), reviewGatewayBindingSchema) {
		t.Fatalf("unsupported binding schema error = %v", err)
	}
}

func TestReviewGatewayRejectsStaleUnauthorizedAndUnsupportedEvents(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:         "oc_review",
			Repository:     "gitlink-org/gitlink-cli",
			Enabled:        true,
			AllowedUserIDs: []string{"ou_owner"},
		}},
	}, ReviewGatewayConfig{
		Now:         func() time.Time { return now },
		StaleWindow: 10 * time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	base := ReviewGatewayEvent{
		EventID:      "evt",
		MessageID:    "om",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看待审查",
		CreateTimeMs: now.UnixMilli(),
	}

	stale := base
	stale.MessageID = "om_stale"
	stale.CreateTimeMs = now.Add(-11 * time.Minute).UnixMilli()
	assertReviewGatewayReason(t, gateway, stale, "stale_event")

	unauthorized := base
	unauthorized.MessageID = "om_unauthorized"
	unauthorized.UserID = "ou_other"
	assertReviewGatewayReason(t, gateway, unauthorized, "sender_not_allowed")

	unsupported := base
	unsupported.MessageID = "om_unsupported"
	unsupported.EventType = "url_verification"
	assertReviewGatewayReason(t, gateway, unsupported, "unsupported_event_type")
}

func TestReviewGatewayBindingPlanRequiresAdminAndDoesNotApply(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{}, ReviewGatewayConfig{
		AdminUserIDs: []string{"ou_admin"},
		Now:          func() time.Time { return now },
	}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	event := ReviewGatewayEvent{
		EventID:      "evt_bind",
		EventType:    "card_action",
		ChatID:       "oc_unbound",
		UserID:       "ou_admin",
		Content:      "绑定仓库 owner/repo",
		CreateTimeMs: now.UnixMilli(),
	}
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !receipt.Accepted || receipt.Bound || receipt.Job == nil || !receipt.Job.CollaborationMutation {
		t.Fatalf("binding plan receipt = %#v", receipt)
	}
	show := event
	show.EventID = "evt_show"
	show.Content = "查看绑定"
	assertReviewGatewayReason(t, gateway, show, "chat_not_bound")
}

func TestSQLiteReviewGatewayStorePersistsDedupeAndRedactsErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway.db")
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	reserved, err := store.Reserve("feishu:message:om_1", now, 24*time.Hour)
	if err != nil || !reserved {
		t.Fatalf("Reserve = %v, %v", reserved, err)
	}
	job := ReviewGatewayJob{
		SchemaVersion:  reviewGatewayJobSchema,
		JobID:          "job-test",
		DedupeKey:      "feishu:message:om_1",
		Status:         "queued",
		Action:         "read_review_context",
		Repository:     "owner/repo",
		PRNumber:       42,
		ChatID:         "oc_review",
		RequestedBy:    "ou_owner",
		CreatedAt:      now.Format(time.RFC3339),
		Mode:           "preview",
		MutatesGitLink: false,
	}
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	rawError := "GET https://user:pass@example.test/api?access_token=url-secret Authorization: Bearer bearer-secret Cookie: session=cookie-secret"
	if err := store.UpdateJobStatus(context.Background(), job.JobID, "failed", rawError); err != nil {
		t.Fatalf("UpdateJobStatus: %v", err)
	}
	var errorSummary string
	if err := store.db.QueryRow("SELECT error_summary FROM review_gateway_jobs WHERE job_id = ?", job.JobID).Scan(&errorSummary); err != nil {
		t.Fatalf("read error summary: %v", err)
	}
	for _, secret := range []string{"url-secret", "bearer-secret", "cookie-secret", "user:pass"} {
		if strings.Contains(errorSummary, secret) {
			t.Fatalf("error summary leaked %q: %s", secret, errorSummary)
		}
	}
	if !strings.Contains(errorSummary, "***") {
		t.Fatalf("error summary missing redaction marker: %s", errorSummary)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()
	reserved, err = reopened.Reserve("feishu:message:om_1", now.Add(time.Minute), 24*time.Hour)
	if err != nil {
		t.Fatalf("Reserve after restart: %v", err)
	}
	if reserved {
		t.Fatal("dedupe reservation was lost across SQLite restart")
	}
}

func TestReviewGatewayQueueRunsAsynchronously(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	store := NewMemoryReviewGatewayJobStore()
	completed := make(chan ReviewGatewayJobOutcome, 1)
	queue := NewReviewGatewayQueue(gateway, store, 1, func(outcome ReviewGatewayJobOutcome) {
		completed <- outcome
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.Run(ctx, func(_ context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
		return ReviewGatewayExecutionResult{
			SchemaVersion: reviewGatewayResultSchema,
			JobID:         job.JobID,
			Status:        "completed",
			Mode:          "preview",
		}, nil
	})

	receipt := queue.Enqueue(ctx, ReviewGatewayEvent{
		EventID:      "evt_queue",
		MessageID:    "om_queue",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看待审查",
		CreateTimeMs: now.UnixMilli(),
	})
	if !receipt.Accepted || receipt.Job == nil {
		t.Fatalf("queue receipt = %#v", receipt)
	}
	select {
	case outcome := <-completed:
		if outcome.Err != nil {
			t.Fatalf("queue handler: %v", outcome.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for asynchronous job")
	}
	store.mu.Lock()
	status := store.Status[receipt.Job.JobID]
	store.mu.Unlock()
	if status != "completed" {
		t.Fatalf("job status = %q", status)
	}
}

func TestPlanReviewSnapshotSyncProtectsPartialAndManualState(t *testing.T) {
	base := workflow.ReviewContext{
		CollectionStatus: "complete",
		CurrentHeadSHA:   "head-2",
		WorkItem: workflow.ReviewWorkItem{
			GitLinkState:      "open",
			SourceFingerprint: "fingerprint-2",
			SourceScope:       workflow.ReviewSourceScope{Complete: true},
		},
	}
	if got := PlanReviewSnapshotSync(base, nil); got.Action != "apply" || !got.PreserveManualFields || !got.OverwriteGitLinkFacts {
		t.Fatalf("complete changed snapshot plan = %#v", got)
	}
	partial := base
	partial.Partial = true
	partial.CollectionStatus = "partial"
	if got := PlanReviewSnapshotSync(partial, nil); got.Action != "preserve_previous" || got.OverwriteGitLinkFacts {
		t.Fatalf("partial snapshot plan = %#v", got)
	}
	merged := base
	merged.WorkItem.GitLinkState = "merged"
	if got := PlanReviewSnapshotSync(merged, nil); got.Action != "archive" || !got.Archive {
		t.Fatalf("merged snapshot plan = %#v", got)
	}
	previous := &ReviewSnapshotState{
		SourceFingerprint: "fingerprint-2",
		HeadSHA:           "head-2",
		Complete:          true,
	}
	if got := PlanReviewSnapshotSync(base, previous); got.Action != "unchanged" || !got.PreserveManualFields {
		t.Fatalf("unchanged snapshot plan = %#v", got)
	}
}

func TestReviewGatewaySDKEventNormalizationExcludesCardToken(t *testing.T) {
	message := reviewGatewayEventFromMessage(&larktypes.NormalizedMessage{
		EventID:      "evt_message",
		MessageID:    "om_message",
		ChatID:       "oc_review",
		ChatType:     "group",
		UserID:       "ou_owner",
		Content:      "查看 PR #42",
		CreateTimeMs: 123,
	})
	if message.EventType != "message" || message.MessageID != "om_message" || message.Content != "查看 PR #42" {
		t.Fatalf("normalized message = %#v", message)
	}

	mentioned := reviewGatewayEventFromMessage(&larktypes.NormalizedMessage{
		EventID:   "evt_mentioned",
		MessageID: "om_mentioned",
		ChatID:    "oc_review",
		ChatType:  "group",
		UserID:    "ou_owner",
		Content:   "@_user_1  查看 PR #42",
		Mentions: []larktypes.Mention{
			{Key: "@_user_1", OpenID: "ou_bot", IsBot: true},
		},
	})
	if mentioned.Content != "查看 PR #42" {
		t.Fatalf("bot mention was not removed from command: %#v", mentioned)
	}

	card, ok := reviewGatewayEventFromCardAction(&larktypes.CardActionEvent{
		EventID:   "evt_card",
		MessageID: "om_card",
		ChatID:    "oc_review",
		Token:     "must-not-be-copied",
		Operator:  larktypes.CardActionOperator{OpenID: "ou_owner"},
		Action: larktypes.CardActionPayload{
			Value: map[string]interface{}{"command": "刷新 PR #42"},
		},
	})
	if !ok || card.EventType != "card_action" || card.Content != "刷新 PR #42" {
		t.Fatalf("normalized card action = %#v, %v", card, ok)
	}
	encoded, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(encoded), "must-not-be-copied") {
		t.Fatalf("card callback token leaked into gateway event: %s", encoded)
	}
}

func TestReviewDraftPreviewUsesStableBoundedTemplate(t *testing.T) {
	reviewContext := workflow.ReviewContext{
		Repository:       "owner/repo",
		PullRequest:      42,
		CurrentVersionID: "7",
		CurrentPatchset:  workflow.ReviewContextPatchset{FilesCount: 3},
		Summary: workflow.ReviewCollaborationSummary{
			Decision:     "changes_pending",
			TotalReviews: 2,
			OpenThreads:  1,
		},
		ReviewerSummaries: []workflow.ReviewContextReviewerSummary{{
			ReviewerKey:     "alice",
			Actor:           "Alice",
			CurrentDecision: "approved",
		}},
		Threads: []workflow.ReviewContextThread{{
			Author:    "Bob",
			Content:   strings.Repeat("需", 450),
			State:     "open",
			Freshness: "current",
			Path:      "main.go",
			LineCode:  "12",
		}},
		WorkItem: workflow.ReviewWorkItem{
			Title:               "Improve review flow",
			Unknowns:            []string{"CI state unavailable"},
			RecommendedNextStep: "owner decision",
		},
	}
	draft := buildReviewDraftPreview(reviewContext)
	if draft.TemplateVersion != "review-draft/v1" || len(draft.ReviewerStates) != 1 || len(draft.OpenThreads) != 1 {
		t.Fatalf("draft = %#v", draft)
	}
	if len([]rune(draft.OpenThreads[0].Content)) != 401 {
		t.Fatalf("thread preview length = %d", len([]rune(draft.OpenThreads[0].Content)))
	}
}

func assertReviewGatewayReason(t *testing.T, gateway *ReviewGateway, event ReviewGatewayEvent, want string) {
	t.Helper()
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if receipt.Accepted || receipt.Reason != want {
		t.Fatalf("receipt = %#v, want reason %q", receipt, want)
	}
}

func TestSQLiteReviewGatewayStoreRecoversExpiredRunningJob(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway-recovery.db")
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	job := testReviewGatewayJob(now, "job-recovery")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-1",
		Now:           now,
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(claimed) != 1 || claimed[0].AttemptCount != 1 {
		t.Fatalf("first claim = %#v, %v", claimed, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	notExpired, err := reopened.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-2",
		Now:           now.Add(30 * time.Second),
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(notExpired) != 0 {
		t.Fatalf("unexpired lease was recovered: %#v, %v", notExpired, err)
	}
	recovered, err := reopened.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-2",
		Now:           now.Add(2 * time.Minute),
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(recovered) != 1 {
		t.Fatalf("expired job recovery = %#v, %v", recovered, err)
	}
	if recovered[0].AttemptCount != 2 || recovered[0].LeaseOwner != "worker-2" {
		t.Fatalf("recovered job metadata = %#v", recovered[0])
	}
}

func TestSQLiteReviewGatewayQueueAtomicallyDeduplicatesAndRecoversOrphanReservation(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-atomic.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(
		ReviewGatewayBindings{Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}}},
		ReviewGatewayConfig{Now: func() time.Time { return now }},
		store,
	)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 2, nil)
	queue.now = func() time.Time { return now }
	event := ReviewGatewayEvent{
		EventID:   "evt_atomic",
		MessageID: "om_atomic",
		EventType: "message",
		ChatID:    "oc_review",
		ChatType:  "group",
		UserID:    "ou_owner",
		Content:   "查看 PR #42",
	}
	first := queue.Enqueue(context.Background(), event)
	if !first.Accepted || first.Job == nil {
		t.Fatalf("first atomic enqueue = %#v", first)
	}
	second := queue.Enqueue(context.Background(), event)
	if second.Accepted || !second.Duplicate || second.Reason != "duplicate_event" {
		t.Fatalf("duplicate atomic enqueue = %#v", second)
	}

	orphanEvent := event
	orphanEvent.EventID = "evt_orphan"
	orphanEvent.MessageID = "om_orphan"
	orphanKey := reviewGatewayDedupeKey(orphanEvent)
	reserved, err := store.ReserveContext(context.Background(), orphanKey, now, 24*time.Hour)
	if err != nil || !reserved {
		t.Fatalf("create P2.0 orphan reservation = %v, %v", reserved, err)
	}
	recovered := queue.Enqueue(context.Background(), orphanEvent)
	if !recovered.Accepted || recovered.Job == nil {
		t.Fatalf("orphan reservation was not recovered atomically: %#v", recovered)
	}
	var jobCount int
	if err := store.db.QueryRow(
		"SELECT COUNT(*) FROM review_gateway_jobs WHERE dedupe_key=?",
		orphanKey,
	).Scan(&jobCount); err != nil {
		t.Fatalf("count recovered job: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("recovered job count = %d", jobCount)
	}
}

func TestSQLiteReviewGatewayStorePersistsResultAndReplyState(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-result.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "job-result")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker-result",
		Now:        now,
		Limit:      1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:    reviewGatewayResultSchema,
		JobID:            job.JobID,
		Status:           "completed",
		Mode:             "preview",
		Action:           job.Action,
		Repository:       job.Repository,
		PRNumber:         job.PRNumber,
		RequestedBy:      job.RequestedBy,
		ReadOnlyGitLink:  true,
		MutatesGitLink:   false,
		CompletedAt:      now.Format(time.RFC3339),
		CollectionStatus: "complete",
		HeadSHA:          "head-sha",
	}
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	var status, resultJSON, replyStatus string
	if err := store.db.QueryRow(
		"SELECT status, result_json, reply_status FROM review_gateway_jobs WHERE job_id=?",
		job.JobID,
	).Scan(&status, &resultJSON, &replyStatus); err != nil {
		t.Fatalf("read persisted result: %v", err)
	}
	if status != "completed" || replyStatus != "pending" || !strings.Contains(resultJSON, `"head_sha":"head-sha"`) {
		t.Fatalf("persisted state = status=%q reply=%q result=%s", status, replyStatus, resultJSON)
	}
	pending, err := store.ClaimPendingReplies(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "reply-worker",
		Now:        time.Now().UTC().Add(time.Second),
		Limit:      1,
	})
	if err != nil || len(pending) != 1 {
		t.Fatalf("ClaimPendingReplies = %#v, %v", pending, err)
	}
	if pending[0].Result.HeadSHA != "head-sha" || pending[0].Job.SourceMessageID != job.SourceMessageID {
		t.Fatalf("pending reply = %#v", pending[0])
	}
}

func TestSQLiteReviewGatewayStoreRetriesThenPublishesTerminalFailure(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-retry.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "job-retry")
	job.MaxAttempts = 2
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claim := func(at time.Time, owner string) ReviewGatewayJob {
		t.Helper()
		jobs, claimErr := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
			LeaseOwner: owner,
			Now:        at,
			Limit:      1,
		})
		if claimErr != nil || len(jobs) != 1 {
			t.Fatalf("ClaimReadyJobs(%s) = %#v, %v", owner, jobs, claimErr)
		}
		return jobs[0]
	}
	first := claim(now, "worker-1")
	failed := failedReviewGatewayResult(first, errors.New("temporary read failure"), now)
	willRetry, err := store.RetryOrFailJob(context.Background(), first, failed, failed.Error, now)
	if err != nil || !willRetry {
		t.Fatalf("first RetryOrFailJob = retry=%v err=%v", willRetry, err)
	}
	tooEarly, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker-early",
		Now:        now.Add(time.Second),
		Limit:      1,
	})
	if err != nil || len(tooEarly) != 0 {
		t.Fatalf("retry was claimable before next_attempt_at: %#v, %v", tooEarly, err)
	}
	second := claim(now.Add(3*time.Second), "worker-2")
	if second.AttemptCount != 2 {
		t.Fatalf("second attempt count = %d", second.AttemptCount)
	}
	failed = failedReviewGatewayResult(second, errors.New("terminal read failure"), now.Add(3*time.Second))
	willRetry, err = store.RetryOrFailJob(context.Background(), second, failed, failed.Error, now.Add(3*time.Second))
	if err != nil || willRetry {
		t.Fatalf("terminal RetryOrFailJob = retry=%v err=%v", willRetry, err)
	}
	pending, err := store.ClaimPendingReplies(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "reply-worker",
		Now:        now.Add(4 * time.Second),
		Limit:      1,
	})
	if err != nil || len(pending) != 1 {
		t.Fatalf("terminal failure reply = %#v, %v", pending, err)
	}
	if pending[0].Result.Status != "failed" || pending[0].Result.AttemptCount != 2 {
		t.Fatalf("terminal failure result = %#v", pending[0].Result)
	}
}

func TestSQLiteReviewGatewayStoreMigratesP20Schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway-migrate.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE review_gateway_events (
			dedupe_key TEXT PRIMARY KEY,
			received_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);
		CREATE TABLE review_gateway_jobs (
			job_id TEXT PRIMARY KEY,
			dedupe_key TEXT NOT NULL,
			status TEXT NOT NULL,
			action TEXT NOT NULL,
			repository TEXT,
			pr_number INTEGER,
			requested_by TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			error_summary TEXT
		);
	`)
	if err != nil {
		t.Fatalf("create P2.0 schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close P2.0 db: %v", err)
	}
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("migrate P2.0 schema: %v", err)
	}
	defer store.Close()
	var resultJSONColumn string
	if err := store.db.QueryRow(
		"SELECT name FROM pragma_table_info('review_gateway_jobs') WHERE name='result_json'",
	).Scan(&resultJSONColumn); err != nil {
		t.Fatalf("result_json migration missing: %v", err)
	}
}

func TestReviewGatewayHandlerBudgetStopsLockedSQLite(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-locked.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{}, store)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 1, nil)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("lock db: %v", err)
	}
	defer tx.Rollback()
	started := time.Now()
	err = handleReviewGatewayInbound(
		context.Background(),
		ReviewGatewayEvent{
			EventID:   "evt_locked",
			MessageID: "om_locked",
			EventType: "message",
			ChatID:    "oc_review",
			UserID:    "ou_owner",
			Content:   "查看 PR #42",
		},
		queue,
		nil,
		&reviewGatewayJSONOutput{writer: io.Discard},
		300*time.Millisecond,
		100*time.Millisecond,
	)
	elapsed := time.Since(started)
	if err == nil || !strings.Contains(err.Error(), "persistence failed") {
		t.Fatalf("locked handler error = %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("handler exceeded controlled budget: %v", elapsed)
	}
}

func TestReviewGatewayAsyncOutputDoesNotBlockHandler(t *testing.T) {
	writer := &blockingReviewGatewayWriter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := &reviewGatewayJSONOutput{writer: writer}
	output.Start(ctx, 1)
	if !output.TryEmit(map[string]string{"event": "first"}) {
		t.Fatal("first output was not accepted")
	}
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	if !output.TryEmit(map[string]string{"event": "second"}) {
		t.Fatal("second output should fit buffered queue")
	}
	started := time.Now()
	if output.TryEmit(map[string]string{"event": "third"}) {
		t.Fatal("third output should be dropped instead of blocking")
	}
	if time.Since(started) > 50*time.Millisecond {
		t.Fatal("non-blocking output path blocked")
	}
	close(writer.release)
}

func TestReviewGatewayReplyDispatcherRepliesOnceToOriginalMessage(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store := NewMemoryReviewGatewayJobStore()
	job := testReviewGatewayJob(now, "job-reply")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker",
		Now:        now,
		Limit:      1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:    reviewGatewayResultSchema,
		JobID:            job.JobID,
		Status:           "completed",
		Mode:             "preview",
		Action:           job.Action,
		Repository:       job.Repository,
		PRNumber:         job.PRNumber,
		RequestedBy:      job.RequestedBy,
		ReadOnlyGitLink:  true,
		MutatesGitLink:   false,
		CompletedAt:      now.Format(time.RFC3339),
		CollectionStatus: "complete",
		ReviewStage:      "triaged",
		Decision:         "pending",
	}
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(
		sender,
		store,
		&reviewGatewayJSONOutput{writer: io.Discard},
		2,
	)
	dispatcher.now = func() time.Time { return now.Add(time.Second) }
	dispatcher.deliverPendingReplies(context.Background())
	dispatcher.deliverPendingReplies(context.Background())

	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.inputs) != 1 {
		t.Fatalf("reply count = %d, want 1", len(sender.inputs))
	}
	if sender.inputs[0].ReplyMessageID != job.SourceMessageID {
		t.Fatalf("reply target = %q", sender.inputs[0].ReplyMessageID)
	}
	if !strings.Contains(sender.inputs[0].Text, "GitLink 写入：0") {
		t.Fatalf("reply boundary missing: %s", sender.inputs[0].Text)
	}
	store.mu.Lock()
	replyStatus := store.ReplyStatus[job.JobID]
	store.mu.Unlock()
	if replyStatus != "sent" {
		t.Fatalf("reply status = %q", replyStatus)
	}
}

func TestReviewGatewayChannelPolicyRequiresPreboundGroup(t *testing.T) {
	bindings := ReviewGatewayBindings{Bindings: []ReviewChatBinding{
		{ChatID: "oc_enabled", Repository: "owner/repo", Enabled: true},
		{ChatID: "oc_disabled", Repository: "owner/other", Enabled: false},
	}}
	channel := newFeishuReviewGatewayChannel("app-id", "app-secret", bindings)
	policy := channel.GetPolicy()
	if len(policy.GroupAllowlist) != 1 || policy.GroupAllowlist[0] != "oc_enabled" {
		t.Fatalf("group allowlist = %#v", policy.GroupAllowlist)
	}
	if policy.RequireMention == nil || !*policy.RequireMention || policy.DMMode != "disabled" {
		t.Fatalf("channel policy = %#v", policy)
	}
}

func testReviewGatewayJob(now time.Time, id string) ReviewGatewayJob {
	return ReviewGatewayJob{
		SchemaVersion:   reviewGatewayJobSchema,
		JobID:           id,
		DedupeKey:       "feishu:message:" + id,
		Status:          "queued",
		Mode:            "preview",
		Action:          "read_review_context",
		Repository:      "owner/repo",
		PRNumber:        42,
		ChatID:          "oc_review",
		RequestedBy:     "ou_owner",
		SourceEventID:   "evt_" + id,
		SourceMessageID: "om_" + id,
		CreatedAt:       now.Format(time.RFC3339Nano),
		MutatesGitLink:  false,
		MaxAttempts:     3,
		NextAttemptAt:   now.Format(time.RFC3339Nano),
	}
}

type recordingReviewGatewaySender struct {
	mu     sync.Mutex
	inputs []larktypes.SendInput
}

func (s *recordingReviewGatewaySender) Send(_ context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, *input)
	return &larktypes.SendResult{
		MessageID: "om_reply",
		ChatID:    input.ChatID,
	}, nil
}

type blockingReviewGatewayWriter struct {
	started chan struct{}
	release chan struct{}
}

func (w *blockingReviewGatewayWriter) Write(payload []byte) (int, error) {
	select {
	case w.started <- struct{}{}:
	default:
	}
	<-w.release
	return len(payload), nil
}
