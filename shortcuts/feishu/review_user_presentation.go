package feishu

import (
	"fmt"
	"strings"
	"time"
)

// This file is the single presentation boundary for Feishu-facing text. The
// durable protocol, JSON, logs, ActionPlan store and GitLink API keep their
// stable English values; cards, acknowledgements, replies and help use these
// mappings instead.

func reviewActionDisplayName(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case reviewActionCommon, "prepare_common_review":
		return "提交审查意见"
	case reviewActionApprove, "prepare_review_approve":
		return "批准"
	case reviewActionReject, "prepare_review_reject":
		return "需要修改"
	case reviewActionRejectClose, "prepare_reject_close":
		return "拒绝并关闭"
	case reviewActionMerge, "prepare_merge":
		return "合并"
	default:
		return "待确认"
	}
}

func reviewActionCompletedTitle(action string, number int) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case reviewActionCommon:
		return fmt.Sprintf("PR #%d 审查意见已提交", number)
	case reviewActionApprove:
		return fmt.Sprintf("PR #%d 已批准", number)
	case reviewActionReject:
		return fmt.Sprintf("PR #%d 已标记为需要修改", number)
	case reviewActionRejectClose:
		return fmt.Sprintf("PR #%d 已拒绝并关闭", number)
	case reviewActionMerge:
		return fmt.Sprintf("PR #%d 已合并", number)
	default:
		return fmt.Sprintf("PR #%d 操作已完成", number)
	}
}

func reviewPRStateDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return "开放中"
	case "closed":
		return "已关闭"
	case "merged":
		return "已合并"
	case "draft":
		return "草稿"
	default:
		return "待确认"
	}
}

func reviewStageDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "triaged", "discovered":
		return "已进入审查流程"
	case "assigned", "human_reviewing", "agent_reviewing", "reviewing":
		return "审查中"
	case "waiting_for_contributor":
		return "等待贡献者修改"
	case "waiting_for_re_review", "re_review_required":
		return "等待重新审查"
	case "ready_for_decision":
		return "等待最终决定"
	case "review_requested":
		return "已请求审查"
	case "unreviewed":
		return "尚未审查"
	case "parked":
		return "暂缓处理"
	case "closed":
		return "已关闭"
	case "merged":
		return "已合并"
	default:
		return "待确认"
	}
}

func reviewDecisionDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return "待审查"
	case "common", "commented":
		return "已提交审查意见"
	case "approved":
		return "已批准"
	case "rejected", "changes_pending", "blocked":
		return "需要修改"
	case "stale", "outdated":
		return "待重新审查"
	case "none":
		return "暂无"
	case "merged":
		return "已合并"
	case "closed":
		return "已关闭"
	default:
		return "待审查"
	}
}

func reviewCollaborationStatusDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "unassigned":
		return "未领取"
	case "reviewing":
		return "审查中"
	case "archived":
		return "已归档"
	case "waiting":
		return "等待处理"
	default:
		return ""
	}
}

func reviewRiskDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "critical":
		return "高风险"
	case "medium":
		return "中风险"
	case "low":
		return "低风险"
	default:
		return "待确认"
	}
}

func reviewCollectionStatusDisplayName(value string, partial bool) string {
	if partial {
		return "数据不完整"
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "complete":
		return "完整"
	case "partial":
		return "数据不完整"
	case "failed":
		return "获取失败"
	case "pending":
		return "获取中"
	default:
		return "待确认"
	}
}

func reviewNextStepDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "assign_human_reviewer":
		return "建议分配人工审查者"
	case "request_re_review_for_current_head":
		return "建议基于当前版本重新审查"
	case "wait_for_contributor":
		return "等待贡献者完成修改"
	case "owner_decision", "ready_for_owner_review":
		return "建议由维护团队完成最终复核"
	case "none":
		return "暂无"
	case "refresh_pr":
		return "刷新 PR 状态后重试"
	case "review_current_head":
		return "请重新审查当前版本"
	case "wait_author_changes":
		return "等待作者修改后重新审查"
	case "fix_failed_checks":
		return "修复失败的检查项"
	case "resolve_merge_conflicts":
		return "解决合并冲突"
	case "wait_remaining_reviews":
		return "等待其余审查结论"
	case "wait_explicit_decision":
		return "等待明确审查结论"
	case "wait_maintainer_merge":
		return "等待具备权限的维护者合并"
	case "wait_checks":
		return "等待检查完成"
	case "wait_review":
		return "等待审查"
	default:
		return ""
	}
}

func reviewWriteStatusDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending_confirmation", "executing":
		return "待本地确认"
	case "completed", "verified":
		return "已完成"
	case "cancelled":
		return "已取消"
	case "stale":
		return "已失效"
	case "unknown", "unknown_needs_reconciliation":
		return "结果待核对"
	case "failed":
		return "执行失败"
	case "write_disabled":
		return "未启用写操作"
	case "dry_run":
		return "试运行完成"
	default:
		return "待确认"
	}
}

func reviewReconciliationDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "verified", "verified by gitlink get read-back":
		return "验证通过"
	case "not_required", "":
		return "无需核对"
	case "pending", "required", "unknown", "unknown_needs_reconciliation":
		return "等待人工核对"
	default:
		return "待确认"
	}
}

func reviewReviewerDecisionDisplayName(value string) string {
	return reviewDecisionDisplayName(value)
}

func reviewReviewerDisplayLine(reviewer ReviewGatewayReviewerView) string {
	name := firstNonEmpty(strings.TrimSpace(reviewer.Reviewer), "GitLink 用户")
	decision := reviewReviewerDecisionDisplayName(reviewer.Decision)
	if reviewedAt, ok := parseReviewGatewayReviewTime(reviewer.ReviewedAt); ok {
		china := time.FixedZone("Asia/Shanghai", 8*60*60)
		return fmt.Sprintf("%s（%s · %s）", name, decision, reviewedAt.In(china).Format("2006-01-02 15:04"))
	}
	return fmt.Sprintf("%s（%s）", name, decision)
}

func reviewReviewerDisplayLines(reviewers []ReviewGatewayReviewerView, maxRunes int) []string {
	if len(reviewers) == 0 {
		return []string{"无"}
	}
	lines := make([]string, 0, len(reviewers))
	used := 0
	for _, reviewer := range reviewers {
		line := reviewReviewerDisplayLine(reviewer)
		if maxRunes > 0 && used+len([]rune(line)) > maxRunes {
			break
		}
		lines = append(lines, line)
		used += len([]rune(line))
	}
	if omitted := len(reviewers) - len(lines); omitted > 0 {
		lines = append(lines, fmt.Sprintf("另有 %d 名审查者", omitted))
	}
	return lines
}

func reviewActionRiskNotice(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case reviewActionMerge:
		return "**高风险操作：最终确认后将合并此 PR，并修改目标分支内容。**"
	case reviewActionRejectClose:
		return "**高风险操作：最终确认后将拒绝并关闭此 PR。**"
	case reviewActionApprove:
		return "最终确认后将向 GitLink 提交“批准”审查结果。"
	case reviewActionReject:
		return "最终确认后将提交“需要修改”审查结果，PR 保持开放。"
	case reviewActionCommon:
		return "最终确认后将向 GitLink 提交一条普通审查意见。"
	default:
		return ""
	}
}

func reviewGatewayHelpText() string {
	return strings.Join([]string{
		"GitLink PR Review 助手",
		"",
		"查询与协作",
		"@gitlink 查看 <拥有者>/<仓库> PR #<编号>",
		"@gitlink 领取 <拥有者>/<仓库> PR #<编号>",
		"@gitlink 取消领取 <拥有者>/<仓库> PR #<编号>",
		"@gitlink 设置 <拥有者>/<仓库> PR #<编号> 审查截止 <YYYY-MM-DD>",
		"@gitlink 清除 <拥有者>/<仓库> PR #<编号> 审查截止",
		"",
		"受控写操作",
		"@gitlink 提交审查意见 <拥有者>/<仓库> PR #<编号> <意见>",
		"@gitlink 批准 <拥有者>/<仓库> PR #<编号> <说明>",
		"@gitlink 需要修改 <拥有者>/<仓库> PR #<编号> <原因>",
		"@gitlink 拒绝并关闭 <拥有者>/<仓库> PR #<编号> <原因>",
		"@gitlink 合并 <拥有者>/<仓库> PR #<编号>",
		"",
		"示例：",
		"@gitlink 查看 muel/gitlink-feishu_agent PR #3",
		"@gitlink 需要修改 muel/gitlink-feishu_agent PR #3 请补充异常场景测试",
		"",
		"“需要修改”只提交审查结论，PR 保持开放。受控写操作会先生成计划，再由绑定的 GitLink 身份在本地确认。",
	}, "\n")
}
