package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/collab"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const wecomWebhookBase = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"

type ReviewPayload struct {
	MsgType      string       `json:"msgtype"`
	TemplateCard TemplateCard `json:"template_card"`
}

type TemplateCard struct {
	CardType              string              `json:"card_type"`
	Source                CardSource          `json:"source"`
	MainTitle             CardTitle           `json:"main_title"`
	EmphasisContent       CardTitle           `json:"emphasis_content"`
	SubTitleText          string              `json:"sub_title_text"`
	HorizontalContentList []HorizontalContent `json:"horizontal_content_list"`
	JumpList              []Jump              `json:"jump_list,omitempty"`
	CardAction            *CardAction         `json:"card_action,omitempty"`
}

type CardSource struct {
	Desc      string `json:"desc"`
	DescColor int    `json:"desc_color"`
}

type CardTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

type HorizontalContent struct {
	KeyName string `json:"keyname"`
	Value   string `json:"value"`
}

type Jump struct {
	Type  int    `json:"type"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

type CardAction struct {
	Type int    `json:"type"`
	URL  string `json:"url"`
}

type DeliveryResult struct {
	Mode          string        `json:"mode"`
	Sent          bool          `json:"sent"`
	WorkItemKey   string        `json:"work_item_key"`
	WebhookTarget string        `json:"webhook_target,omitempty"`
	Payload       ReviewPayload `json:"payload"`
	ErrCode       int           `json:"errcode,omitempty"`
	ErrMsg        string        `json:"errmsg,omitempty"`
	GitLinkWrites int           `json:"gitlink_writes"`
}

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "check",
			Description: "Validate Enterprise WeChat robot configuration without sending a message",
			Flags: []common.Flag{
				{Name: "webhook-url", Usage: "WeCom group robot webhook. Defaults to WECOM_WEBHOOK_URL"},
			},
			Run: runCheck,
		},
		{
			Name:        "review-notify",
			Description: "Render or explicitly send one canonical GitLink Review WorkItem to Enterprise WeChat",
			Flags: []common.Flag{
				{Name: "from", Usage: "Canonical WorkItem or Feishu collaboration bundle JSON", Required: true},
				{Name: "webhook-url", Usage: "WeCom group robot webhook. Defaults to WECOM_WEBHOOK_URL"},
				{Name: "send", Usage: "Send to Enterprise WeChat instead of previewing", Bool: true, Default: "false"},
			},
			Run: runReviewNotify,
		},
		newReviewCoreShortcut(),
	}
}

func runCheck(runtime *common.RuntimeContext) error {
	webhookURL := first(runtime.Arg("webhook-url"), os.Getenv("WECOM_WEBHOOK_URL"))
	result := map[string]interface{}{
		"schema_version": "wecom.check/v1",
		"configured":     webhookURL != "",
		"target":         redactWebhook(webhookURL),
		"remote":         false,
		"gitlink_writes": 0,
	}
	if webhookURL != "" {
		if err := validateWebhookURL(webhookURL); err != nil {
			result["valid"] = false
			result["error"] = err.Error()
			_ = runtime.OutputData(result)
			return err
		}
		result["valid"] = true
	}
	return runtime.OutputData(result)
}

func runReviewNotify(runtime *common.RuntimeContext) error {
	item, err := readWorkItem(runtime.Arg("from"))
	if err != nil {
		return err
	}
	payload := BuildReviewPayload(item)
	webhookURL := first(runtime.Arg("webhook-url"), os.Getenv("WECOM_WEBHOOK_URL"))
	send := strings.EqualFold(runtime.Arg("send"), "true")
	result := DeliveryResult{
		Mode:          "preview",
		WorkItemKey:   item.PRKey,
		WebhookTarget: redactWebhook(webhookURL),
		Payload:       payload,
		GitLinkWrites: 0,
	}
	if !send {
		return runtime.OutputData(result)
	}
	if err := validateWebhookURL(webhookURL); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	errCode, errMsg, err := sendWebhook(ctx, http.DefaultClient, webhookURL, payload)
	if err != nil {
		return err
	}
	result.Mode = "sent"
	result.Sent = errCode == 0
	result.ErrCode = errCode
	result.ErrMsg = errMsg
	if errCode != 0 {
		_ = runtime.OutputData(result)
		return fmt.Errorf("WeCom webhook returned errcode %d: %s", errCode, errMsg)
	}
	return runtime.OutputData(result)
}

func BuildReviewPayload(item collab.WorkItem) ReviewPayload {
	status := fmt.Sprintf("%s / %s", item.ReviewStage, item.Decision)
	contents := []HorizontalContent{
		{KeyName: "协作状态", Value: item.CollaborationStatus},
		{KeyName: "负责人", Value: first(item.AssignedTo, "未认领")},
		{KeyName: "截止时间", Value: first(item.DueAt, "未设置")},
		{KeyName: "下一步", Value: item.NextStep},
		{KeyName: "边界", Value: "GitLink 写入 0"},
	}
	card := TemplateCard{
		CardType: "text_notice",
		Source: CardSource{
			Desc:      "GitLink Review 协作",
			DescColor: 1,
		},
		MainTitle: CardTitle{
			Title: fmt.Sprintf("%s PR #%d", item.Repository, item.PRNumber),
			Desc:  status,
		},
		EmphasisContent: CardTitle{
			Title: first(item.CollectionStatus, "unknown"),
			Desc:  "数据完整性",
		},
		SubTitleText:          fmt.Sprintf("同一 WorkItem：%s", item.PRKey),
		HorizontalContentList: contents,
	}
	if item.GitLinkURL != "" {
		card.JumpList = []Jump{{Type: 1, URL: item.GitLinkURL, Title: "打开 GitLink PR"}}
		card.CardAction = &CardAction{Type: 1, URL: item.GitLinkURL}
	}
	return ReviewPayload{MsgType: "template_card", TemplateCard: card}
}

func readWorkItem(path string) (collab.WorkItem, error) {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return collab.WorkItem{}, err
	}
	var item collab.WorkItem
	if err := json.Unmarshal(payload, &item); err == nil && item.SchemaVersion == collab.WorkItemSchema {
		return item, item.Validate()
	}
	var wrapper struct {
		Canonical collab.WorkItem `json:"canonical"`
	}
	if err := json.Unmarshal(payload, &wrapper); err != nil {
		return collab.WorkItem{}, fmt.Errorf("parse WorkItem: %w", err)
	}
	if err := wrapper.Canonical.Validate(); err != nil {
		return collab.WorkItem{}, err
	}
	return wrapper.Canonical, nil
}

func sendWebhook(
	ctx context.Context,
	client *http.Client,
	webhookURL string,
	payload ReviewPayload,
) (int, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := client.Do(request)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, "", fmt.Errorf("WeCom webhook HTTP %d", response.StatusCode)
	}
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return 0, "", fmt.Errorf("parse WeCom webhook response: %w", err)
	}
	return result.ErrCode, result.ErrMsg, nil
}

func validateWebhookURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("WeCom webhook must be a valid HTTPS URL")
	}
	if !strings.EqualFold(parsed.Host, "qyapi.weixin.qq.com") &&
		!strings.EqualFold(parsed.Host, "localhost") &&
		parsed.Hostname() != "127.0.0.1" {
		return fmt.Errorf("WeCom webhook host must be qyapi.weixin.qq.com")
	}
	if strings.TrimSpace(parsed.Query().Get("key")) == "" && strings.EqualFold(parsed.Host, "qyapi.weixin.qq.com") {
		return fmt.Errorf("WeCom webhook is missing key")
	}
	return nil
}

func redactWebhook(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "***"
	}
	query := parsed.Query()
	if query.Has("key") {
		query.Set("key", "***")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
