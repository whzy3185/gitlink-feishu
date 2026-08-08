package feishu

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewPresentationMappings(t *testing.T) {
	tests := map[string]string{
		reviewPRStateDisplayName("open"):                                     "开放中",
		reviewPRStateDisplayName("closed"):                                   "已关闭",
		reviewPRStateDisplayName("merged"):                                   "已合并",
		reviewStageDisplayName("triaged"):                                    "已进入审查流程",
		reviewStageDisplayName("waiting_for_re_review"):                      "等待重新审查",
		reviewDecisionDisplayName("pending"):                                 "待决定",
		reviewDecisionDisplayName("approved"):                                "已批准",
		reviewDecisionDisplayName("rejected"):                                "需修改",
		reviewDecisionDisplayName("unknown"):                                 "待确认",
		reviewCollectionStatusDisplayName("complete", false):                 "完整",
		reviewCollectionStatusDisplayName("partial", true):                   "数据不完整",
		reviewNextStepDisplayName("assign_human_reviewer"):                   "建议分配人工审查者",
		reviewNextStepDisplayName("request_re_review_for_current_head"):      "建议基于当前版本重新审查",
		reviewCollaborationStatusDisplayName("unassigned"):                   "未领取",
		reviewCollaborationStatusDisplayName("reviewing"):                    "审查中",
		reviewCollaborationStatusDisplayName("archived"):                     "已归档",
		reviewWriteStatusDisplayName("pending_confirmation"):                 "待本地确认",
		reviewReconciliationDisplayName("verified by GitLink GET read-back"): "验证通过",
	}
	for got, want := range tests {
		if got != want {
			t.Fatalf("presentation mapping = %q, want %q", got, want)
		}
	}
	for _, got := range []string{
		reviewPRStateDisplayName("new_internal_state"),
		reviewStageDisplayName("new_internal_stage"),
		reviewDecisionDisplayName("new_internal_decision"),
		reviewNextStepDisplayName("new_internal_next_step"),
	} {
		if got != "待确认" {
			t.Fatalf("unknown presentation value leaked as %q", got)
		}
	}
}

func TestReviewGatewayChineseCommandsAndCompatibilityAliases(t *testing.T) {
	tests := []struct {
		input, action, argument string
	}{
		{"查看 owner/repo PR #1", "read_review_context", ""},
		{"领取 owner/repo PR #1", "claim_review", ""},
		{"取消领取 owner/repo PR #1", "release_review", ""},
		{"设置 owner/repo PR #1 审查截止 2026-08-10", "set_review_deadline", "2026-08-10"},
		{"提交审查意见 owner/repo PR #1 good", "prepare_common_review", "good"},
		{"批准 owner/repo PR #1 good", "prepare_review_approve", "good"},
		{"要求修改 owner/repo PR #1 fix this", "prepare_review_reject", "fix this"},
		{"拒绝并关闭 owner/repo PR #1 reason", "prepare_reject_close", "reason"},
		{"合并 owner/repo PR #1", "prepare_merge", ""},
		{"review owner/repo#1 good", "prepare_common_review", "good"},
		{"approve owner/repo#1 good", "prepare_review_approve", "good"},
		{"reject owner/repo#1 fix", "prepare_review_reject", "fix"},
		{"refuse owner/repo#1 reason", "prepare_reject_close", "reason"},
		{"merge owner/repo#1", "prepare_merge", ""},
		{"释放 owner/repo PR #1", "release_review", ""},
		{"设置 owner/repo PR #1 截止 2026-08-10", "set_review_deadline", "2026-08-10"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := parseReviewGatewayIntent(test.input)
			if got.Name != test.action || got.Repository != "owner/repo" || got.PRNumber != 1 || got.Argument != test.argument {
				t.Fatalf("parseReviewGatewayIntent(%q) = %#v", test.input, got)
			}
		})
	}
}

func TestReviewGatewayUserPresentationDoesNotLeakInternalFields(t *testing.T) {
	job, result, item := reviewGatewayCardFixture()
	result.PullRequest.RecommendedNextStep = "assign_human_reviewer"
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, &item))
	if !ok {
		t.Fatal("ordinary PR card unexpectedly downgraded")
	}
	assertReviewPresentationDoesNotContain(t, cardJSON,
		"partial=false", "assign_human_reviewer", "waiting_for_re_review", "triaged")

	plan := NewControlledReviewActionPlan(job, "gitlink-user", result.HeadSHA, result.SourceFingerprint, reviewActionMerge, "", time.Now().UTC())
	result.ActionPlan = &plan
	result.WriteResult = nil
	result.Action = "prepare_merge"
	cardJSON, ok = safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok {
		t.Fatal("pending plan card unexpectedly downgraded")
	}
	assertReviewPresentationDoesNotContain(t, cardJSON,
		"HIGH RISK", "No remote body for this lifecycle action", "ActionPlan", "Request ID",
		"Feishu actor", "Expected head", "Content / Reason", "Local command", "pending_confirmation",
		"APPROVE", "REJECT REVIEW", "REJECT & CLOSE", "MERGE")
	if !strings.Contains(cardJSON, "高风险操作") || !strings.Contains(cardJSON, "待本地确认") {
		t.Fatalf("pending plan card lost Chinese safety copy: %s", cardJSON)
	}
}

