package feishu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

var reviewProductInvariantTime = time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)

func fullReviewGatewayResultFixture() ReviewGatewayExecutionResult {
	return ReviewGatewayExecutionResult{
		SchemaVersion:     reviewGatewayResultSchema,
		JobID:             "job-product-invariant",
		Status:            "completed",
		Mode:              "preview",
		Action:            "read_review_context",
		Repository:        "owner/repo",
		PRNumber:          42,
		RequestedBy:       "ou_owner",
		ReadOnlyGitLink:   true,
		MutatesGitLink:    false,
		CompletedAt:       reviewProductInvariantTime.Format(time.RFC3339),
		CollectionStatus:  "complete",
		Partial:           false,
		HeadSHA:           "0123456789abcdef0123456789abcdef01234567",
		SourceFingerprint: "source-fingerprint-7",
		ReviewStage:       "human_reviewing",
		Decision:          "pending",
		GitLinkState:      "open",
		ReviewCount:       3,
		ThreadCount:       4,
		OpenThreadCount:   2,
		PullRequest: &ReviewGatewayPullRequestView{
			Title:               "Preserve complete PR context",
			Author:              "alice",
			BaseBranch:          "main",
			HeadBranch:          "feature/card-continuity",
			GitLinkURL:          "https://www.gitlink.org.cn/owner/repo/pulls/42",
			PatchsetID:          "patchset-7",
			FilesCount:          12,
			CommitsCount:        4,
			Additions:           328,
			Deletions:           91,
			RiskLevel:           "medium",
			Unknowns:            []string{"CI status unavailable"},
			RecommendedNextStep: "resolve two open threads",
			Reviewers: []ReviewGatewayReviewerView{
				{Reviewer: "bob", Decision: "changes_pending"},
			},
		},
	}
}

func fullReviewCollaborationItemFixture() ReviewCollaborationItem {
	result := fullReviewGatewayResultFixture()
	return ReviewCollaborationItem{
		SchemaVersion:       reviewCollaborationItemSchema,
		PRKey:               reviewCollaborationScopeKey("test", "oc_review", "owner/repo", 42),
		InstallationID:      "test",
		Repository:          "owner/repo",
		PRNumber:            42,
		ChatID:              "oc_review",
		ReviewStage:         result.ReviewStage,
		Decision:            result.Decision,
		CollectionStatus:    result.CollectionStatus,
		HeadSHA:             result.HeadSHA,
		SourceFingerprint:   result.SourceFingerprint,
		AssignedTo:          "ou_owner",
		CollaborationStatus: "reviewing",
		DueAt:               "2026-08-10",
		UpdatedBy:           "ou_owner",
		UpdatedAt:           reviewProductInvariantTime.Format(time.RFC3339Nano),
	}
}

func productInvariantCardText(card Card) string {
	parts := []string{}
	var visit func(interface{})
	visit = func(value interface{}) {
		switch typed := value.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if key == "content" {
					if text, ok := typed[key].(string); ok {
						parts = append(parts, text)
						continue
					}
				}
				visit(typed[key])
			}
		case []interface{}:
			for _, item := range typed {
				visit(item)
			}
		}
	}
	visit(map[string]interface{}(card))
	return strings.Join(parts, "\n")
}

func assertFullPRCardFacts(t *testing.T, card Card) {
	t.Helper()
	text := productInvariantCardText(card)
	expected := map[string]string{
		"PR title":          "Preserve complete PR context",
		"author":            "alice",
		"target branch":     "**目标分支**\nmain",
		"source branch":     "**来源分支**\nfeature/card-continuity",
		"short head SHA":    "0123456789ab",
		"files and changes": "12 个文件 · +328 / -91 · 4 次提交",
		"review count":      "**Review**\n3 条",
		"reviewer":          "bob（需要修改）",
		"assignee field":    "**负责人**",
	}
	missing := []string{}
	for label, value := range expected {
		if !strings.Contains(text, value) {
			missing = append(missing, fmt.Sprintf("%s=%q", label, value))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("card lost complete PR facts: %s", strings.Join(missing, ", "))
	}
	for _, forbidden := range []string{"未解决讨论", "协作状态", "数据状态", "风险", "本次操作未修改 GitLink"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("card leaked removed field %q", forbidden)
		}
	}
}

