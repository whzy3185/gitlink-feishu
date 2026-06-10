package message

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	messageTypeNotification = "notification"
	messageTypeAtme         = "atme"
	messageTypeAll          = "all"

	messageStatusAll    = "all"
	messageStatusUnread = "unread"
	messageStatusRead   = "read"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

type currentUserResponse struct {
	Login string `json:"login"`
}

type listResponse struct {
	TotalCount         int          `json:"total_count"`
	Type               string       `json:"type"`
	UnreadNotification int          `json:"unread_notification"`
	UnreadAtme         int          `json:"unread_atme"`
	Messages           []messageRow `json:"messages"`
}

type messageRow struct {
	ID              int64                  `json:"id"`
	Status          int                    `json:"status"`
	Content         string                 `json:"content"`
	NotificationURL string                 `json:"notification_url"`
	Source          string                 `json:"source"`
	CreatedAt       string                 `json:"created_at"`
	TimeAgo         string                 `json:"time_ago"`
	Type            string                 `json:"type"`
	Sender          map[string]interface{} `json:"sender"`
}

type messageOutput struct {
	ID              int64                  `json:"id"`
	Type            string                 `json:"type"`
	Status          int                    `json:"status"`
	Read            bool                   `json:"read"`
	Source          string                 `json:"source,omitempty"`
	Content         string                 `json:"content"`
	ContentText     string                 `json:"content_text"`
	NotificationURL string                 `json:"notification_url,omitempty"`
	CreatedAt       string                 `json:"created_at,omitempty"`
	TimeAgo         string                 `json:"time_ago,omitempty"`
	Sender          map[string]interface{} `json:"sender,omitempty"`
}

// Shortcuts returns message management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List user messages with filters and plain-text content",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "Target user login (defaults to current authenticated user)"},
				{Name: "type", Short: "t", Usage: "Message type: notification, atme, or all", Default: messageTypeAll},
				{Name: "status", Usage: "Message status: unread, read, or all", Default: messageStatusAll},
				{Name: "page", Usage: "Page number", Default: "1"},
				{Name: "limit", Usage: "Items per page", Default: "20"},
			},
			Run: runList,
		},
		{
			Name:        "stats",
			Description: "Show unread message counters for a user",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "Target user login (defaults to current authenticated user)"},
				{Name: "type", Short: "t", Usage: "Message type: notification, atme, or all", Default: messageTypeAll},
			},
			Run: runStats,
		},
		{
			Name:        "read",
			Description: "Mark selected messages as read",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "Target user login (defaults to current authenticated user)"},
				{Name: "type", Short: "t", Usage: "Message type: notification or atme", Required: true},
				{Name: "ids", Usage: "Comma-separated message IDs"},
				{Name: "all", Usage: "Mark all unread messages of the selected type as read", Bool: true, Default: "false"},
				{Name: "dry-run", Usage: "Preview the request without sending it", Bool: true, Default: "false"},
			},
			Run: runRead,
		},
		{
			Name:        "delete",
			Description: "Delete selected messages",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "Target user login (defaults to current authenticated user)"},
				{Name: "type", Short: "t", Usage: "Message type: notification or atme", Required: true},
				{Name: "ids", Usage: "Comma-separated message IDs"},
				{Name: "all", Usage: "Delete all unread messages of the selected type", Bool: true, Default: "false"},
				{Name: "dry-run", Usage: "Preview the request without sending it", Bool: true, Default: "false"},
			},
			Run: runDelete,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	login, err := resolveTargetLogin(ctx)
	if err != nil {
		return err
	}
	query, messageType, status, page, limit, err := buildListQuery(ctx.Arg("type"), ctx.Arg("status"), ctx.Arg("page"), ctx.Arg("limit"))
	if err != nil {
		return err
	}
	response, err := fetchMessages(ctx, login, query)
	if err != nil {
		return err
	}

	return ctx.OutputData(map[string]interface{}{
		"action":              "list_messages",
		"login":               login,
		"type":                messageType,
		"status":              status,
		"page":                page,
		"limit":               limit,
		"total_count":         response.TotalCount,
		"unread_notification": response.UnreadNotification,
		"unread_atme":         response.UnreadAtme,
		"messages":            normalizeMessages(response.Messages),
	})
}

func runStats(ctx *common.RuntimeContext) error {
	login, err := resolveTargetLogin(ctx)
	if err != nil {
		return err
	}
	messageType, err := parseListType(ctx.Arg("type"))
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("page", "1")
	query.Set("limit", "1")
	if messageType != messageTypeAll {
		query.Set("type", messageType)
	}

	response, err := fetchMessages(ctx, login, query)
	if err != nil {
		return err
	}

	return ctx.OutputData(map[string]interface{}{
		"action":              "message_stats",
		"login":               login,
		"type":                messageType,
		"total_count":         response.TotalCount,
		"unread_notification": response.UnreadNotification,
		"unread_atme":         response.UnreadAtme,
		"unread_total":        response.UnreadNotification + response.UnreadAtme,
	})
}

func runRead(ctx *common.RuntimeContext) error {
	return mutateMessages(ctx, "read_messages", httpMutation{
		Method:     "POST",
		PathSuffix: "/read",
	})
}

func runDelete(ctx *common.RuntimeContext) error {
	return mutateMessages(ctx, "delete_messages", httpMutation{
		Method:     "DELETE",
		PathSuffix: "",
	})
}

