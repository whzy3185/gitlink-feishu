package feishu

import (
	"fmt"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type Card map[string]interface{}

type WebhookPayload struct {
	Timestamp string            `json:"timestamp,omitempty"`
	Sign      string            `json:"sign,omitempty"`
	MsgType   string            `json:"msg_type"`
	Card      Card              `json:"card,omitempty"`
	Content   map[string]string `json:"content,omitempty"`
}

func NewInteractivePayload(card Card) WebhookPayload {
	return WebhookPayload{
		MsgType: "interactive",
		Card:    card,
	}
}

func BuildBotTestCard(title, message, lang string) Card {
	title = firstNonEmpty(title, feishuLabel(lang, "bot_title"))
	message = firstNonEmpty(message, feishuLabel(lang, "bot_message"))
	return baseCard(title, "blue", []interface{}{
		div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "bot_status"), feishuLabel(lang, "ready"))),
		div(message),
		note(feishuLabel(lang, "bot_generated")),
	})
}

func BuildWorkflowCard(report workflow.RepoReportResult, include []string, title string, lang string, docURL string) Card {
	title = firstNonEmpty(title, reportTitle(report, lang))
	elements := []interface{}{
		div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "repository"), escapeMD(report.Repository))),
		fields([]fieldValue{
			{Label: feishuLabel(lang, "report_score"), Value: fmt.Sprintf("%d", report.ReportScore)},
			{Label: feishuLabel(lang, "risk_level"), Value: report.RiskLevel},
			{Label: feishuLabel(lang, "source"), Value: report.Source},
		}),
	}
	if hasItem(include, "health") {
		healthScore := "N/A"
		healthRisk := "N/A"
		if report.Health != nil {
			healthScore = fmt.Sprintf("%d", report.Health.HealthScore)
			healthRisk = report.Health.RiskLevel
		}
		elements = append(elements, fields([]fieldValue{
			{Label: feishuLabel(lang, "health_score"), Value: healthScore},
			{Label: feishuLabel(lang, "health_risk"), Value: healthRisk},
		}))
	}
	if hasItem(include, "issues") {
		elements = append(elements, fields([]fieldValue{
			{Label: feishuLabel(lang, "issues"), Value: fmt.Sprintf("%d", report.IssueSummary.Total)},
			{Label: feishuLabel(lang, "high_risk_issues"), Value: fmt.Sprintf("%d", report.IssueSummary.HighRisk)},
			{Label: feishuLabel(lang, "missing_info"), Value: fmt.Sprintf("%d", report.IssueSummary.MissingInfo)},
		}))
	}
	if hasItem(include, "prs") {
		elements = append(elements, fields([]fieldValue{
			{Label: feishuLabel(lang, "pull_requests"), Value: fmt.Sprintf("%d", report.PRSummary.Total)},
			{Label: feishuLabel(lang, "high_risk_prs"), Value: fmt.Sprintf("%d", report.PRSummary.HighRisk)},
		}))
		if len(report.PRSummary.ReviewFocus) > 0 {
			elements = append(elements, div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "review_focus"), bulletList(localizeFeishuLines(report.PRSummary.ReviewFocus, lang), 4))))
		}
	}
	if len(report.Recommendations) > 0 {
		elements = append(elements, div(fmt.Sprintf("**%s**\n%s", feishuLabel(lang, "recommendations"), bulletList(localizeFeishuLines(report.Recommendations, lang), 5))))
	}
	if strings.TrimSpace(docURL) != "" {
		elements = append(elements, actionButton(feishuLabel(lang, "open_feishu_report"), docURL))
	}
	elements = append(elements, note(feishuLabel(lang, "preview_note")))
	return baseCard(title, templateForRisk(report.RiskLevel), elements)
}

func reportTitle(report workflow.RepoReportResult, lang string) string {
	return fmt.Sprintf(feishuLabel(lang, "workflow_report_title"), report.Repository)
}

func baseCard(title string, template string, elements []interface{}) Card {
	return Card{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": template,
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		},
		"elements": elements,
	}
}

func div(content string) map[string]interface{} {
	return map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": content,
		},
	}
}

type fieldValue struct {
	Label string
	Value string
}

func fields(values []fieldValue) map[string]interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, map[string]interface{}{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**%s**\n%s", value.Label, escapeMD(value.Value)),
			},
		})
	}
	return map[string]interface{}{
		"tag":    "div",
		"fields": result,
	}
}

func note(content string) map[string]interface{} {
	return map[string]interface{}{
		"tag": "note",
		"elements": []interface{}{
			map[string]interface{}{
				"tag":     "plain_text",
				"content": content,
			},
		},
	}
}

func actionButton(text string, url string) map[string]interface{} {
	return map[string]interface{}{
		"tag": "action",
		"actions": []interface{}{
			map[string]interface{}{
				"tag": "button",
				"text": map[string]interface{}{
					"tag":     "plain_text",
					"content": text,
				},
				"type": "primary",
				"url":  strings.TrimSpace(url),
			},
		},
	}
}

func bulletList(values []string, limit int) string {
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	lines := make([]string, 0, limit)
	for _, value := range values[:limit] {
		lines = append(lines, "- "+escapeMD(value))
	}
	return strings.Join(lines, "\n")
}

func templateForRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "critical":
		return "red"
	case "high":
		return "orange"
	case "medium":
		return "yellow"
	default:
		return "green"
	}
}

func escapeMD(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "N/A"
	}
	return strings.ReplaceAll(value, "\n", " ")
}