func prepareProductInvariantCollaborationStore(t *testing.T) (*SQLiteReviewGatewayStore, ReviewGatewayJob) {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "product-invariant.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	job := testReviewGatewayJob(reviewProductInvariantTime, "product-context")
	result := fullReviewGatewayResultFixture()
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, result, reviewProductInvariantTime); err != nil {
		_ = store.Close()
		t.Fatalf("seed complete PR facts: %v", err)
	}
	return store, job
}

func executeProductInvariantAction(
	t *testing.T,
	store *SQLiteReviewGatewayStore,
	job ReviewGatewayJob,
	action,
	argument string,
	at time.Time,
) ReviewGatewayExecutionResult {
	t.Helper()
	job.JobID = "job-product-" + action
	job.Action = action
	job.Argument = argument
	job.CollaborationAuthorized = true
	executor := &ReviewGatewayExecutor{
		Collaboration: store,
		Now:           func() time.Time { return at },
		Collaborators: &staticReviewCollaboratorReader{collaborators: []ReviewRepositoryCollaborator{{Login: "gitlink-reviewer"}}},
		IdentityBindings: []ReviewIdentityBinding{{
			InstallationID: job.InstallationID,
			FeishuUserID:   job.RequestedBy,
			GitLinkLogin:   "gitlink-reviewer",
			Enabled:        true,
		}},
	}
	result, err := executor.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execute %s: %v", action, err)
	}
	return result
}

func TestReviewCardPreservesPRFactsAfterClaim(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	initial := fullReviewGatewayResultFixture()
	initialItem := fullReviewCollaborationItemFixture()
	assertFullPRCardFacts(t, buildReviewGatewayResultCard(job, initial, &initialItem))

	result := executeProductInvariantAction(t, store, job, "claim_review", "", reviewProductInvariantTime.Add(time.Minute))
	assertFullPRCardFacts(t, result.ResultCard)
}

func TestReviewCardPreservesPRFactsAfterRelease(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	claimJob := job
	claimJob.JobID = "job-product-release-claim"
	claimJob.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), claimJob, reviewProductInvariantTime.Add(time.Minute)); err != nil {
		t.Fatalf("prepare claim: %v", err)
	}

	result := executeProductInvariantAction(t, store, job, "release_review", "", reviewProductInvariantTime.Add(2*time.Minute))
	assertFullPRCardFacts(t, result.ResultCard)
}

func TestReviewCardPreservesPRFactsAfterDeadlineChange(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	claimJob := job
	claimJob.JobID = "job-product-deadline-claim"
	claimJob.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), claimJob, reviewProductInvariantTime.Add(time.Minute)); err != nil {
		t.Fatalf("prepare claim: %v", err)
	}

	result := executeProductInvariantAction(t, store, job, "set_review_deadline", "2026-08-10", reviewProductInvariantTime.Add(2*time.Minute))
	assertFullPRCardFacts(t, result.ResultCard)
	if text := productInvariantCardText(result.ResultCard); !strings.Contains(text, "2026-08-10") {
		t.Fatalf("deadline card lost updated deadline")
	}
}

func TestCollaborationActionWithoutPresentationMustFailClosed(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "missing-presentation.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "missing-presentation")
	job.Action = "claim_review"
	job.CollaborationAuthorized = true
	executor := &ReviewGatewayExecutor{
		Collaboration: store,
		Now:           func() time.Time { return reviewProductInvariantTime },
		Collaborators: &staticReviewCollaboratorReader{collaborators: []ReviewRepositoryCollaborator{{Login: "gitlink-reviewer"}}},
		IdentityBindings: []ReviewIdentityBinding{{
			InstallationID: job.InstallationID,
			FeishuUserID:   job.RequestedBy,
			GitLinkLogin:   "gitlink-reviewer",
			Enabled:        true,
		}},
	}
	result, executeErr := executor.Execute(context.Background(), job)
	if executeErr == nil {
		t.Fatalf("collaboration action without a complete PR presentation succeeded and produced a card with %d elements", len(result.ResultCard))
	}
}

