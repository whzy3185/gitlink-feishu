// Package notification implements GitLink user message shortcuts.
package notification

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const allUnreadMessageID = -1

// Shortcuts returns user notification/message shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	userFlag := common.Flag{Name: "user", Short: "u", Usage: tr.T("flag.notification.user")}
	typeFlag := common.Flag{Name: "type", Short: "t", Usage: tr.T("flag.notification.type")}

	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.notification.list.short"),
			Flags: []common.Flag{
				userFlag,
				typeFlag,
				{Name: "status", Short: "s", Usage: tr.T("flag.notification.status")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runList,
		},
		{
			Name:        "read",
			Description: tr.T("cmd.notification.read.short"),
			Flags: []common.Flag{
				userFlag,
				typeFlag,
				{Name: "ids", Short: "i", Usage: tr.T("flag.notification.ids")},
				{Name: "all-unread", Usage: tr.T("flag.notification.all_unread"), Bool: true, Default: "false"},
				{Name: "dry-run", Usage: tr.T("flag.notification.dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.notification.yes"), Bool: true, Default: "false"},
			},
			Run: runRead,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.notification.delete.short"),
			Flags: []common.Flag{
				userFlag,
				typeFlag,
				{Name: "ids", Short: "i", Usage: tr.T("flag.notification.ids"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.notification.dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.notification.yes"), Bool: true, Default: "false"},
			},
			Run: runDelete,
		},
		{
			Name:        "send-atme",
			Description: tr.T("cmd.notification.send_atme.short"),
			Flags: []common.Flag{
				userFlag,
				{Name: "receivers", Short: "r", Usage: tr.T("flag.notification.receivers"), Required: true},
				{Name: "atmeable-type", Usage: tr.T("flag.notification.atmeable_type"), Required: true},
				{Name: "atmeable-id", Usage: tr.T("flag.notification.atmeable_id"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.notification.dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.notification.yes"), Bool: true, Default: "false"},
			},
			Run: runSendAtme,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	user, err := resolveUser(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	setQueryIfPresent(q, "type", ctx.Arg("type"))
	setQueryIfPresent(q, "status", normalizeStatus(ctx.Arg("status")))
	q.Set("page", firstNonEmpty(ctx.Arg("page"), "1"))
	q.Set("limit", firstNonEmpty(ctx.Arg("limit"), "20"))
	env, err := ctx.CallAPIWithQuery("GET", messagesPath(user), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRead(ctx *common.RuntimeContext) error {
	user, err := resolveUser(ctx)
	if err != nil {
		return err
	}
	payload, err := readPayload(ctx)
	if err != nil {
		return err
	}
	path := messagesReadPath(user)
	if ctx.Arg("dry-run") == "true" {
		return writeDryRun(ctx, "read_notifications", "POST", path, user, payload)
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("marking messages as read changes remote state; run --dry-run first, then pass --yes to execute")
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDelete(ctx *common.RuntimeContext) error {
	user, err := resolveUser(ctx)
	if err != nil {
		return err
	}
	ids, err := parseIDs(ctx.Arg("ids"))
	if err != nil {
		return err
	}
	payload := messageActionPayload(ctx.Arg("type"), ids)
	path := messagesPath(user)
	if ctx.Arg("dry-run") == "true" {
		return writeDryRun(ctx, "delete_notifications", "DELETE", path, user, payload)
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("deleting messages is destructive; run --dry-run first, then pass --yes to execute")
	}
	env, err := ctx.CallAPI("DELETE", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runSendAtme(ctx *common.RuntimeContext) error {
	user, err := resolveUser(ctx)
	if err != nil {
		return err
	}
	payload, err := sendAtmePayload(ctx)
	if err != nil {
		return err
	}
	path := messagesPath(user)
	if ctx.Arg("dry-run") == "true" {
		return writeDryRun(ctx, "send_atme", "POST", path, user, payload)
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("sending @ messages changes remote state; run --dry-run first, then pass --yes to execute")
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func readPayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	if ctx.Arg("all-unread") == "true" {
		if strings.TrimSpace(ctx.Arg("ids")) != "" {
			return nil, fmt.Errorf("--ids and --all-unread cannot be used together")
		}
		return messageActionPayload(ctx.Arg("type"), []int{allUnreadMessageID}), nil
	}
	ids, err := parseIDs(ctx.Arg("ids"))
	if err != nil {
		return nil, err
	}
	return messageActionPayload(ctx.Arg("type"), ids), nil
}

func messageActionPayload(messageType string, ids []int) map[string]interface{} {
	return map[string]interface{}{
		"type": firstNonEmpty(strings.TrimSpace(messageType), "notification"),
		"ids":  ids,
	}
}

func sendAtmePayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	receivers, err := parseStringList(ctx.Arg("receivers"), "--receivers")
	if err != nil {
		return nil, err
	}
	atmeableType, err := ctx.RequireArg("atmeable-type")
	if err != nil {
		return nil, err
	}
	atmeableID, err := parsePositiveInt(ctx.Arg("atmeable-id"), "--atmeable-id")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"type":            "atme",
		"receivers_login": receivers,
		"atmeable_type":   atmeableType,
		"atmeable_id":     atmeableID,
	}, nil
}

func writeDryRun(ctx *common.RuntimeContext, action, method, path, user string, payload map[string]interface{}) error {
	return ctx.OutputData(map[string]interface{}{
		"dry_run": true,
		"action":  action,
		"method":  method,
		"path":    path,
		"user":    user,
		"payload": payload,
	})
}

func resolveUser(ctx *common.RuntimeContext) (string, error) {
	if user := strings.TrimSpace(ctx.Arg("user")); user != "" {
		return user, nil
	}
	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", err
	}
	if login := extractLogin(env); login != "" {
		return login, nil
	}
	return "", fmt.Errorf("%s", ctx.Tr.T("error.notification.user_required"))
}

func extractLogin(env *output.Envelope) string {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return ""
	}
	if login, ok := data["login"].(string); ok {
		return login
	}
	return ""
}

func parseIDs(raw string) ([]int, error) {
	return parseIntList(raw, "--ids", true)
}

func parseIntList(raw, flag string, positiveOnly bool) ([]int, error) {
	parts := strings.Split(raw, ",")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: use comma-separated integers", flag, part)
		}
		if positiveOnly && value <= 0 {
			return nil, fmt.Errorf("invalid %s value %q: use a positive integer", flag, part)
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%s must include at least one id", flag)
	}
	return values, nil
}

func parsePositiveInt(raw, flag string) (int, error) {
	values, err := parseIntList(raw, flag, true)
	if err != nil {
		return 0, err
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("%s must include exactly one id", flag)
	}
	return values[0], nil
}

func parseStringList(raw, flag string) ([]string, error) {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%s must include at least one value", flag)
	}
	return values, nil
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "unread":
		return "1"
	case "read":
		return "2"
	default:
		return strings.TrimSpace(status)
	}
}

func messagesPath(user string) string {
	return fmt.Sprintf("/api/users/%s/messages", url.PathEscape(user))
}

func messagesReadPath(user string) string {
	return fmt.Sprintf("%s/read", messagesPath(user))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func setQueryIfPresent(q url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		q.Set(key, strings.TrimSpace(value))
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
