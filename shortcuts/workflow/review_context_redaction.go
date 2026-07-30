package workflow

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/client"
)

const maxReviewContextDiagnosticLength = 512

var (
	reviewContextURLPattern     = regexp.MustCompile(`https?://[^\s"'<>]+`)
	reviewContextSecretPatterns = []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{
			pattern:     regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s,;]+`),
			replacement: "${1}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(cookie|set-cookie|x-auth-token|private-token|access_token|refresh_token|client_secret|app_secret)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
		{
			pattern:     regexp.MustCompile(`(?i)(token|secret|password)(\s*[:=]\s*)[^\s,;]+`),
			replacement: "${1}${2}***",
		},
	}
)

func safeReviewContextErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		code := redactReviewContextDiagnostic(fmt.Sprint(apiErr.Code))
		if code == "" || code == "<nil>" {
			code = "unknown"
		}
		if apiErr.StatusCode > 0 {
			return fmt.Sprintf("GitLink API request failed (HTTP %d, code %s)", apiErr.StatusCode, code)
		}
		return fmt.Sprintf("GitLink API request failed (code %s)", code)
	}
	return redactReviewContextDiagnostic(err.Error())
}

func sanitizeReviewContextDiagnostics(context *ReviewContext) {
	if context == nil {
		return
	}
	for i := range context.FetchErrors {
		context.FetchErrors[i].Message = redactReviewContextDiagnostic(context.FetchErrors[i].Message)
	}
	for i := range context.Notes {
		context.Notes[i].Note = redactReviewContextDiagnostic(context.Notes[i].Note)
	}
}

func redactReviewContextDiagnostic(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
	if value == "" {
		return ""
	}
	value = reviewContextURLPattern.ReplaceAllStringFunc(value, redactReviewContextDiagnosticURL)
	for _, rule := range reviewContextSecretPatterns {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	for _, name := range []string{
		"GITLINK_TOKEN",
		"GITLINK_ACCESS_TOKEN",
		"FEISHU_APP_SECRET",
		"FEISHU_WEBHOOK_SECRET",
		"FEISHU_WEBHOOK_URL",
	} {
		if secret := strings.TrimSpace(os.Getenv(name)); len(secret) >= 6 {
			value = strings.ReplaceAll(value, secret, "***")
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > maxReviewContextDiagnosticLength {
		value = string(runes[:maxReviewContextDiagnosticLength]) + "..."
	}
	return value
}

func redactReviewContextDiagnosticURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "https://..."
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := range parts {
		if i > 0 && (strings.EqualFold(parts[i-1], "hook") || strings.EqualFold(parts[i-1], "webhook")) {
			parts[i] = "***"
		}
	}
	parsed.Path = "/" + strings.Join(parts, "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		parsed.Path = ""
	}
	return parsed.String()
}
