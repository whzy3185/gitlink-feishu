package feishu

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const reviewGatewayCardJSONLimit = 28000

// buildReviewGatewayResultCard creates one bounded card schema for complete,
// partial, and failed PR results. Public reads intentionally receive only the
// GitLink navigation button; collaboration controls and resource links belong
// to bound installations and later milestones.
func buildReviewGatewayResultCard(
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	item *ReviewCollaborationItem,
) Card {
	repository := firstNonEmpty(result.Repository, job.Repository)
	number := result.PRNumber
	if number <= 0 {
		number = job.PRNumber
	}
	view := result.PullRequest
	title := fmt.Sprintf("%s · PR #%d", repository, number)
	if plan := result.ActionPlan; plan != nil {
		title = reviewActionPlanCardTitle(*plan, result.WriteResult)
	}

	elements := []interface{}{}
	if result.Status == "failed" {
		elements = append(elements,
			div("**读取失败**\n"+escapeMD(truncateReviewGatewayText(redactReviewGatewayError(firstNonEmpty(result.Error, "unknown")), 500))),
			note("未执行 GitLink 写入；请检查网络或稍后刷新。"),
		)
		return boundedReviewGatewayCard(baseCard(title, "red", elements))
	}
	if plan := result.ActionPlan; plan != nil {
		elements = append(elements, reviewActionPlanCardElements(*plan, result.WriteResult, result.Action, result.ConfirmationStateDB)...)
		gitLinkURL := reviewGatewayGitLinkURL(repository, number)
		if view != nil && strings.HasPrefix(strings.TrimSpace(view.GitLinkURL), "https://www.gitlink.org.cn/") {
			gitLinkURL = strings.TrimSpace(view.GitLinkURL)
		}
		if gitLinkURL != "" {
			elements = append(elements, actionButton("打开 GitLink PR", gitLinkURL))
		}
		elements = append(elements, note(reviewGatewayWriteBoundary(result)))
		return boundedReviewGatewayCard(baseCard(title, reviewActionPlanCardTemplate(*plan, result.WriteResult), elements))
	}

	if result.Partial || (result.CollectionStatus != "" && result.CollectionStatus != "complete") {
		elements = append(elements, div("**⚠ 数据不完整**\n当前结果不会覆盖已有完整快照，请稍后刷新。"))
	}

	if view != nil {
		if strings.TrimSpace(view.Title) != "" {
			elements = append(elements, div("**"+escapeMD(truncateReviewGatewayText(view.Title, 120))+"**"))
		}
		elements = append(elements,
			fields([]fieldValue{
				{Label: "作者", Value: firstNonEmpty(view.Author, "待确认")},
				{Label: "PR 状态", Value: reviewPRStateDisplayName(result.GitLinkState)},
				{Label: "目标分支", Value: firstNonEmpty(view.BaseBranch, "待确认")},
				{Label: "来源分支", Value: firstNonEmpty(view.HeadBranch, "待确认")},
			}),
			fields([]fieldValue{
				{Label: "当前版本", Value: shortReviewGatewaySHA(result.HeadSHA)},
				{Label: "文件变化", Value: fmt.Sprintf("%d 个文件 · +%d / -%d", view.FilesCount, view.Additions, view.Deletions)},
				{Label: "提交数量", Value: fmt.Sprintf("%d 次提交", view.CommitsCount)},
				{Label: "版本批次", Value: firstNonEmpty(view.PatchsetID, "待确认")},
			}),
		)
	}

	elements = append(elements, fields([]fieldValue{
		{Label: "审查阶段", Value: reviewStageDisplayName(firstNonEmpty(result.ReviewStage, reviewItemValue(item, func(value ReviewCollaborationItem) string { return value.ReviewStage })))},
		{Label: "当前结论", Value: reviewDecisionDisplayName(firstNonEmpty(result.Decision, reviewItemValue(item, func(value ReviewCollaborationItem) string { return value.Decision })))},
		{Label: "审查记录", Value: fmt.Sprintf("%d", result.ReviewCount)},
		{Label: "未解决讨论", Value: fmt.Sprintf("%d", result.OpenThreadCount)},
	}))

	if view != nil && len(view.Reviewers) > 0 {
		lines := make([]string, 0, 8)
		for _, reviewer := range view.Reviewers {
			if len(lines) >= 8 {
				break
			}
			lines = append(lines, fmt.Sprintf(
				"%s：%s",
				truncateReviewGatewayText(reviewer.Reviewer, 64),
				reviewReviewerDecisionDisplayName(reviewer.Decision),
			))
		}
		elements = append(elements, div("**Reviewer 摘要**\n"+bulletList(lines, 8)))
	}

	if view != nil {
		elements = append(elements, fields([]fieldValue{
			{Label: "数据状态", Value: reviewCollectionStatusDisplayName(result.CollectionStatus, result.Partial)},
			{Label: "风险", Value: reviewRiskDisplayName(view.RiskLevel)},
		}))
		if len(view.Unknowns) > 0 {
			unknowns := make([]string, 0, 5)
			for _, unknown := range view.Unknowns {
				if len(unknowns) >= 5 {
					break
				}
				unknowns = append(unknowns, truncateReviewGatewayText(redactReviewGatewayError(unknown), 180))
			}
			elements = append(elements, div("**待确认信息**\n"+bulletList(unknowns, 5)))
		}
		if strings.TrimSpace(view.RecommendedNextStep) != "" {
			elements = append(elements, div("**建议下一步**\n"+reviewNextStepDisplayName(view.RecommendedNextStep)))
		}
	}

	if item != nil && !result.PublicRead {
		elements = append(elements, fields([]fieldValue{
			{Label: "协作状态", Value: reviewCollaborationStatusDisplayName(item.CollaborationStatus)},
			{Label: "负责人", Value: reviewGatewayAssigneeLabel(*item)},
			{Label: "审查截止", Value: firstNonEmpty(item.DueAt, "未设置")},
		}))
	}
	gitLinkURL := reviewGatewayGitLinkURL(repository, number)
	if view != nil && strings.HasPrefix(strings.TrimSpace(view.GitLinkURL), "https://www.gitlink.org.cn/") {
		gitLinkURL = strings.TrimSpace(view.GitLinkURL)
	}
	if gitLinkURL != "" {
		elements = append(elements, actionButton("打开 GitLink PR", gitLinkURL))
	}
	elements = append(elements, note(reviewGatewayWriteBoundary(result)))

	template := templateForRisk("")
	if view != nil {
		template = templateForRisk(view.RiskLevel)
	}
	if result.Partial || result.CollectionStatus == "partial" {
		template = "yellow"
	}
	if result.ActionPlan != nil && (result.ActionPlan.Action == reviewActionMerge || result.ActionPlan.Action == reviewActionRejectClose) {
		template = "red"
	}
	switch strings.ToLower(strings.TrimSpace(result.GitLinkState)) {
	case "merged":
		template = "green"
	case "closed":
		template = "grey"
	}
	return boundedReviewGatewayCard(baseCard(title, template, elements))
}