func productInvariantBundle(result ReviewGatewayExecutionResult, item ReviewCollaborationItem) ReviewCollaborationBundle {
	item.HeadSHA = result.HeadSHA
	item.SourceFingerprint = result.SourceFingerprint
	item.ReviewStage = result.ReviewStage
	item.Decision = result.Decision
	item.CollectionStatus = result.CollectionStatus
	bundle := BuildReviewCollaborationBundle(item)
	bundle.Card = buildReviewGatewayResultCard(
		ReviewGatewayJob{Repository: result.Repository, PRNumber: result.PRNumber},
		result,
		&item,
	)
	return bundle
}

func deliverProductInvariantReply(
	t *testing.T,
	store *SQLiteReviewGatewayStore,
	dispatcher *ReviewGatewayReplyDispatcher,
	id string,
	at time.Time,
	result ReviewGatewayExecutionResult,
	item ReviewCollaborationItem,
) ReviewCollaborationBundle {
	t.Helper()
	job := testReviewGatewayJob(at, id)
	job.SourceMessageID = "om_source_" + id
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob %s: %v", id, err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "product-worker", Now: at, LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs %s = %#v, %v", id, claimed, err)
	}
	bundle := productInvariantBundle(result, item)
	result.JobID = job.JobID
	result.Action = job.Action
	result.Collaboration = &bundle
	result.ResultCard = bundle.Card
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob %s: %v", id, err)
	}
	dispatcher.deliverPendingReplies(context.Background())
	return bundle
}

func productInvariantSenderCounts(sender *recordingReviewGatewaySender) (fullCards, textNotices, updates int) {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	for _, input := range sender.inputs {
		if input.MsgType == "interactive" && strings.Contains(input.Card, "Preserve complete PR context") {
			fullCards++
		}
		if input.MsgType == "text" && strings.TrimSpace(input.Text) != "" {
			textNotices++
		}
	}
	return fullCards, textNotices, len(sender.updates)
}

func newProductInvariantReplyHarness(t *testing.T) (*SQLiteReviewGatewayStore, *recordingReviewGatewaySender, *ReviewGatewayReplyDispatcher) {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "canonical-card.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 2)
	dispatcher.now = func() time.Time { return reviewProductInvariantTime }
	return store, sender, dispatcher
}

func TestRepeatedPRQueryDoesNotSendDuplicateFullCard(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "repeat-1", reviewProductInvariantTime, result, item)
	deliverProductInvariantReply(t, store, dispatcher, "repeat-2", reviewProductInvariantTime.Add(time.Minute), result, item)
	fullCards, textNotices, _ := productInvariantSenderCounts(sender)
	if fullCards != 1 || textNotices != 1 {
		t.Fatalf("second query sent duplicate full interactive card: full_cards=%d text_notices=%d", fullCards, textNotices)
	}
}

func TestChangedPRQueryPatchesCanonicalCardOnly(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "changed-1", reviewProductInvariantTime, result, item)
	result.HeadSHA = "fedcba9876543210fedcba9876543210fedcba98"
	result.SourceFingerprint = "source-fingerprint-8"
	deliverProductInvariantReply(t, store, dispatcher, "changed-2", reviewProductInvariantTime.Add(time.Minute), result, item)
	fullCards, textNotices, updates := productInvariantSenderCounts(sender)
	if updates != 1 || fullCards != 1 || textNotices != 1 {
		t.Fatalf("changed query did not use PATCH plus one lightweight notice: updates=%d full_cards=%d text_notices=%d", updates, fullCards, textNotices)
	}
}

func TestUnchangedPRQueryDoesNotCreateAnotherFullCard(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	deliverProductInvariantReply(t, store, dispatcher, "unchanged-1", reviewProductInvariantTime, result, item)
	deliverProductInvariantReply(t, store, dispatcher, "unchanged-2", reviewProductInvariantTime.Add(time.Minute), result, item)
	fullCards, textNotices, updates := productInvariantSenderCounts(sender)
	if updates != 0 || fullCards != 1 || textNotices != 1 {
		t.Fatalf("unchanged query copied the full card: updates=%d full_cards=%d text_notices=%d", updates, fullCards, textNotices)
	}
}

