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
	title := fmt.Sprintf("%s PR #%d Review", repository, number)
	if view != nil && strings.TrimSpace(view.Title) != "" {
		title += " · " + truncateReviewGatewayText(view.Title, 100)
	}

	elements := []interface{}{}
	if result.Status == "failed" {
		elements = append(elements,
			div("**读取失败**\n"+escapeMD(truncateReviewGatewayText(redactReviewGatewayError(firstNonEmpty(result.Error, "unknown")), 500))),
			note("未执行 GitLink 写入；请检查网络或稍后刷新。"),
		)
		return boundedReviewGatewayCard(baseCard(title, "red", elements))
	}

	if result.Partial || (result.CollectionStatus != "" && result.CollectionStatus != "complete") {
		elements = append(elements, div(fmt.Sprintf(
			"**⚠ 数据不完整**\ncollection_status=%s；partial=%t。当前结果不得覆盖已有完整快照。",
			escapeMD(firstNonEmpty(result.CollectionStatus, "unknown")),
			result.Partial,
		)))
	}

	if view != nil {
		branch := firstNonEmpty(view.BaseBranch, "unknown") + " ← " + firstNonEmpty(view.HeadBranch, "unknown")
		elements = append(elements,
			fields([]fieldValue{
				{Label: "作者", Value: firstNonEmpty(view.Author, "unknown")},
				{Label: "状态", Value: firstNonEmpty(result.GitLinkState, "unknown")},
				{Label: "分支", Value: branch},
				{Label: "Head", Value: shortReviewGatewaySHA(result.HeadSHA)},
			}),
			fields([]fieldValue{
				{Label: "文件", Value: fmt.Sprintf("%d", view.FilesCount)},
				{Label: "变更", Value: fmt.Sprintf("+%d / -%d", view.Additions, view.Deletions)},
				{Label: "提交", Value: fmt.Sprintf("%d", view.CommitsCount)},
				{Label: "Patchset", Value: firstNonEmpty(view.PatchsetID, "unknown")},
			}),
		)
	}

	elements = append(elements, fields([]fieldValue{
		{Label: "Review 阶段", Value: firstNonEmpty(result.ReviewStage, reviewItemValue(item, func(value ReviewCollaborationItem) string { return value.ReviewStage }), "unknown")},
		{Label: "当前决定", Value: firstNonEmpty(result.Decision, reviewItemValue(item, func(value ReviewCollaborationItem) string { return value.Decision }), "unknown")},
		{Label: "Review", Value: fmt.Sprintf("%d", result.ReviewCount)},
		{Label: "未解决线程", Value: fmt.Sprintf("%d", result.OpenThreadCount)},
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
				truncateReviewGatewayText(firstNonEmpty(reviewer.Decision, "unknown"), 32),
			))
		}
		elements = append(elements, div("**Reviewer 摘要**\n"+bulletList(lines, 8)))
	}

	if view != nil {
		elements = append(elements, fields([]fieldValue{
			{Label: "风险", Value: firstNonEmpty(view.RiskLevel, "unknown")},
			{Label: "数据完整性", Value: fmt.Sprintf("%s / partial=%t", firstNonEmpty(result.CollectionStatus, "unknown"), result.Partial)},
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
			elements = append(elements, div("**建议下一步**\n"+escapeMD(truncateReviewGatewayText(view.RecommendedNextStep, 240))))
		}
	}

	if item != nil && !result.PublicRead {
		elements = append(elements, fields([]fieldValue{
			{Label: "协作状态", Value: firstNonEmpty(item.CollaborationStatus, "unassigned")},
			{Label: "负责人", Value: reviewGatewayAssigneeLabel(*item)},
			{Label: "截止时间", Value: firstNonEmpty(item.DueAt, "未设置")},
			{Label: "协作资源", Value: "Base / Doc / Task 按配置同步"},
		}))
	}
	if plan := result.ActionPlan; plan != nil {
		localCommand := fmt.Sprintf("gitlink-cli feishu +review-confirm-local --plan-id %s --state-db .local/review-gateway.db", plan.PlanID)
		actionLabel := reviewActionLabel(plan.Action)
		if plan.Action == reviewActionMerge {
			elements = append(elements, div("**HIGH RISK: this action will merge the PR and change its target branch.**"))
		} else if plan.Action == reviewActionRejectClose {
			elements = append(elements, div("**HIGH RISK: this action will reject and close the PR.**"))
		}
		elements = append(elements,
			fields([]fieldValue{{Label: "Action", Value: actionLabel}, {Label: "ActionPlan", Value: plan.PlanID}, {Label: "Request ID", Value: plan.RequestID},
				{Label: "Feishu actor", Value: reviewGatewayHashIdentifier(plan.ActorID)}, {Label: "GitLink login", Value: plan.GitLinkLogin},
				{Label: "Expected head", Value: shortReviewGatewaySHA(plan.ExpectedHeadSHA)}, {Label: "Expires", Value: plan.ExpiresAt}}),
			div("**Content / Reason**\n"+escapeMD(truncateReviewGatewayText(firstNonEmpty(plan.Content, "No remote body for this lifecycle action"), 1200))),
			note("Local command: "+localCommand),
			reviewCommandActions([][2]string{{"本地执行计划", "本地执行 Review " + plan.PlanID}, {"取消", "取消 Review " + plan.PlanID},
				{"刷新 Context", fmt.Sprintf("刷新 %s PR #%d", plan.Repository, plan.PRNumber)}}),
		)
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
	return "飞书成员 " + reviewGatewayHashIdentifier(item.AssignedTo)
}

func shortReviewGatewaySHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}
	return firstNonEmpty(value, "unknown")
}

func reviewGatewayWriteBoundary(result ReviewGatewayExecutionResult) string {
	if result.WriteResult == nil {
		return "GitLink 写入：0；最终 Review 和合并决定仍由仓库 Owner 完成。"
	}
	switch result.WriteResult.MutationStatus {
	case reviewMutationConfirmed:
		return "GitLink 写入：已确认；请依据回读结果完成审计。"
	case reviewMutationPossible:
		return "GitLink 写入：结果不确定；禁止自动重试，必须人工对账。"
	default:
		return "GitLink 写入：0。"
	}
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