func quoteReviewGatewayCLIArgument(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\r\n\"") {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func reviewCommandActions(actions [][2]string) map[string]interface{} {
	buttons := make([]interface{}, 0, len(actions))
	for _, action := range actions {
		buttons = append(buttons, map[string]interface{}{"tag": "button", "text": map[string]interface{}{"tag": "plain_text", "content": action[0]}, "value": map[string]interface{}{"command": action[1]}})
	}
	return map[string]interface{}{"tag": "action", "actions": buttons}
}

func reviewItemValue(item *ReviewCollaborationItem, selector func(ReviewCollaborationItem) string) string {
	if item == nil {
		return ""
	}
	return selector(*item)
}

var reviewGatewayFeishuOpenIDPattern = regexp.MustCompile(`^ou_[A-Za-z0-9_-]+$`)

func reviewGatewayAssigneeLabel(item ReviewCollaborationItem) string {
	if strings.TrimSpace(item.AssignedTo) == "" {
		return "未认领"
	}
	if strings.TrimSpace(item.AssignedDisplayName) != "" {
		return truncateReviewGatewayText(item.AssignedDisplayName, 80)
	}
	userID := strings.TrimSpace(item.AssignedTo)
	if reviewGatewayFeishuOpenIDPattern.MatchString(userID) {
		// Lark markdown resolves this opaque open_id to the tenant-visible member
		// name. The card no longer invents a generic assignee label.
		return fmt.Sprintf("<at id=%s></at>", userID)
	}
	return "负责人身份待解析"
}

func reviewGatewayAssigneePlainLabel(item ReviewCollaborationItem) string {
	if strings.TrimSpace(item.AssignedTo) == "" {
		return "未认领"
	}
	if strings.TrimSpace(item.AssignedDisplayName) != "" {
		return truncateReviewGatewayText(item.AssignedDisplayName, 80)
	}
	return "飞书成员"
}

func shortReviewGatewaySHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}
	return firstNonEmpty(value, "待确认")
}