func TestTenRepeatedQueriesStillHaveOneCanonicalCard(t *testing.T) {
	store, sender, dispatcher := newProductInvariantReplyHarness(t)
	defer store.Close()
	result := fullReviewGatewayResultFixture()
	item := fullReviewCollaborationItemFixture()
	var bundle ReviewCollaborationBundle
	for index := 0; index < 10; index++ {
		bundle = deliverProductInvariantReply(
			t, store, dispatcher, fmt.Sprintf("ten-%02d", index),
			reviewProductInvariantTime.Add(time.Duration(index)*time.Minute), result, item,
		)
	}
	fullCards, textNotices, updates := productInvariantSenderCounts(sender)
	state, err := store.GetReviewResourceState(context.Background(), bundle.UniqueKey, "feishu_card")
	if err != nil {
		t.Fatalf("GetReviewResourceState: %v", err)
	}
	if fullCards != 1 || textNotices != 9 || updates != 0 || state.RemoteID != "om_reply" {
		t.Fatalf("ten queries did not preserve one canonical card: full_cards=%d text_notices=%d updates=%d remote_id=%q", fullCards, textNotices, updates, state.RemoteID)
	}
}

func insertLegacyProductInvariantItem(
	t *testing.T,
	store *SQLiteReviewGatewayStore,
	chatID string,
	installations []string,
	resourceTypes []string,
) {
	t.Helper()
	for _, installationID := range installations {
		if _, err := store.db.Exec(`INSERT INTO chat_repository_bindings (
			chat_id, installation_id, repository, enabled, updated_at
		) VALUES (?, ?, 'owner/repo', 1, '2026-08-04T08:00:00Z')`, chatID, installationID); err != nil {
			t.Fatalf("insert binding %s: %v", installationID, err)
		}
	}
	if _, err := store.db.Exec(`INSERT INTO review_collaboration_items (
		pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
	) VALUES (
		'owner/repo#42', 'owner/repo', 42, ?, 'human_reviewing', 'pending',
		'complete', 'head-legacy', 'fingerprint-legacy', 'ou_legacy',
		'reviewing', '2026-08-10', 0, 'ou_legacy', '2026-08-04T08:00:00Z'
	)`, chatID); err != nil {
		t.Fatalf("insert legacy collaboration: %v", err)
	}
	legacyKey := stableKey("review-work-item", "owner/repo", "42")
	for _, resourceType := range resourceTypes {
		if _, err := store.db.Exec(`INSERT INTO review_collaboration_resources (
			work_item_key, resource_type, remote_id, content_fingerprint, updated_at
		) VALUES (?, ?, ?, 'legacy-fingerprint', '2026-08-04T08:00:00Z')`,
			legacyKey, resourceType, "remote_legacy_"+resourceType,
		); err != nil {
			t.Fatalf("insert legacy %s mapping: %v", resourceType, err)
		}
	}
}

func scopedProductInvariantResourceKey(installationID, chatID string) string {
	return stableKey(
		"review-work-item",
		reviewCollaborationScopeKey(installationID, chatID, "owner/repo", 42),
	)
}

func TestLegacyCardMappingWithoutTrustedChatIsNotMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "untrusted-card.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	insertLegacyProductInvariantItem(t, store, "oc_untrusted", []string{"installation-a"}, []string{"feishu_card"})
	_ = store.Close()

	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("run migration: %v", err)
	}
	defer store.Close()
	state, err := store.GetReviewResourceState(
		context.Background(), scopedProductInvariantResourceKey("installation-a", "oc_untrusted"), "feishu_card",
	)
	if err != nil {
		t.Fatalf("read scoped card mapping: %v", err)
	}
	if state.RemoteID != "" {
		t.Fatalf("untrusted legacy card remote_id leaked into chat scope: remote_id=%q", state.RemoteID)
	}
}

