package webhook

import (
	"fmt"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List webhooks",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/webhooks", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a webhook",
			Flags: []common.Flag{
				{Name: "url", Short: "u", Usage: "Webhook target URL", Required: true},
				{Name: "type", Short: "t", Usage: "Webhook provider type", Default: "gitea"},
				{Name: "events", Short: "e", Usage: "Comma-separated events, for example: push,create,delete", Required: true},
				{Name: "secret", Short: "s", Usage: "Webhook secret"},
				{Name: "content-type", Usage: "POST content type: json or form", Default: "json"},
				{Name: "http-method", Usage: "Delivery method: GET or POST", Default: "POST"},
				{Name: "branch-filter", Usage: "Branch filter glob", Default: "*"},
				{Name: "active", Short: "a", Usage: "Enable webhook (true/false)", Default: "true"},
			},
			Run: runCreateWebhook,
		},
		{
			Name:        "view",
			Description: "Show webhook details",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Webhook ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/webhooks/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update a webhook",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Webhook ID", Required: true},
				{Name: "url", Short: "u", Usage: "Webhook target URL"},
				{Name: "type", Short: "t", Usage: "Webhook provider type"},
				{Name: "events", Short: "e", Usage: "Comma-separated events"},
				{Name: "secret", Short: "s", Usage: "Webhook secret"},
				{Name: "content-type", Usage: "POST content type: json or form"},
				{Name: "http-method", Usage: "Delivery method: GET or POST"},
				{Name: "branch-filter", Usage: "Branch filter glob"},
				{Name: "active", Short: "a", Usage: "Enable webhook (true/false)"},
			},
			Run: runUpdateWebhook,
		},
		{
			Name:        "delete",
			Description: "Delete a webhook",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Webhook ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/webhooks/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "test",
			Description: "Trigger a webhook test delivery",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Webhook ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/webhooks/%s/tests", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func runCreateWebhook(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	payload, err := buildWebhookPayload(ctx, nil, true)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/webhooks", payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runUpdateWebhook(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}

	currentEnv, err := ctx.CallAPI("GET", fmt.Sprintf("%s/webhooks/%s", ctx.RepoPath(), id), nil)
	if err != nil {
		return err
	}
	current, _ := currentEnv.Data.(map[string]interface{})

	payload, err := buildWebhookPayload(ctx, current, false)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("PUT", fmt.Sprintf("%s/webhooks/%s", ctx.RepoPath(), id), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func buildWebhookPayload(ctx *common.RuntimeContext, current map[string]interface{}, requireEvents bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{}

	if v := ctx.Arg("type"); v != "" {
		payload["type"] = v
	} else if current != nil {
		if v, ok := current["type"].(string); ok && v != "" {
			payload["type"] = v
		}
	}

	if v := ctx.Arg("url"); v != "" {
		payload["url"] = v
	} else if current != nil {
		if v, ok := current["url"].(string); ok && v != "" {
			payload["url"] = v
		}
	}

	if v := ctx.Arg("content-type"); v != "" {
		payload["content_type"] = v
	} else if current != nil {
		if v, ok := current["content_type"].(string); ok && v != "" {
			payload["content_type"] = v
		}
	}

	if v := ctx.Arg("http-method"); v != "" {
		payload["http_method"] = v
	} else if current != nil {
		if v, ok := current["http_method"].(string); ok && v != "" {
			payload["http_method"] = v
		}
	}

	if v := ctx.Arg("secret"); v != "" {
		payload["secret"] = v
	} else if current != nil {
		if v, ok := current["secret"].(string); ok && v != "" {
			payload["secret"] = v
		}
	}

	if v := ctx.Arg("branch-filter"); v != "" {
		payload["branch_filter"] = v
	} else if current != nil {
		if v, ok := current["branch_filter"].(string); ok && v != "" {
			payload["branch_filter"] = v
		}
	}

	eventsValue := ctx.Arg("events")
	if eventsValue != "" {
		events, err := parseCommaList(eventsValue)
		if err != nil {
			return nil, err
		}
		payload["events"] = events
	} else if current != nil {
		if events, ok := normalizeStringSlice(current["events"]); ok {
			payload["events"] = events
		}
	}

	activeValue := ctx.Arg("active")
	if activeValue != "" {
		active, err := parseBoolString(activeValue)
		if err != nil {
			return nil, err
		}
		payload["active"] = active
	} else if current != nil {
		if active, ok := current["active"].(bool); ok {
			payload["active"] = active
		}
	}

	if requireEvents {
		if _, ok := payload["events"]; !ok {
			return nil, fmt.Errorf("required flag --events is missing")
		}
	}

	if _, ok := payload["type"]; !ok {
		payload["type"] = "gitea"
	}
	if _, ok := payload["content_type"]; !ok {
		payload["content_type"] = "json"
	}
	if _, ok := payload["http_method"]; !ok {
		payload["http_method"] = "POST"
	}
	if _, ok := payload["branch_filter"]; !ok {
		payload["branch_filter"] = "*"
	}
	if _, ok := payload["active"]; !ok {
		payload["active"] = true
	}

	return payload, nil
}

func parseCommaList(value string) ([]string, error) {
	items := strings.Split(value, ",")
	results := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		results = append(results, value)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no values provided")
	}
	return results, nil
}

func normalizeStringSlice(value interface{}) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		return v, true
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			items = append(items, s)
		}
		return items, len(items) > 0
	default:
		return nil, false
	}
}

func parseBoolString(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q: use true or false", value)
	}
}
