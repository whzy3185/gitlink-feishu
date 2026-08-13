package feishu

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
)

func reviewGatewayResultSendInput(job ReviewGatewayJob, result ReviewGatewayExecutionResult) *larktypes.SendInput {
	input := &larktypes.SendInput{
		ChatID:         job.ChatID,
		ReplyMessageID: job.SourceMessageID,
		MsgType:        "text",
		Text:           formatReviewGatewayResultReply(job, result),
	}
	if result.Status == "failed" || result.ReviewData == nil {
		if result.ActionPlan == nil {
			return input
		}
	}
	if result.ActionPlan != nil {
		card, err := buildReviewActionPlanCard(job, *result.ActionPlan, result.ExecutionPath, result.FallbackReason, result.Error)
		if err == nil {
			input.MsgType, input.Card, input.Text = "interactive", card, ""
		}
		return input
	}
	card, err := buildReviewGatewayPRCard(job, result)
	if err != nil {
		return input
	}
	input.MsgType = "interactive"
	input.Card = card
	input.Text = ""
	return input
}

func buildReviewActionPlanCard(job ReviewGatewayJob, plan ReviewActionPlan, executionPath, fallbackReason, errorSummary string) (string, error) {
	status := reviewActionPlanStatusDisplay(plan.Status)
	title := fmt.Sprintf("%s PR #%d", reviewActionDisplayName(plan.Action), plan.PRNumber)
	rows := []string{
		fmt.Sprintf("**仓库**：%s", plan.Repository),
		fmt.Sprintf("**操作**：%s", reviewActionDisplayName(plan.Action)),
		fmt.Sprintf("**GitLink 身份**：%s", plan.GitLinkLogin),
		fmt.Sprintf("**基于版本**：%s", truncateReviewGatewayText(plan.ExpectedHeadSHA, 12)),
		fmt.Sprintf("**操作编号**：%s", plan.RequestID),
		fmt.Sprintf("**状态**：%s", status),
	}
	switch plan.Action {
	case reviewActionComment:
		rows = append(rows, "最终确认后将在 GitLink PR 会话区发布一条评论。")
	case reviewActionRejectClose:
		rows = append(rows, "**高风险操作**：最终确认后将拒绝并关闭此 PR。")
	case reviewActionMerge:
		rows = append(rows, "**高风险操作**：最终确认后将合并此 PR，并修改目标分支内容。")
	case reviewActionReject:
		rows = append(rows, "最终确认后将提交“需要修改”审查结果，PR 保持开放。")
	case reviewActionApprove:
		rows = append(rows, "最终确认后将向 GitLink 提交“批准”审查结果。")
	case reviewActionCommon:
		rows = append(rows, "最终确认后将向 GitLink 提交一条普通审查意见。")
	}
	switch plan.Status {
	case "pending_confirmation":
		if executionPath == reviewExecutionPathLocalConfirmation {
			rows = append(rows, reviewLocalConfirmationReason(fallbackReason))
		}
		rows = append(rows,
			"操作计划已经生成，当前尚未修改 GitLink。",
			"请在已登录对应 GitLink 身份的本地 CLI 中完成最终确认。",
			fmt.Sprintf("本地确认标识：`%s`", plan.PlanID),
			"GitLink 权限将在执行时由 GitLink 服务端校验。",
		)
	case "completed":
		rows = append(rows, "**结果**：操作已完成并通过 GitLink 回读验证")
		if plan.ReviewID != "" {
			label := "审查记录"
			if plan.Action == reviewActionComment {
				label = "评论记录"
			}
			rows = append(rows, "**"+label+"**：#"+plan.ReviewID)
		}
	case "unknown_needs_reconciliation":
		rows = append(rows, "操作结果暂无法确认。系统已停止自动重试，以避免重复写入。")
	case "cancelled":
		rows = append(rows, "操作已取消，本次未修改 GitLink。")
	case "failed":
		rows = append(rows, "操作未完成，请核对错误信息后重新生成计划。")
		if strings.TrimSpace(errorSummary) != "" {
			rows = append(rows, "**原因**："+truncateReviewGatewayText(errorSummary, 500))
		}
	}
	payload := map[string]interface{}{
		"schema": "2.0",
		"config": map[string]interface{}{"update_multi": true},
		"header": map[string]interface{}{"title": map[string]interface{}{"tag": "plain_text", "content": title}},
		"body":   map[string]interface{}{"elements": []map[string]interface{}{{"tag": "markdown", "content": strings.Join(rows, "\n")}}},
	}
	encoded, err := json.Marshal(payload)
	return string(encoded), err
}