func TestLegacyCardMappingWithAmbiguousInstallationIsNotMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous-card.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	insertLegacyProductInvariantItem(
		t, store, "oc_ambiguous", []string{"installation-a", "installation-b"}, []string{"feishu_card"},
	)
	_ = store.Close()

	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("run migration: %v", err)
	}
	defer store.Close()
	for _, installationID := range []string{"installation-a", "installation-b"} {
		state, err := store.GetReviewResourceState(
			context.Background(), scopedProductInvariantResourceKey(installationID, "oc_ambiguous"), "feishu_card",
		)
		if err != nil {
			t.Fatalf("read %s scoped card: %v", installationID, err)
		}
		if state.RemoteID != "" {
			t.Fatalf("ambiguous installation selected %s and leaked card %q", installationID, state.RemoteID)
		}
	}
	legacyKey := stableKey("review-work-item", "owner/repo", "42")
	legacy, err := store.GetReviewResourceState(context.Background(), legacyKey, "feishu_card")
	if err != nil || legacy.RemoteID == "" {
		t.Fatalf("ambiguous legacy mapping was not retained: %#v, %v", legacy, err)
	}
}

func TestLegacyCardMappingWithVerifiedChatCanBeMigrated(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "verified-card.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	item := fullReviewCollaborationItemFixture()
	if _, err := store.db.Exec(`INSERT INTO chat_repository_bindings (
		chat_id, installation_id, repository, enabled, updated_at
	) VALUES (?, ?, ?, 1, ?)`, item.ChatID, item.InstallationID, item.Repository, item.UpdatedAt); err != nil {
		t.Fatalf("insert verified binding: %v", err)
	}
	if _, err := store.db.Exec(`INSERT INTO review_collaboration_items (
		pr_key, repository, pr_number, chat_id, updated_at
	) VALUES (?, ?, ?, ?, ?)`, reviewCollaborationPRKey(item.Repository, item.PRNumber),
		item.Repository, item.PRNumber, item.ChatID, item.UpdatedAt); err != nil {
		t.Fatalf("insert verified legacy item: %v", err)
	}
	if _, err := store.db.Exec(`INSERT INTO review_collaboration_states (
		collaboration_key, installation_id, chat_id, repository, pr_number,
		assigned_to, assigned_display_name, collaboration_status, due_at, updated_by, updated_at
	) VALUES (?, ?, ?, ?, ?, '', '', 'unassigned', '', '', ?)`,
		item.PRKey, item.InstallationID, item.ChatID, item.Repository, item.PRNumber, item.UpdatedAt,
	); err != nil {
		t.Fatalf("insert verified scoped state: %v", err)
	}
	legacyKey := stableKey("review-work-item", "owner/repo", "42")
	if _, err := store.db.Exec(`INSERT INTO review_collaboration_resources (
		work_item_key, resource_type, remote_id, content_fingerprint, updated_at
	) VALUES (?, 'feishu_card', 'om_verified_card', 'verified-fingerprint', ?)`, legacyKey, item.UpdatedAt); err != nil {
		t.Fatalf("insert verified legacy card: %v", err)
	}
	if err := store.ScanLegacyReviewResourceMigrations(context.Background(), reviewProductInvariantTime); err != nil {
		t.Fatalf("scan verified card: %v", err)
	}
	plans, err := store.ListReviewResourceMigrations(context.Background(), "", ReviewResourceCard, item.InstallationID)
	if err != nil || len(plans) != 1 {
		t.Fatalf("verified card plans=%#v err=%v", plans, err)
	}
	verification := LegacyReviewResourceVerificationInput{
		ExpectedInstallation: item.InstallationID,
		ExpectedScope:        ReviewResourceScopeChat,
		ExpectedChatID:       item.ChatID,
		Method:               "fixture_verified",
		Actor:                "test-operator",
		Confirmed:            true,
	}
	if _, err := store.VerifyLegacyReviewResourceMigration(
		context.Background(), plans[0].MigrationID, verification,
		fixtureLegacyReviewResourceVerifier{}, reviewProductInvariantTime.Add(time.Minute),
	); err != nil {
		t.Fatalf("verify legacy card: %v", err)
	}
	if _, err := store.ApplyLegacyReviewResourceMigration(
		context.Background(), plans[0].MigrationID, "test-operator", reviewProductInvariantTime.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("apply verified card: %v", err)
	}
	state, err := store.GetReviewResourceState(
		context.Background(), stableKey("review-work-item", item.PRKey), "feishu_card",
	)
	if err != nil || state.RemoteID != "om_verified_card" {
		t.Fatalf("verified card was not migrated: %#v, %v", state, err)
	}
}