func TestReviewGatewayPlanLifecyclePresentationIsMutuallyExclusive(t *testing.T) {
	job, result, _ := reviewGatewayCardFixture()
	plan := NewControlledReviewActionPlan(job, "gitlink-user", result.HeadSHA, result.SourceFingerprint, reviewActionApprove, "done", time.Now().UTC())
	result.ActionPlan = &plan

	pendingJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !strings.Contains(pendingJSON, "待本地确认") || !strings.Contains(pendingJSON, "尚未修改 GitLink") || strings.Contains(pendingJSON, "已完成并通过") {
		t.Fatalf("pending lifecycle mixed terminal state: %s", pendingJSON)
	}

	completed := result
	completed.WriteResult = &ReviewWriteResult{Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber, Status: "completed", MutationStatus: reviewMutationConfirmed, ReviewID: "131", RequestID: plan.RequestID}
	completedJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, completed, nil))
	if !strings.Contains(completedJSON, "已完成并通过 GitLink 回读验证") || !strings.Contains(completedJSON, "#131") || strings.Contains(completedJSON, "待本地确认") || strings.Contains(completedJSON, "查看本地确认方式") {
		t.Fatalf("completed lifecycle mixed pending state: %s", completedJSON)
	}
	completedFromStore := result
	completedPlan := plan
	completedPlan.Status = "completed"
	completedFromStore.ActionPlan = &completedPlan
	completedFromStore.WriteResult = nil
	completedStoreJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, completedFromStore, nil))
	if !strings.Contains(completedStoreJSON, "GitLink 操作已完成并通过回读验证") || strings.Contains(completedStoreJSON, "尚未修改 GitLink") {
		t.Fatalf("stored completed plan mixed pending boundary: %s", completedStoreJSON)
	}

	stale := result
	stale.WriteResult = &ReviewWriteResult{Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber, Status: "stale", MutationStatus: reviewMutationNone}
	staleJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, stale, nil))
	if !strings.Contains(staleJSON, "操作计划已失效") || !strings.Contains(staleJSON, "本次未修改 GitLink") || strings.Contains(staleJSON, "操作已完成") {
		t.Fatalf("stale lifecycle = %s", staleJSON)
	}

	cancelled := result
	cancelled.WriteResult = &ReviewWriteResult{Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber, Status: "cancelled", MutationStatus: reviewMutationNone}
	cancelledJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, cancelled, nil))
	if !strings.Contains(cancelledJSON, "操作已取消") || !strings.Contains(cancelledJSON, "本次未修改 GitLink") || strings.Contains(cancelledJSON, "待本地确认") {
		t.Fatalf("cancelled lifecycle = %s", cancelledJSON)
	}

	unknown := result
	unknown.WriteResult = &ReviewWriteResult{Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber, Status: "unknown_needs_reconciliation", MutationStatus: reviewMutationPossible}
	unknownJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, unknown, nil))
	if !strings.Contains(unknownJSON, "结果暂无法确认") || !strings.Contains(unknownJSON, "停止自动重试") || strings.Contains(unknownJSON, "操作已完成") {
		t.Fatalf("unknown lifecycle = %s", unknownJSON)
	}
}

func TestReviewGatewayAcknowledgementsAreActionSpecific(t *testing.T) {
	tests := map[string]string{
		"read_review_context":    "PR 查询请求",
		"claim_review":           "正在领取 PR #1",
		"release_review":         "正在取消 PR #1",
		"set_review_deadline":    "正在更新 PR #1 的审查截止时间",
		"prepare_common_review":  "提交审查意见",
		"prepare_review_approve": "批准 PR #1",
		"prepare_review_reject":  "要求修改 PR #1",
		"prepare_reject_close":   "拒绝并关闭 PR #1",
		"prepare_merge":          "合并 PR #1",
	}
	for action, want := range tests {
		ack := formatReviewGatewayAcknowledgement(ReviewGatewayJob{Action: action, Repository: "owner/repo", PRNumber: 1})
		if !strings.Contains(ack, want) || strings.Contains(ack, "只读 Review 请求") {
			t.Fatalf("ack %s = %q", action, ack)
		}
	}
}

