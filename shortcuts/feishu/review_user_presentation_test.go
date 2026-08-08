package feishu

import (
	"context"
	"fmt"
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
		reviewDecisionDisplayName("pending"):                                 "待审查",
		reviewDecisionDisplayName("approved"):                                "已批准",
		reviewDecisionDisplayName("rejected"):                                "需要修改",
		reviewDecisionDisplayName("unknown"):                                 "待审查",
		reviewCollectionStatusDisplayName("complete", false):                 "完整",
		reviewCollectionStatusDisplayName("partial", true):                   "数据不完整",
		reviewNextStepDisplayName("assign_human_reviewer"):                   "建议分配人工审查者",
		reviewNextStepDisplayName("request_re_review_for_current_head"):      "建议基于当前版本重新审查",
		reviewCollaborationStatusDisplayName("unassigned"):                   "未领取",
		reviewCollaborationStatusDisplayName("reviewing"):                    "审查中",
		reviewCollaborationStatusDisplayName("archived"):                     "已归档",
		reviewWriteStatusDisplayName("pending_confirmation"):                 "待本地确认",
		reviewWriteStatusDisplayName("failed"):                               "执行失败",
		reviewWriteStatusDisplayName("write_disabled"):                       "未启用写操作",
		reviewWriteStatusDisplayName("dry_run"):                              "试运行完成",
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
		reviewWriteStatusDisplayName("new_internal_write_status"),
	} {
		if got != "待确认" {
			t.Fatalf("unknown presentation value leaked as %q", got)
		}
	}
	if got := reviewNextStepDisplayName("new_internal_next_step"); got != "" {
		t.Fatalf("unknown next-step should be hidden, got %q", got)
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
		{"清除 owner/repo PR #1 审查截止", "clear_review_deadline", ""},
		{"提交审查意见 owner/repo PR #1 good", "prepare_common_review", "good"},
		{"批准 owner/repo PR #1 good", "prepare_review_approve", "good"},
		{"需要修改 owner/repo PR #1 fix this", "prepare_review_reject", "fix this"},
		{"要求修改 owner/repo PR #1 fix this", "prepare_review_reject", "fix this"},
		{"请求修改 owner/repo PR #1 fix this", "prepare_review_reject", "fix this"},
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
	for _, action := range []string{"help", "read_review_context", "claim_review", "release_review", "set_review_deadline", "clear_review_deadline"} {
		if ack := formatReviewGatewayAcknowledgement(ReviewGatewayJob{Action: action, Repository: "owner/repo", PRNumber: 1}); ack != "" {
			t.Fatalf("ordinary action %s sent ACK %q", action, ack)
		}
	}
	tests := map[string]string{
		"prepare_common_review":  "提交审查意见",
		"prepare_review_approve": "批准 PR #1",
		"prepare_review_reject":  "需要修改 PR #1",
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

func TestReviewersAreCompleteSortedAndBoundedOnlyAtPresentation(t *testing.T) {
	reviewers := []ReviewGatewayReviewerView{
		{Reviewer: "approved-old", Decision: "approved", ReviewedAt: "2026-08-08T12:00:00Z"},
		{Reviewer: "comment-old", Decision: "common", ReviewedAt: "2026-08-08T12:01:00Z"},
		{Reviewer: "rejected-old", Decision: "rejected", ReviewedAt: "2026-08-08T12:02:00Z"},
		{Reviewer: "rejected-new", Decision: "rejected", ReviewedAt: "2026-08-08T12:03:00Z"},
		{Reviewer: "comment-new", Decision: "commented", ReviewedAt: "2026-08-08T12:04:00Z"},
		{Reviewer: "approved-new", Decision: "approved", ReviewedAt: "2026-08-08T12:05:00Z"},
	}
	for index := 0; index < 6; index++ {
		reviewers = append(reviewers, ReviewGatewayReviewerView{Reviewer: fmt.Sprintf("other-%d", index), Decision: "unknown"})
	}
	sortReviewGatewayReviewers(reviewers)
	wantPrefix := []string{"rejected-new", "rejected-old", "comment-new", "comment-old", "approved-new", "approved-old"}
	for index, want := range wantPrefix {
		if reviewers[index].Reviewer != want {
			t.Fatalf("reviewers[%d]=%q, want %q; all=%#v", index, reviewers[index].Reviewer, want, reviewers)
		}
	}
	if len(reviewers) != 12 {
		t.Fatalf("reviewer aggregation was truncated: %d", len(reviewers))
	}
	lines := reviewReviewerDisplayLines(reviewers, 80)
	if len(lines) >= len(reviewers) || !strings.HasPrefix(lines[len(lines)-1], "另有 ") {
		t.Fatalf("bounded presentation did not report omitted reviewers: %#v", lines)
	}
}

func TestNeedChangesIsTheOnlyPresentedRejectTerm(t *testing.T) {
	job, result, _ := reviewGatewayCardFixture()
	job.Action = "prepare_review_reject"
	plan := NewControlledReviewActionPlan(job, "gitlink-user", result.HeadSHA, result.SourceFingerprint, reviewActionReject, "请补充回归测试", time.Now().UTC())
	result.Action = job.Action
	result.ActionPlan = &plan
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok {
		t.Fatal("need-changes ActionPlan card unexpectedly downgraded")
	}
	ack := formatReviewGatewayAcknowledgement(job)
	completed := formatReviewWriteResultReply(ReviewWriteResult{
		Action: reviewActionReject, Repository: job.Repository, PRNumber: job.PRNumber,
		Status: "completed", MutationStatus: reviewMutationConfirmed, ReviewID: "132",
	})
	stale := formatReviewWriteResultReply(ReviewWriteResult{
		Action: reviewActionReject, Repository: job.Repository, PRNumber: job.PRNumber,
		Status: "stale", MutationStatus: reviewMutationNone,
	})
	for label, text := range map[string]string{"ack": ack, "card": cardJSON, "completed": completed, "stale": stale} {
		if !strings.Contains(text, "需要修改") {
			t.Fatalf("%s does not present need-changes wording: %s", label, text)
		}
		for _, forbidden := range []string{"要求修改", "请求修改", "REJECT REVIEW", "prepare_review_reject", "reviewActionReject", "rejected"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s leaked compatibility/internal term %q: %s", label, forbidden, text)
			}
		}
	}
	if !strings.Contains(completed, "已标记为需要修改") || !strings.Contains(completed, "PR 状态：保持开放") {
		t.Fatalf("completed need-changes reply lost open-PR semantics: %s", completed)
	}
}

func TestReviewGatewayCollaborationRepliesUseReadableIdentityAndActionCopy(t *testing.T) {
	item := ReviewCollaborationItem{
		Repository: "owner/repo", PRNumber: 1, CollaborationStatus: "reviewing",
		AssignedTo: "opaque-sensitive-user-id", AssignedDisplayName: "张三", DueAt: "2026-08-10",
	}
	item.ActionOutcome = "claimed"
	claim := formatReviewCollaborationReply(ReviewGatewayJob{Action: "claim_review"}, item)
	if claim != "PR #1 已由张三负责" || strings.Contains(claim, "协作状态") || strings.Contains(claim, item.AssignedTo) {
		t.Fatalf("claim reply = %q", claim)
	}
	item.ActionOutcome = "deadline_updated"
	deadline := formatReviewCollaborationReply(ReviewGatewayJob{Action: "set_review_deadline"}, item)
	if !strings.Contains(deadline, "审查截止时间已更新") || !strings.Contains(deadline, "2026-08-10") {
		t.Fatalf("deadline reply = %q", deadline)
	}
	item.ActionOutcome = "released"
	release := formatReviewCollaborationReply(ReviewGatewayJob{Action: "release_review"}, item)
	if release != "PR #1 已取消负责人" || strings.Contains(release, "协作状态") || strings.Contains(release, item.AssignedTo) {
		t.Fatalf("release reply = %q", release)
	}
	item.AssignedDisplayName = ""
	if got := reviewGatewayAssigneePlainLabel(item); got != "负责人信息待同步" || strings.Contains(got, reviewGatewayHashIdentifier(item.AssignedTo)) {
		t.Fatalf("plain assignee fallback leaked identity: %q", got)
	}
}

func TestCollaborationFailuresAlwaysProduceUserVisibleReply(t *testing.T) {
	tests := []struct {
		action string
		detail string
		want   string
	}{
		{"claim_review", "当前账号尚未绑定 GitLink 身份", "当前账号尚未绑定 GitLink 身份"},
		{"claim_review", "当前 GitLink 身份不是该仓库的协作者", "不是该仓库的协作者"},
		{"claim_review", "暂时无法确认当前 GitLink 身份的仓库协作者状态，请稍后重试", "请稍后重试"},
		{"claim_review", "PR #3 当前已由测试负责", "当前已由测试负责"},
		{"release_review", "当前 PR 由测试负责，只有当前负责人或协作管理员可以取消领取", "只有当前负责人"},
		{"set_review_deadline", "日期格式应为 YYYY-MM-DD", "YYYY-MM-DD"},
		{"set_review_deadline", "database is locked", "协作状态正忙"},
		{"read_review_context", "GitLink API request failed", "GitLink 暂时不可用"},
	}
	for _, test := range tests {
		reply := formatReviewGatewayFailureReply(ReviewGatewayJob{Action: test.action, PRNumber: 3}, test.detail)
		if strings.TrimSpace(reply) == "" || !strings.Contains(reply, test.want) {
			t.Fatalf("failure reply action=%s detail=%q => %q", test.action, test.detail, reply)
		}
	}
}

func TestReviewGatewayHelpPromotesChineseCommandsAndCurrentBoundaries(t *testing.T) {
	help := reviewGatewayHelpText()
	for _, required := range []string{
		"查看 <拥有者>/<仓库> PR #<编号>", "取消领取 <拥有者>/<仓库> PR #<编号>", "审查截止 <YYYY-MM-DD>",
		"清除 <拥有者>/<仓库>", "提交审查意见", "批准 <拥有者>/<仓库>", "需要修改 <拥有者>/<仓库>", "拒绝并关闭 <拥有者>/<仓库>", "合并 <拥有者>/<仓库>",
		"查看 muel/gitlink-feishu_agent PR #3",
		"需要修改”只提交审查结论，PR 保持开放",
	} {
		if !strings.Contains(help, required) {
			t.Fatalf("help missing %q: %s", required, help)
		}
	}
	for _, forbidden := range []string{"approve owner/repo", "reject owner/repo", "要求修改 owner/repo", "请求修改 owner/repo", "始终禁用"} {
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

func TestReviewGatewayWriteResultNeverReportsUnknownStateAsSuccess(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{status: "failed", want: "执行失败"},
		{status: "write_disabled", want: "未启用 GitLink 写操作"},
		{status: "dry_run", want: "试运行"},
		{status: "garbage_status", want: "结果暂无法确认"},
		{status: "future_new_status", want: "结果暂无法确认"},
		{status: "", want: "结果暂无法确认"},
	}
	for _, test := range tests {
		t.Run(firstNonEmpty(test.status, "empty"), func(t *testing.T) {
			reply := formatReviewWriteResultReply(ReviewWriteResult{
				Action: reviewActionApprove, Repository: "owner/repo", PRNumber: 2,
				Status: test.status, MutationStatus: reviewMutationNone,
			})
			if !strings.Contains(reply, test.want) {
				t.Fatalf("status %q reply missing %q: %s", test.status, test.want, reply)
			}
			for _, forbidden := range []string{"GitLink 写入已完成", "远程回读：验证通过", "待本地确认"} {
				if strings.Contains(reply, forbidden) {
					t.Fatalf("status %q reported success/pending via %q: %s", test.status, forbidden, reply)
				}
			}
		})
	}

	completed := formatReviewWriteResultReply(ReviewWriteResult{
		Action: reviewActionApprove, Repository: "owner/repo", PRNumber: 2,
		Status: "completed", MutationStatus: reviewMutationConfirmed,
	})
	if !strings.Contains(completed, "GitLink 写入已完成") || !strings.Contains(completed, "远程回读：验证通过") || strings.Contains(completed, "待本地确认") {
		t.Fatalf("explicit success reply = %s", completed)
	}
}

func TestTerminalOrUnknownActionPlanNeverShowsConfirmationControls(t *testing.T) {
	job, result, _ := reviewGatewayCardFixture()
	plan := NewControlledReviewActionPlan(job, "gitlink-user", result.HeadSHA, result.SourceFingerprint, reviewActionApprove, "done", time.Now().UTC())
	result.ActionPlan = &plan

	pendingJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok || !strings.Contains(pendingJSON, "待本地确认") || !strings.Contains(pendingJSON, "查看本地确认方式") || !strings.Contains(pendingJSON, "取消操作") {
		t.Fatalf("pending_confirmation lost confirmation controls: %s", pendingJSON)
	}

	tests := []struct {
		status string
		want   string
	}{
		{status: "completed", want: "已完成"},
		{status: "stale", want: "已失效"},
		{status: "cancelled", want: "已取消"},
		{status: "unknown", want: "暂无法确认"},
		{status: "failed", want: "执行失败"},
		{status: "write_disabled", want: "未启用 GitLink 写操作"},
		{status: "dry_run", want: "试运行"},
		{status: "future_status", want: "暂无法确认"},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			terminal := result
			terminal.WriteResult = &ReviewWriteResult{
				Action: reviewActionApprove, Repository: job.Repository, PRNumber: job.PRNumber,
				Status: test.status, MutationStatus: reviewMutationNone,
			}
			cardJSON, cardOK := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, terminal, nil))
			if !cardOK || !strings.Contains(cardJSON, test.want) {
				t.Fatalf("status %q card missing %q: %s", test.status, test.want, cardJSON)
			}
			for _, forbidden := range []string{"查看本地确认方式", "待本地确认", "取消操作", "GitLink 写入已完成", "远程回读验证通过"} {
				if strings.Contains(cardJSON, forbidden) {
					t.Fatalf("status %q exposed terminally invalid %q: %s", test.status, forbidden, cardJSON)
				}
			}
		})
	}

	for _, status := range []string{"failed", "future_plan_status", ""} {
		t.Run("stored_plan_"+firstNonEmpty(status, "empty"), func(t *testing.T) {
			stored := result
			storedPlan := plan
			storedPlan.Status = status
			stored.ActionPlan = &storedPlan
			stored.WriteResult = nil
			cardJSON, cardOK := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, stored, nil))
			if !cardOK || strings.Contains(cardJSON, "待本地确认") || strings.Contains(cardJSON, "查看本地确认方式") {
				t.Fatalf("stored plan status %q fell back to pending: %s", status, cardJSON)
			}
		})
	}
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
	if stored.PlanID != plan.PlanID || stored.RequestID != plan.RequestID || !strings.Contains(cardJSON, plan.PlanID) || strings.Contains(cardJSON, "操作编号") || strings.Contains(reply, plan.RequestID) {
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