func TestLegacyBaseDocTaskMigrationUsesExplicitPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource-policy.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	resourceTypes := []string{"feishu_bitable", "feishu_doc", "feishu_task"}
	insertLegacyProductInvariantItem(t, store, "oc_policy", []string{"installation-a"}, resourceTypes)
	_ = store.Close()

	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("run migration: %v", err)
	}
	defer store.Close()
	for _, resourceType := range resourceTypes {
		state, err := store.GetReviewResourceState(
			context.Background(), scopedProductInvariantResourceKey("installation-a", "oc_policy"), resourceType,
		)
		if err != nil {
			t.Fatalf("read %s mapping: %v", resourceType, err)
		}
		if state.RemoteID != "" {
			t.Fatalf("legacy %s migrated without an explicit chat/installation scope policy: remote_id=%q", resourceType, state.RemoteID)
		}
	}
}

func TestLegacyMigrationDoesNotOverwriteExistingScopedCard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing-scoped-card.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	insertLegacyProductInvariantItem(t, store, "oc_existing", []string{"installation-a"}, []string{"feishu_card"})
	scopedKey := scopedProductInvariantResourceKey("installation-a", "oc_existing")
	if err := store.SaveReviewResourceState(
		context.Background(), scopedKey, "feishu_card", "om_new_card", "new-fingerprint", reviewProductInvariantTime,
	); err != nil {
		t.Fatalf("save existing scoped card: %v", err)
	}
	_ = store.Close()

	for pass := 0; pass < 2; pass++ {
		store, err = OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("migration pass %d: %v", pass+1, err)
		}
		state, readErr := store.GetReviewResourceState(context.Background(), scopedKey, "feishu_card")
		if readErr != nil || state.RemoteID != "om_new_card" {
			t.Fatalf("migration pass %d overwrote scoped card: %#v, %v", pass+1, state, readErr)
		}
		_ = store.Close()
	}
}

type productInvariantReplyPersistFailureStore struct {
	*MemoryReviewGatewayJobStore
	markReplySentErr error
}

func (s *productInvariantReplyPersistFailureStore) MarkReplySent(
	context.Context,
	string,
	string,
	time.Time,
) error {
	return s.markReplySentErr
}

func TestReplySendSuccessAndLocalPersistFailureDoesNotBlindlyResend(t *testing.T) {
	base := NewMemoryReviewGatewayJobStore()
	store := &productInvariantReplyPersistFailureStore{
		MemoryReviewGatewayJobStore: base,
		markReplySentErr:            errors.New("simulated reply persistence failure"),
	}
	job := testReviewGatewayJob(reviewProductInvariantTime, "reply-persist-failure")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "product-worker", Now: reviewProductInvariantTime, LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := fullReviewGatewayResultFixture()
	result.JobID = job.JobID
	result.ResultCard = buildReviewGatewayResultCard(job, result, nil)
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 1)
	currentTime := reviewProductInvariantTime
	dispatcher.now = func() time.Time { return currentTime }
	dispatcher.deliverPendingReplies(context.Background())
	currentTime = currentTime.Add(2 * time.Minute)
	dispatcher.deliverPendingReplies(context.Background())

	sender.mu.Lock()
	sendCount := len(sender.inputs)
	sender.mu.Unlock()
	store.mu.Lock()
	replyStatus := store.ReplyStatus[job.JobID]
	store.mu.Unlock()
	if sendCount != 1 || (replyStatus != "unknown" && replyStatus != "needs_reconciliation") {
		t.Fatalf("reply was sent %d times after MarkReplySent failure; reply_status=%q, want one send and unknown/reconciliation", sendCount, replyStatus)
	}
}
