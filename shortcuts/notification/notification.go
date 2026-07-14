package notification

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var messageTypes = map[string]string{
	"notification": "notification",
	"atme":         "atme",
}

var listStatuses = map[string]string{
	"unread": "1",
	"read":   "2",
}

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.notification.list.short"),
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: tr.T("flag.notification.user")},
				{Name: "type", Short: "t", Usage: tr.T("flag.notification.type_all"), Default: "all"},
				{Name: "status", Short: "s", Usage: tr.T("flag.notification.status"), Default: "all"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runList,
		},
		{
			Name:        "read",
			Description: tr.T("cmd.notification.read.short"),
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: tr.T("flag.notification.user")},
				{Name: "type", Short: "t", Usage: tr.T("flag.notification.type"), Required: true},
				{Name: "ids", Short: "i", Usage: tr.T("flag.notification.ids_read"), Required: true},
			},
			Run: runRead,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.notification.delete.short"),
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: tr.T("flag.notification.user")},
				{Name: "type", Short: "t", Usage: tr.T("flag.notification.type"), Required: true},
				{Name: "ids", Short: "i", Usage: tr.T("flag.notification.ids"), Required: true},
			},
			Run: runDelete,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	user, err := resolveUserLogin(ctx)
	if err != nil {
		return err
	}
	query, err := listQuery(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPIWithQuery("GET", messagesPath(user), query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRead(ctx *common.RuntimeContext) error {
	user, payload, err := messagePayload(ctx, true)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("POST", messagesPath(user)+"/read", payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDelete(ctx *common.RuntimeContext) error {
	user, payload, err := messagePayload(ctx, false)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("DELETE", messagesPath(user), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func messagesPath(user string) string {
	return fmt.Sprintf("/users/%s/messages", url.PathEscape(user))
}

func listQuery(ctx *common.RuntimeContext) (url.Values, error) {
	page, err := positiveInt(defaultString(ctx.Arg("page"), "1"), "page")
	if err != nil {
		return nil, err
	}
	limit, err := positiveInt(defaultString(ctx.Arg("limit"), "20"), "limit")
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(limit))
	if typ, err := normalizeOptionalType(ctx.Arg("type")); err != nil {
		return nil, err
	} else if typ != "" {
		query.Set("type", typ)
	}
	if status, err := normalizeStatus(ctx.Arg("status")); err != nil {
		return nil, err
	} else if status != "" {
		query.Set("status", status)
	}
	return query, nil
}

func messagePayload(ctx *common.RuntimeContext, allowAllUnread bool) (string, map[string]interface{}, error) {
	user, err := resolveUserLogin(ctx)
	if err != nil {
		return "", nil, err
	}
	typ, err := normalizeRequiredType(ctx.Arg("type"))
	if err != nil {
		return "", nil, err
	}
	ids, err := parseIDs(ctx.Arg("ids"), allowAllUnread)
	if err != nil {
		return "", nil, err
	}
	return user, map[string]interface{}{
		"type": typ,
		"ids":  ids,
	}, nil
}

func resolveUserLogin(ctx *common.RuntimeContext) (string, error) {
	if user := strings.TrimSpace(ctx.Arg("user")); user != "" {
		return user, nil
	}
	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("resolve current user: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("resolve current user: unexpected response")
	}
	login, _ := data["login"].(string)
	if strings.TrimSpace(login) == "" {
		return "", fmt.Errorf("resolve current user: login is missing")
	}
	return strings.TrimSpace(login), nil
}

func normalizeOptionalType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "all" {
		return "", nil
	}
	return normalizeRequiredType(value)
}

func normalizeRequiredType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if typ, ok := messageTypes[value]; ok {
		return typ, nil
	}
	return "", fmt.Errorf("invalid --type %q: use notification or atme", value)
}

func normalizeStatus(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "all" {
		return "", nil
	}
	if status, ok := listStatuses[value]; ok {
		return status, nil
	}
	return "", fmt.Errorf("invalid --status %q: use unread, read, or all", value)
}

func parseIDs(value string, allowAllUnread bool) ([]int, error) {
	parts := strings.Split(value, ",")
	ids := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		raw := strings.TrimSpace(part)
		if raw == "" {
			continue
		}
		id, err := strconv.Atoi(raw)
		if err != nil || id == 0 || id < -1 {
			return nil, fmt.Errorf("invalid --ids value %q: use positive integer IDs", raw)
		}
		if id == -1 && !allowAllUnread {
			return nil, fmt.Errorf("invalid --ids value -1: delete requires explicit message IDs")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("required flag --ids is empty")
	}
	return ids, nil
}

func positiveInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", name, value)
	}
	return parsed, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
