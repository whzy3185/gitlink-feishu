package notification

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var allowedMessageTypes = map[string]bool{"notification": true, "atme": true}
var allowedAtmeableTypes = map[string]bool{"Journal": true, "Issue": true, "PullRequest": true}

// Shortcuts returns notification and message OpenAPI shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List user messages and notifications",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name", Required: true},
				{Name: "type", Short: "t", Usage: "Message type: notification or atme"},
				{Name: "status", Short: "s", Usage: "Read status: unread/1 or read/2"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: runList,
		},
		{
			Name:        "mark-read",
			Description: "Mark messages as read by IDs or all unread messages",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name", Required: true},
				{Name: "ids", Short: "i", Usage: "Comma-separated message IDs, for example: 101,102"},
				{Name: "all-unread", Usage: "Mark all unread messages as read using the OpenAPI -1 sentinel", Bool: true, Default: "false"},
				{Name: "type", Short: "t", Usage: "Message type: notification or atme"},
				{Name: "dry-run", Usage: "Preview the request body without changing messages", Bool: true, Default: "false"},
			},
			Run: runMarkRead,
		},
		{
			Name:        "delete",
			Description: "Delete messages by IDs",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name", Required: true},
				{Name: "ids", Short: "i", Usage: "Comma-separated message IDs, for example: 101,102", Required: true},
				{Name: "type", Short: "t", Usage: "Message type: notification or atme"},
				{Name: "dry-run", Usage: "Preview the request body without deleting messages", Bool: true, Default: "false"},
			},
			Run: runDelete,
		},
		{
			Name:        "create-atme",
			Description: "Create @me notifications for users on an Issue, PullRequest, or Journal",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name used in the API path", Required: true},
				{Name: "receivers", Short: "r", Usage: "Comma-separated receiver login names", Required: true},
				{Name: "atmeable-type", Usage: "Mention target type: Journal, Issue, or PullRequest", Required: true},
				{Name: "atmeable-id", Usage: "Mention target database ID", Required: true},
				{Name: "dry-run", Usage: "Preview the request body without creating messages", Bool: true, Default: "false"},
			},
			Run: runCreateAtme,
		},
		{
			Name:        "platform-settings",
			Description: "List platform message setting templates",
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/template_message_settings", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "settings",
			Description: "List user message settings",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				user, err := ctx.RequireArg("user")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", userMessageSettingsPath(user), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "settings-update",
			Description: "Update user message settings while preserving unspecified keys",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "User login name", Required: true},
				{Name: "notification", Usage: "Comma-separated station-message settings, for example: Normal::Project=true,ManageProject::Issue=false"},
				{Name: "email", Usage: "Comma-separated email settings, for example: Normal::Project=false,ManageProject::Issue=true"},
				{Name: "dry-run", Usage: "Preview the merged settings without changing them", Bool: true, Default: "false"},
			},
			Run: runSettingsUpdate,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	user, err := ctx.RequireArg("user")
	if err != nil {
		return err
	}
	messageType, err := normalizeMessageType(ctx.Arg("type"))
	if err != nil {
		return err
	}
	status, err := normalizeMessageStatus(ctx.Arg("status"))
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("page", defaultString(ctx.Arg("page"), "1"))
	q.Set("limit", defaultString(ctx.Arg("limit"), "20"))
	if messageType != "" {
		q.Set("type", messageType)
	}
	if status != "" {
		q.Set("status", status)
	}

	env, err := ctx.CallAPIWithQuery("GET", userMessagesPath(user), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runMarkRead(ctx *common.RuntimeContext) error {
	user, payload, err := messageActionPayload(ctx, true)
	if err != nil {
		return err
	}
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(dryRunData("mark-read", userMessagesReadPath(user), payload))
	}
	env, err := ctx.CallAPI("POST", userMessagesReadPath(user), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDelete(ctx *common.RuntimeContext) error {
	user, payload, err := messageActionPayload(ctx, false)
	if err != nil {
		return err
	}
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(dryRunData("delete", userMessagesPath(user), payload))
	}
	env, err := ctx.CallAPI("DELETE", userMessagesPath(user), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCreateAtme(ctx *common.RuntimeContext) error {
	user, err := ctx.RequireArg("user")
	if err != nil {
		return err
	}
	receivers, err := parseStringList(ctx.Arg("receivers"))
	if err != nil {
		return err
	}
	if len(receivers) == 0 {
		return fmt.Errorf("required flag --receivers is missing")
	}
	atmeableType, err := normalizeAtmeableType(ctx.Arg("atmeable-type"))
	if err != nil {
		return err
	}
	atmeableID, err := parsePositiveInt(ctx.Arg("atmeable-id"), "atmeable-id")
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"type":            "atme",
		"receivers_login": receivers,
		"atmeable_type":   atmeableType,
		"atmeable_id":     atmeableID,
	}
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(dryRunData("create-atme", userMessagesPath(user), payload))
	}
	env, err := ctx.CallAPI("POST", userMessagesPath(user), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runSettingsUpdate(ctx *common.RuntimeContext) error {
	user, err := ctx.RequireArg("user")
	if err != nil {
		return err
	}
	notificationUpdates, err := parseBoolPairs(ctx.Arg("notification"))
	if err != nil {
		return err
	}
	emailUpdates, err := parseBoolPairs(ctx.Arg("email"))
	if err != nil {
		return err
	}
	if len(notificationUpdates) == 0 && len(emailUpdates) == 0 {
		return fmt.Errorf("at least one of --notification or --email is required")
	}

	current, err := fetchMessageSettings(ctx, user)
	if err != nil {
		return fmt.Errorf("fetch message settings: %w", err)
	}
	payload, err := mergedSettingsPayload(current, notificationUpdates, emailUpdates)
	if err != nil {
		return err
	}
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(dryRunData("settings-update", userMessageSettingsUpdatePath(user), payload))
	}
	env, err := ctx.CallAPI("POST", userMessageSettingsUpdatePath(user), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func userMessagesPath(user string) string {
	return fmt.Sprintf("/users/%s/messages", user)
}

func userMessagesReadPath(user string) string {
	return fmt.Sprintf("%s/read", userMessagesPath(user))
}

func userMessageSettingsPath(user string) string {
	return fmt.Sprintf("/users/%s/template_message_settings", user)
}

func userMessageSettingsUpdatePath(user string) string {
	return fmt.Sprintf("%s/update_setting", userMessageSettingsPath(user))
}

func messageActionPayload(ctx *common.RuntimeContext, allowAllUnread bool) (string, map[string]interface{}, error) {
	user, err := ctx.RequireArg("user")
	if err != nil {
		return "", nil, err
	}
	messageType, err := normalizeMessageType(ctx.Arg("type"))
	if err != nil {
		return "", nil, err
	}
	ids, err := parseMessageIDs(ctx.Arg("ids"), parseBool(ctx.Arg("all-unread")), allowAllUnread)
	if err != nil {
		return "", nil, err
	}
	payload := map[string]interface{}{"ids": ids}
	if messageType != "" {
		payload["type"] = messageType
	}
	return user, payload, nil
}

func normalizeMessageType(value string) (string, error) {
	messageType := strings.ToLower(strings.TrimSpace(value))
	if messageType == "" {
		return "", nil
	}
	if !allowedMessageTypes[messageType] {
		return "", fmt.Errorf("invalid --type %q: use notification or atme", value)
	}
	return messageType, nil
}

func normalizeMessageStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case "unread", "1":
		return "1", nil
	case "read", "2":
		return "2", nil
	default:
		return "", fmt.Errorf("invalid --status %q: use unread/1 or read/2", value)
	}
}

func normalizeAtmeableType(value string) (string, error) {
	atmeableType := strings.TrimSpace(value)
	if atmeableType == "" {
		return "", fmt.Errorf("required flag --atmeable-type is missing")
	}
	for allowed := range allowedAtmeableTypes {
		if strings.EqualFold(atmeableType, allowed) {
			return allowed, nil
		}
	}
	return "", fmt.Errorf("invalid --atmeable-type %q: use Journal, Issue, or PullRequest", value)
}

func parseMessageIDs(value string, allUnread bool, allowAllUnread bool) ([]int, error) {
	if allUnread && value != "" {
		return nil, fmt.Errorf("use either --ids or --all-unread, not both")
	}
	if allUnread {
		if !allowAllUnread {
			return nil, fmt.Errorf("--all-unread is only supported by notification +mark-read")
		}
		return []int{-1}, nil
	}
	parts, err := parseStringList(value)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("required flag --ids is missing")
	}
	ids := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		id, err := parsePositiveInt(part, "ids")
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

func parsePositiveInt(value, flagName string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", flagName, value)
	}
	return id, nil
}

func parseStringList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		if seen[text] {
			continue
		}
		seen[text] = true
		values = append(values, text)
	}
	return values, nil
}

