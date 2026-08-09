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
