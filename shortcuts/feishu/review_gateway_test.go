package feishu

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
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
	completed := make(chan error, 1)
	queue := NewReviewGatewayQueue(gateway, store, 1, func(_ ReviewGatewayJob, err error) {
		completed <- err
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.Run(ctx, func(context.Context, ReviewGatewayJob) error { return nil })

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
	case err := <-completed:
		if err != nil {
			t.Fatalf("queue handler: %v", err)
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
