package feishu

import (
	"fmt"
	"strings"
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
		return "要求修改"
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
		return fmt.Sprintf("PR #%d 已要求修改", number)
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
		return "待决定"
	case "common", "commented":
		return "已提交审查意见"
	case "approved":
		return "已批准"
	case "rejected", "changes_pending", "blocked":
		return "需修改"
	case "none":
		return "暂无"
	default:
		return "待确认"
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
		return "待确认"
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
	default:
		return "待确认"
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

func reviewActionRiskNotice(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case reviewActionMerge:
		return "**高风险操作：最终确认后将合并此 PR，并修改目标分支内容。**"
	case reviewActionRejectClose:
		return "**高风险操作：最终确认后将拒绝并关闭此 PR。**"
	case reviewActionApprove:
		return "最终确认后将向 GitLink 提交“批准”审查结果。"
	case reviewActionReject:
		return "最终确认后将提交“需修改”审查结果，PR 保持开放。"
	case reviewActionCommon:
		return "最终确认后将向 GitLink 提交一条普通审查意见。"
	default:
		return ""
	}
}

func reviewGatewayHelpText() string {
	return strings.Join([]string{
		"GitLink Review 助手命令",
		"",
		"PR 查询",
		"查看 owner/repo PR #123",
		"",
		"协作",
		"领取 owner/repo PR #123",
		"取消领取 owner/repo PR #123",
		"设置 owner/repo PR #123 审查截止 YYYY-MM-DD",
		"",
		"审查",
		"提交审查意见 owner/repo PR #123 <意见>",
		"批准 owner/repo PR #123 <说明>",
		"要求修改 owner/repo PR #123 <原因>",
		"拒绝并关闭 owner/repo PR #123 <原因>",
		"合并 owner/repo PR #123",
		"",
		"“要求修改”只提交审查结论，PR 保持开放。批准、要求修改、拒绝并关闭和合并均只生成操作计划，最终执行需要绑定的 GitLink 身份在本地确认。GitLink 权限由 GitLink 服务端在执行时校验。",
		"Reviewer 修改、行级评论写入和讨论解决暂未支持。",
	}, "\n")
}