func reviewGatewayWriteBoundary(result ReviewGatewayExecutionResult) string {
	if result.WriteResult == nil {
		if plan := result.ActionPlan; plan != nil {
			switch reviewActionPlanPresentationState(*plan, nil) {
			case "completed":
				return "GitLink 操作已完成并通过回读验证。"
			case "stale":
				return "PR 状态已经变化，原操作计划已失效；本次未修改 GitLink。"
			case "cancelled":
				return "操作已取消；本次未修改 GitLink。"
			case "unknown":
				return "远程结果暂无法确认，系统已停止自动重试。"
			default:
				return "操作计划已生成，尚未修改 GitLink。最终执行需要使用绑定的 GitLink 身份完成本地确认。"
			}
		}
		return "本次操作未修改 GitLink。"
	}
	switch result.WriteResult.MutationStatus {
	case reviewMutationConfirmed:
		return "GitLink 操作已完成并通过回读验证。"
	case reviewMutationPossible:
		return "远程结果暂无法确认，系统已停止自动重试。"
	default:
		if result.WriteResult.Status == "stale" {
			return "PR 状态已经变化，原操作计划已失效；本次未修改 GitLink。"
		}
		return "本次操作未修改 GitLink。"
	}
}

func reviewActionPlanCardTitle(plan ReviewActionPlan, write *ReviewWriteResult) string {
	state := reviewActionPlanPresentationState(plan, write)
	switch state {
	case "completed":
		return reviewActionCompletedTitle(plan.Action, plan.PRNumber)
	case "stale":
		return "操作计划已失效"
	case "cancelled":
		return "操作已取消"
	case "unknown":
		return "操作结果暂无法确认"
	default:
		return fmt.Sprintf("%s PR #%d", reviewActionDisplayName(plan.Action), plan.PRNumber)
	}
}

func reviewActionPlanCardTemplate(plan ReviewActionPlan, write *ReviewWriteResult) string {
	switch reviewActionPlanPresentationState(plan, write) {
	case "completed":
		return "green"
	case "stale", "cancelled":
		return "grey"
	case "unknown":
		return "yellow"
	default:
		if plan.Action == reviewActionMerge || plan.Action == reviewActionRejectClose {
			return "red"
		}
		return "blue"
	}
}

func reviewActionPlanPresentationState(plan ReviewActionPlan, write *ReviewWriteResult) string {
	if write != nil {
		switch strings.ToLower(strings.TrimSpace(write.Status)) {
		case "completed", "verified":
			return "completed"
		case "stale":
			return "stale"
		case "cancelled":
			return "cancelled"
		case "unknown", "unknown_needs_reconciliation":
			return "unknown"
		}
		if write.MutationStatus == reviewMutationPossible {
			return "unknown"
		}
	}
	switch strings.ToLower(strings.TrimSpace(plan.Status)) {
	case "completed":
		return "completed"
	case "stale":
		return "stale"
	case "cancelled":
		return "cancelled"
	case "unknown", "unknown_needs_reconciliation":
		return "unknown"
	default:
		return "pending"
	}
}

