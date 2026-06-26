package feishu

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type DeliveryOptions struct {
	WebhookURL string `json:"-"`
	Secret     string `json:"-"`
	Send       bool   `json:"send"`
	DryRun     bool   `json:"dry_run"`
}

func deliveryOptionsFromContext(ctx *common.RuntimeContext) (DeliveryOptions, error) {
	opts := DeliveryOptions{
		WebhookURL: firstNonEmpty(ctx.Arg("webhook-url"), os.Getenv("FEISHU_WEBHOOK_URL")),
		Secret:     firstNonEmpty(ctx.Arg("secret"), os.Getenv("FEISHU_WEBHOOK_SECRET")),
		Send:       parseBool(ctx.Arg("send")),
		DryRun:     parseBool(ctx.Arg("dry-run")),
	}
	if opts.Send && opts.DryRun {
		return DeliveryOptions{}, fmt.Errorf("--send and --dry-run cannot be used together")
	}
	if opts.Send && strings.TrimSpace(opts.WebhookURL) == "" {
		return DeliveryOptions{}, fmt.Errorf("--send requires --webhook-url or FEISHU_WEBHOOK_URL")
	}
	if opts.WebhookURL != "" {
		if err := validateWebhookURL(opts.WebhookURL); err != nil {
			return DeliveryOptions{}, err
		}
	}
	return opts, nil
}

func validateWebhookURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid Feishu webhook URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("invalid Feishu webhook URL: scheme must be https or http")
	}
	return nil
}

func redactWebhookURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "***"
	}
	path := strings.Trim(parsed.EscapedPath(), "/")
	parts := strings.Split(path, "/")
	last := ""
	if len(parts) > 0 {
		last = parts[len(parts)-1]
	}
	if len(last) > 8 {
		last = last[:4] + "..." + last[len(last)-4:]
	} else if last != "" {
		last = "***"
	}
	return parsed.Scheme + "://" + parsed.Host + "/.../" + last
}

func redactToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func redactResourceURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "***"
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		switch parts[i] {
		case "wiki", "docx", "base", "folder":
			parts[i+1] = redactToken(parts[i+1])
		}
	}
	parsed.Path = "/" + strings.Join(parts, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func parseList(value string) []string {
	parts := strings.Split(value, ",")
	seen := map[string]bool{}
	result := []string{}
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" || seen[part] {
			continue
		}
		if part == "pulls" {
			part = "prs"
		}
		seen[part] = true
		result = append(result, part)
	}
	return result
}

func hasItem(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
