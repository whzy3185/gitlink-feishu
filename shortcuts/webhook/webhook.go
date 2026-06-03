package webhook

import (
	"fmt"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var allowedWebhookTypes = map[string]bool{
	"gitea": true, "slack": true, "discord": true, "dingtalk": true, "telegram": true,
	"msteams": true, "feishu": true, "matrix": true, "jianmu": true, "softbot": true,
}

var allowedWebhookContentTypes = map[string]bool{"json": true, "form": true}
var allowedWebhookMethods = map[string]bool{"GET": true, "POST": true}

var allowedWebhookEvents = map[string]bool{
	"push": true, "create": true, "delete": true,
	"issues_only": true, "issue_assign": true, "issue_label": true, "issue_comment": true,
	"pull_request_only": true, "pull_request_assign": true, "pull_request_comment": true,
}

// Shortcuts returns webhook management shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.webhook.list.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", webhookPath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.webhook.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", webhookItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.webhook.create.short"),
			Flags: []common.Flag{
				{Name: "url", Short: "u", Usage: tr.T("flag.webhook.url"), Required: true},
				{Name: "events", Short: "e", Usage: tr.T("flag.webhook.events"), Required: true},
				{Name: "type", Short: "t", Usage: tr.T("flag.webhook.type"), Default: "gitea"},
				{Name: "content-type", Usage: tr.T("flag.webhook.content_type"), Default: "json"},
				{Name: "http-method", Usage: tr.T("flag.webhook.http_method"), Default: "POST"},
				{Name: "secret", Short: "s", Usage: tr.T("flag.webhook.secret")},
				{Name: "branch-filter", Usage: tr.T("flag.webhook.branch_filter"), Default: "*"},
				{Name: "active", Usage: tr.T("flag.webhook.active"), Default: "true"},
			},
			Run: runCreate,
		},
		{
			Name:        "update",
			Description: tr.T("cmd.webhook.update.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
				{Name: "url", Short: "u", Usage: tr.T("flag.webhook.url")},
				{Name: "events", Short: "e", Usage: tr.T("flag.webhook.events")},
				{Name: "type", Short: "t", Usage: tr.T("flag.webhook.type")},
				{Name: "content-type", Usage: tr.T("flag.webhook.content_type")},
				{Name: "http-method", Usage: tr.T("flag.webhook.http_method")},
				{Name: "secret", Short: "s", Usage: tr.T("flag.webhook.secret_update")},
				{Name: "branch-filter", Usage: tr.T("flag.webhook.branch_filter")},
				{Name: "active", Usage: tr.T("flag.webhook.active")},
			},
			Run: runUpdate,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.webhook.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", webhookItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "test",
			Description: tr.T("cmd.webhook.test.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/tests", webhookItemPath(ctx, id)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tasks",
			Description: tr.T("cmd.webhook.tasks.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/hooktasks", webhookItemPath(ctx, id)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func runCreate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	payload, err := webhookPayloadFromArgs(ctx, nil)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("POST", webhookPath(ctx), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}

	current, err := fetchWebhook(ctx, id)
	if err != nil {
		return fmt.Errorf("fetch webhook: %w", err)
	}
	payload, err := webhookPayloadFromArgs(ctx, current)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("PUT", webhookItemPath(ctx, id), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func webhookPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/webhooks", ctx.Owner, ctx.Repo)
}

func webhookItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", webhookPath(ctx), id)
}

func fetchWebhook(ctx *common.RuntimeContext, id string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", webhookItemPath(ctx, id), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse webhook data")
	}
	return data, nil
}

func webhookPayloadFromArgs(ctx *common.RuntimeContext, current map[string]interface{}) (map[string]interface{}, error) {
	url := firstNonEmpty(ctx.Arg("url"), stringFromMap(current, "url"))
	if url == "" {
		return nil, fmt.Errorf("required flag --url is missing")
	}

	eventValue := ctx.Arg("events")
	var events []string
	var err error
	if eventValue != "" {
		events, err = parseWebhookEvents(eventValue)
		if err != nil {
			return nil, err
		}
	} else {
		events, err = eventsFromMap(current)
		if err != nil {
			return nil, err
		}
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("required flag --events is missing")
	}

	webhookType := strings.ToLower(firstNonEmpty(ctx.Arg("type"), stringFromMap(current, "type"), "gitea"))
	if err := validateOneOf("type", webhookType, allowedWebhookTypes); err != nil {
		return nil, err
	}
	contentType := strings.ToLower(firstNonEmpty(ctx.Arg("content-type"), stringFromMap(current, "content_type"), "json"))
	if err := validateOneOf("content-type", contentType, allowedWebhookContentTypes); err != nil {
		return nil, err
	}
	httpMethod := strings.ToUpper(firstNonEmpty(ctx.Arg("http-method"), stringFromMap(current, "http_method"), "POST"))
	if err := validateOneOf("http-method", httpMethod, allowedWebhookMethods); err != nil {
		return nil, err
	}
	branchFilter := firstNonEmpty(ctx.Arg("branch-filter"), stringFromMap(current, "branch_filter"), "*")
	active, err := activeFromArgs(ctx.Arg("active"), current)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"type":          webhookType,
		"active":        active,
		"content_type":  contentType,
		"http_method":   httpMethod,
		"url":           url,
		"branch_filter": branchFilter,
		"events":        events,
	}
	if secret := firstNonEmpty(ctx.Arg("secret"), stringFromMap(current, "secret")); secret != "" {
		payload["secret"] = secret
	}
	return payload, nil
}

func parseWebhookEvents(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	events := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		event := strings.TrimSpace(part)
		if event == "" {
			continue
		}
		if !allowedWebhookEvents[event] {
			return nil, fmt.Errorf("invalid --events value %q", event)
		}
		if seen[event] {
			continue
		}
		seen[event] = true
		events = append(events, event)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("required flag --events is missing")
	}
	return events, nil
}

func eventsFromMap(values map[string]interface{}) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	raw, ok := values["events"]
	if !ok || raw == nil {
		return nil, nil
	}
	switch events := raw.(type) {
	case []interface{}:
		result := make([]string, 0, len(events))
		for _, event := range events {
			name, ok := event.(string)
			if !ok {
				return nil, fmt.Errorf("failed to parse webhook events")
			}
			result = append(result, name)
		}
		return result, nil
	case []string:
		return events, nil
	default:
		return nil, fmt.Errorf("failed to parse webhook events")
	}
}

func activeFromArgs(value string, current map[string]interface{}) (bool, error) {
	if value != "" {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("invalid --active value %q: use true or false", value)
		}
	}
	if current != nil {
		if active, ok := current["active"].(bool); ok {
			return active, nil
		}
		if active, ok := current["is_active"].(bool); ok {
			return active, nil
		}
	}
	return true, nil
}

func stringFromMap(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func validateOneOf(name, value string, allowed map[string]bool) error {
	if allowed[value] {
		return nil
	}
	return fmt.Errorf("invalid --%s value %q", name, value)
}