func reviewLocalConfirmationReason(reason string) string {
	switch reason {
	case reviewFallbackIdentityMismatch:
		return "当前 Gateway Credential 与绑定的 GitLink 身份不一致，已安全回退到本地确认。"
	case reviewFallbackCredentialUnavailable:
		return "当前 Gateway 没有可验证的 GitLink Credential，已安全回退到本地确认。"
	case reviewFallbackIdentityUnverified:
		return "当前 Gateway 无法验证 GitLink 身份，已安全回退到本地确认。"
	default:
		return "当前部署使用本地确认模式。"
	}
}

func reviewActionPlanStatusDisplay(status string) string {
	switch status {
	case "pending_confirmation":
		return "待本地确认"
	case "completed":
		return "已完成"
	case "cancelled":
		return "已取消"
	case "unknown_needs_reconciliation":
		return "结果待核对"
	case "failed":
		return "执行失败"
	default:
		return "待确认"
	}
}

func buildReviewGatewayPRCard(job ReviewGatewayJob, result ReviewGatewayExecutionResult) (string, error) {
	data := *result.ReviewData
	rows := []string{
		fmt.Sprintf("**作者**：%s", reviewGatewayCardValue(data.Author)),
		fmt.Sprintf("**状态**：%s", reviewGatewayPRStateDisplay(data.State)),
		fmt.Sprintf("**目标分支**：%s", reviewGatewayCardValue(data.BaseBranch)),
		fmt.Sprintf("**来源分支**：%s", reviewGatewayCardValue(data.HeadBranch)),
		fmt.Sprintf("**当前版本**：%s", reviewGatewayCardValue(truncateReviewGatewayText(data.HeadSHA, 12))),
		fmt.Sprintf("**变更**：%d 个文件 · +%d / -%d · %d 次提交", data.Patchset.FilesCount, data.Patchset.Additions, data.Patchset.Deletions, data.Patchset.CommitsCount),
		fmt.Sprintf("**Review**：%d 条", data.Summary.TotalReviews),
		fmt.Sprintf("**当前结论**：%s", reviewGatewayDecisionDisplay(data.Summary.ReviewStatus)),
	}
	if len(data.ReviewerSummaries) > 0 {
		rows = append(rows, "**审查者**：")
		for _, reviewer := range data.ReviewerSummaries {
			rows = append(rows, "- "+reviewGatewayReviewerDisplay(reviewer))
		}
	}
	if item := result.Collaboration; item != nil {
		rows = append(rows,
			fmt.Sprintf("**负责人**：%s", firstNonEmpty(item.AssignedDisplayName, "无")),
		)
		if item.DueAt != "" {
			rows = append(rows, fmt.Sprintf("**审查截止**：%s", item.DueAt))
		}
	}
	elements := []map[string]interface{}{{
		"tag":     "markdown",
		"content": strings.Join(rows, "\n"),
	}}
	if strings.TrimSpace(data.GitLinkURL) != "" {
		elements = append(elements, map[string]interface{}{
			"tag":  "button",
			"text": map[string]interface{}{"tag": "plain_text", "content": "打开 GitLink PR"},
			"type": "primary",
			"url":  data.GitLinkURL,
		})
	}
	payload := map[string]interface{}{
		"schema": "2.0",
		"config": map[string]interface{}{"update_multi": true},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": fmt.Sprintf("%s PR #%d · %s", job.Repository, job.PRNumber, reviewGatewayCardValue(data.Title)),
			},
		},
		"body": map[string]interface{}{"elements": elements},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal review gateway card: %w", err)
	}
	return string(encoded), nil
}

func reviewGatewayPRStateDisplay(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return "开放中"
	case "closed":
		return "已关闭"
	case "merged":
		return "已合并"
	default:
		return "待确认"
	}
}

func reviewGatewayDecisionDisplay(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "approved":
		return "已批准"
	case "rejected", "blocked":
		return "需要修改"
	case "common", "commented":
		return "已提交审查意见"
	case "none", "pending":
		return "待审查"
	default:
		return "待确认"
	}
}

func reviewGatewayReviewerDisplay(reviewer ReviewDataReviewerSummary) string {
	name := firstNonEmpty(reviewer.Actor, reviewer.ActorID, "GitLink 用户")
	line := fmt.Sprintf("%s（%s", name, reviewGatewayDecisionDisplay(reviewer.CurrentDecision))
	if reviewer.LatestEffectiveReview != nil {
		value := firstNonEmpty(reviewer.LatestEffectiveReview.UpdatedAt, reviewer.LatestEffectiveReview.CreatedAt)
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			china := time.FixedZone("Asia/Shanghai", 8*60*60)
			line += " · " + parsed.In(china).Format("2006-01-02 15:04")
		}
	}
	return line + "）"
}

func reviewGatewayCardValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "unknown" {
		return "待确认"
	}
	return value
}