func reviewActionPlanCardElements(plan ReviewActionPlan, write *ReviewWriteResult, resultAction, stateDB string) []interface{} {
	state := reviewActionPlanPresentationState(plan, write)
	common := fields([]fieldValue{
		{Label: "仓库", Value: plan.Repository},
		{Label: "操作", Value: reviewActionDisplayName(plan.Action)},
		{Label: "操作人", Value: reviewGatewayActorLabel(plan.ActorID)},
		{Label: "GitLink 身份", Value: firstNonEmpty(plan.GitLinkLogin, "待确认")},
		{Label: "基于版本", Value: shortReviewGatewaySHA(plan.ExpectedHeadSHA)},
		{Label: "操作编号", Value: firstNonEmpty(plan.RequestID, "待确认")},
	})
	content := reviewPresentationPlanContent(plan.Content)
	switch state {
	case "completed":
		elements := []interface{}{common, fields([]fieldValue{{Label: "结果", Value: "操作已完成并通过 GitLink 回读验证"}})}
		if write != nil && strings.TrimSpace(write.ReviewID) != "" {
			elements = append(elements, fields([]fieldValue{{Label: "审查记录", Value: "#" + write.ReviewID}}))
		}
		return elements
	case "stale":
		return []interface{}{common, div("PR 的代码版本已经发生变化。\n本次未修改 GitLink。\n请刷新 PR 后重新发起操作。")}
	case "cancelled":
		return []interface{}{common, div("本次未修改 GitLink。")}
	case "unknown":
		return []interface{}{common, div("系统已停止自动重试，以避免重复写入。\n请先核对 GitLink 当前状态。")}
	default:
		elements := []interface{}{common}
		if content != "" {
			elements = append(elements, div("**审查意见 / 操作原因**\n"+escapeMD(truncateReviewGatewayText(content, 1200))))
		}
		if notice := reviewActionRiskNotice(plan.Action); notice != "" {
			elements = append(elements, div(notice))
		}
		elements = append(elements,
			fields([]fieldValue{{Label: "状态", Value: "待本地确认"}}),
			div("操作计划已经生成，当前尚未修改 GitLink。\n请在已登录对应 GitLink 身份的本地 CLI 中完成最终确认。\nGitLink 权限将在执行时由 GitLink 服务端校验。"),
			reviewCommandActions([][2]string{
				{"查看本地确认方式", "本地执行 Review " + plan.PlanID},
				{"取消操作", "取消 Review " + plan.PlanID},
				{"刷新 PR 状态", fmt.Sprintf("刷新 %s PR #%d", plan.Repository, plan.PRNumber)},
			}),
		)
		if resultAction == "show_local_review_plan" {
			command := fmt.Sprintf(
				"gitlink-cli feishu +review-confirm-local --plan-id %s --state-db %s",
				plan.PlanID,
				quoteReviewGatewayCLIArgument(firstNonEmpty(stateDB, ".local/review-gateway.db")),
			)
			elements = append(elements, div("**本地确认命令（高级信息）**\n`"+escapeMD(command)+"`"))
		}
		return elements
	}
}

func reviewGatewayActorLabel(actorID string) string {
	actorID = strings.TrimSpace(actorID)
	if reviewGatewayFeishuOpenIDPattern.MatchString(actorID) {
		return fmt.Sprintf("<at id=%s></at>", actorID)
	}
	return "当前飞书用户"
}

func reviewPresentationPlanContent(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Ref: RW-") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func boundedReviewGatewayCard(card Card) Card {
	if _, ok := safeReviewGatewayCardJSON(card); !ok {
		return nil
	}
	return card
}

func safeReviewGatewayCardJSON(card Card) (string, bool) {
	if len(card) == 0 {
		return "", false
	}
	encoded, err := json.Marshal(card)
	if err != nil || len(encoded) > reviewGatewayCardJSONLimit {
		return "", false
	}
	return string(encoded), true
}