func TestReviewGatewayCollaborationRepliesUseReadableIdentityAndActionCopy(t *testing.T) {
	item := ReviewCollaborationItem{
		Repository: "owner/repo", PRNumber: 1, CollaborationStatus: "reviewing",
		AssignedTo: "opaque-sensitive-user-id", AssignedDisplayName: "张三", DueAt: "2026-08-10",
	}
	claim := formatReviewCollaborationReply(ReviewGatewayJob{Action: "claim_review"}, item)
	if !strings.Contains(claim, "领取成功") || !strings.Contains(claim, "负责人：张三") || !strings.Contains(claim, "协作状态：审查中") || strings.Contains(claim, item.AssignedTo) {
		t.Fatalf("claim reply = %q", claim)
	}
	deadline := formatReviewCollaborationReply(ReviewGatewayJob{Action: "set_review_deadline"}, item)
	if !strings.Contains(deadline, "审查截止时间已更新") || !strings.Contains(deadline, "2026-08-10") {
		t.Fatalf("deadline reply = %q", deadline)
	}
	release := formatReviewCollaborationReply(ReviewGatewayJob{Action: "release_review"}, item)
	if !strings.Contains(release, "已取消领取") || !strings.Contains(release, "负责人：未领取") || !strings.Contains(release, "协作状态：未领取") || strings.Contains(release, item.AssignedTo) {
		t.Fatalf("release reply = %q", release)
	}
	item.AssignedDisplayName = ""
	if got := reviewGatewayAssigneePlainLabel(item); got != "飞书成员" || strings.Contains(got, reviewGatewayHashIdentifier(item.AssignedTo)) {
		t.Fatalf("plain assignee fallback leaked identity: %q", got)
	}
}

func TestReviewGatewayHelpPromotesChineseCommandsAndCurrentBoundaries(t *testing.T) {
	help := reviewGatewayHelpText()
	for _, required := range []string{
		"查看 owner/repo PR #123", "取消领取 owner/repo PR #123", "审查截止 YYYY-MM-DD",
		"提交审查意见", "批准 owner/repo", "要求修改 owner/repo", "拒绝并关闭 owner/repo", "合并 owner/repo",
		"要求修改”只提交审查结论，PR 保持开放",
	} {
		if !strings.Contains(help, required) {
			t.Fatalf("help missing %q: %s", required, help)
		}
	}
	for _, forbidden := range []string{"approve owner/repo", "reject owner/repo", "始终禁用"} {
		if strings.Contains(help, forbidden) {
			t.Fatalf("help still promotes obsolete copy %q: %s", forbidden, help)
		}
	}
}

func TestReviewGatewayResultReplyDoesNotLeakInternalVocabulary(t *testing.T) {
	job := ReviewGatewayJob{Repository: "owner/repo", PRNumber: 2}
	result := ReviewGatewayExecutionResult{WriteResult: &ReviewWriteResult{
		Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber,
		Status: "completed", MutationStatus: reviewMutationConfirmed, ReviewID: "131",
		Reconciliation: "verified by GitLink GET read-back",
	}}
	reply := formatReviewGatewayResultReply(job, result)
	if !strings.Contains(reply, "PR #2 已批准") || !strings.Contains(reply, "远程回读：验证通过") {
		t.Fatalf("completed reply lost product wording: %s", reply)
	}
	assertReviewPresentationDoesNotContain(t, reply,
		"APPROVE", "REJECT REVIEW", "REJECT & CLOSE", "MERGE", "completed",
		"verified by GitLink GET read-back", "assign_human_reviewer")
}

func TestReviewGatewayActionPlanIdentityIsConsistentAcrossStoreCardAndReply(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "presentation-plan.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 8, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "presentation-plan")
	job.SourceMessageID = "om_same_message"
	job.Action = "prepare_review_reject"
	plan, err := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "gitlink-user", "head", "fingerprint", reviewActionReject, "fix", now))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	result := ReviewGatewayExecutionResult{Status: "completed", Action: job.Action, Repository: job.Repository, PRNumber: job.PRNumber, ActionPlan: &plan}
	cardJSON, _ := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	reply := formatReviewGatewayResultReply(job, result)
	if stored.PlanID != plan.PlanID || stored.RequestID != plan.RequestID || !strings.Contains(cardJSON, plan.PlanID) || !strings.Contains(cardJSON, plan.RequestID) || !strings.Contains(reply, plan.RequestID) {
		t.Fatalf("plan identity drifted: stored=%#v card=%s reply=%s", stored, cardJSON, reply)
	}
	replayed, err := store.CreateReviewActionPlan(context.Background(), NewControlledReviewActionPlan(job, "gitlink-user", "head", "fingerprint", reviewActionReject, "fix", now))
	if err != nil || replayed.PlanID != plan.PlanID || replayed.RequestID != plan.RequestID {
		t.Fatalf("same job produced a second logical plan: %#v, %v", replayed, err)
	}
}

func assertReviewPresentationDoesNotContain(t *testing.T, value string, forbidden ...string) {
	t.Helper()
	for _, item := range forbidden {
		if strings.Contains(value, item) {
			t.Fatalf("user presentation leaked %q: %s", item, value)
		}
	}
}