type httpMutation struct {
	Method     string
	PathSuffix string
}

func mutateMessages(ctx *common.RuntimeContext, action string, mutation httpMutation) error {
	login, err := resolveTargetLogin(ctx)
	if err != nil {
		return err
	}
	messageType, err := parseMutationType(ctx.Arg("type"))
	if err != nil {
		return err
	}
	ids, mode, err := parseMutationIDs(ctx.Arg("ids"), parseBoolArg(ctx.Arg("all")))
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"action":     action,
		"login":      login,
		"type":       messageType,
		"mode":       mode,
		"ids":        ids,
		"dry_run":    parseBoolArg(ctx.Arg("dry-run")),
		"item_count": len(ids),
	}
	if mode == "all" {
		result["item_count"] = "all"
	}

	if parseBoolArg(ctx.Arg("dry-run")) {
		return ctx.OutputData(result)
	}

	payload := map[string]interface{}{
		"type": messageType,
		"ids":  ids,
	}
	env, err := ctx.CallAPI(mutation.Method, fmt.Sprintf("/api/users/%s/messages%s", login, mutation.PathSuffix), payload)
	if err != nil {
		return err
	}
	result["updated"] = env.Data
	return ctx.OutputData(result)
}

func resolveTargetLogin(ctx *common.RuntimeContext) (string, error) {
	if login := strings.TrimSpace(ctx.Arg("login")); login != "" {
		return login, nil
	}

	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("fetch current user: %w", err)
	}
	var current currentUserResponse
	if err := decodeEnvelopeData(env.Data, &current); err != nil {
		return "", fmt.Errorf("parse current user: %w", err)
	}
	if strings.TrimSpace(current.Login) == "" {
		return "", fmt.Errorf("current user response did not include a login")
	}
	return current.Login, nil
}

func buildListQuery(typeValue, statusValue, pageValue, limitValue string) (url.Values, string, string, int, int, error) {
	messageType, err := parseListType(typeValue)
	if err != nil {
		return nil, "", "", 0, 0, err
	}
	status, statusCode, err := parseListStatus(statusValue)
	if err != nil {
		return nil, "", "", 0, 0, err
	}
	page, err := parsePositiveInt(pageValue, "page")
	if err != nil {
		return nil, "", "", 0, 0, err
	}
	limit, err := parsePositiveInt(limitValue, "limit")
	if err != nil {
		return nil, "", "", 0, 0, err
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(limit))
	if messageType != messageTypeAll {
		query.Set("type", messageType)
	}
	if statusCode != 0 {
		query.Set("status", strconv.Itoa(statusCode))
	}
	return query, messageType, status, page, limit, nil
}

func fetchMessages(ctx *common.RuntimeContext, login string, query url.Values) (*listResponse, error) {
	env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/api/users/%s/messages", login), query)
	if err != nil {
		return nil, fmt.Errorf("fetch messages: %w", err)
	}
	var response listResponse
	if err := decodeEnvelopeData(env.Data, &response); err != nil {
		return nil, fmt.Errorf("parse message list: %w", err)
	}
	return &response, nil
}

func normalizeMessages(rows []messageRow) []messageOutput {
	items := make([]messageOutput, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageOutput{
			ID:              row.ID,
			Type:            row.Type,
			Status:          row.Status,
			Read:            row.Status == 2,
			Source:          row.Source,
			Content:         row.Content,
			ContentText:     normalizeMessageText(row.Content),
			NotificationURL: row.NotificationURL,
			CreatedAt:       row.CreatedAt,
			TimeAgo:         row.TimeAgo,
			Sender:          row.Sender,
		})
	}
	return items
}

func normalizeMessageText(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}

func parseListType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = messageTypeAll
	}
	switch value {
	case messageTypeAll, messageTypeNotification, messageTypeAtme:
		return value, nil
	default:
		return "", fmt.Errorf("invalid --type value %q", value)
	}
}

func parseMutationType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case messageTypeNotification, messageTypeAtme:
		return value, nil
	default:
		return "", fmt.Errorf("invalid --type value %q", value)
	}
}

func parseListStatus(value string) (string, int, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = messageStatusAll
	}
	switch value {
	case messageStatusAll:
		return value, 0, nil
	case messageStatusUnread:
		return value, 1, nil
	case messageStatusRead:
		return value, 2, nil
	default:
		return "", 0, fmt.Errorf("invalid --status value %q", value)
	}
}

func parsePositiveInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s value %q", name, value)
	}
	return parsed, nil
}

func parseMutationIDs(value string, all bool) ([]int64, string, error) {
	if all {
		if strings.TrimSpace(value) != "" {
			return nil, "", fmt.Errorf("--ids cannot be used together with --all")
		}
		return []int64{-1}, "all", nil
	}

	parts := strings.Split(strings.TrimSpace(value), ",")
	ids := make([]int64, 0, len(parts))
	seen := map[int64]bool{}
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		id, err := strconv.ParseInt(token, 10, 64)
		if err != nil || id <= 0 {
			return nil, "", fmt.Errorf("invalid --ids value %q", token)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, "", fmt.Errorf("one of --ids or --all is required")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, "selected", nil
}

func parseBoolArg(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func decodeEnvelopeData(data interface{}, target interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