func parseBoolPairs(value string) (map[string]bool, error) {
	pairs := map[string]bool{}
	if strings.TrimSpace(value) == "" {
		return pairs, nil
	}
	for _, part := range strings.Split(value, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		key, rawValue, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("invalid setting %q: use key=true or key=false", item)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid setting %q: key is empty", item)
		}
		parsed, err := strconv.ParseBool(strings.TrimSpace(rawValue))
		if err != nil {
			return nil, fmt.Errorf("invalid boolean value in %q: use true or false", item)
		}
		pairs[key] = parsed
	}
	return pairs, nil
}

func fetchMessageSettings(ctx *common.RuntimeContext, user string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", userMessageSettingsPath(user), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse message settings")
	}
	return data, nil
}

func mergedSettingsPayload(current map[string]interface{}, notificationUpdates, emailUpdates map[string]bool) (map[string]interface{}, error) {
	notificationBody, err := boolMapFromInterface(current["notification_body"])
	if err != nil {
		return nil, fmt.Errorf("parse notification_body: %w", err)
	}
	emailBody, err := boolMapFromInterface(current["email_body"])
	if err != nil {
		return nil, fmt.Errorf("parse email_body: %w", err)
	}
	for key, value := range notificationUpdates {
		notificationBody[key] = value
	}
	for key, value := range emailUpdates {
		emailBody[key] = value
	}
	return map[string]interface{}{
		"setting": map[string]interface{}{
			"notification_body": notificationBody,
			"email_body":        emailBody,
		},
	}, nil
}

func boolMapFromInterface(value interface{}) (map[string]bool, error) {
	body, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected object")
	}
	result := make(map[string]bool, len(body))
	for key, raw := range body {
		parsed, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("%s is not a boolean", key)
		}
		result[key] = parsed
	}
	return result, nil
}

func dryRunData(action, path string, payload map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"dry_run": true,
		"action":  action,
		"method":  dryRunMethod(action),
		"path":    path,
		"body":    payload,
	}
}

func dryRunMethod(action string) string {
	switch action {
	case "mark-read", "create-atme", "settings-update":
		return "POST"
	case "delete":
		return "DELETE"
	default:
		return ""
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}
