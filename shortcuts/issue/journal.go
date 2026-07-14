package issue

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newIssueCommentsShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "comments",
		Description: "List issue comments and operation journals",
		Flags: []common.Flag{
			{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
			{Name: "category", Short: "c", Usage: "Journal category: all, comment, or operate", Default: "all"},
			{Name: "keyword", Short: "k", Usage: "Search keyword"},
			{Name: "sort-by", Usage: "Sort field: created_on or updated_on"},
			{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
			{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
		},
		Run: runIssueComments,
	}
}

func newIssueCommentUpdateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "comment-update",
		Description: "Update an issue comment",
		Flags: []common.Flag{
			{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
			{Name: "comment-id", Short: "i", Usage: "Issue comment/journal ID", Required: true},
			{Name: "body", Short: "b", Usage: "New comment body", Required: true},
			{Name: "attachment-ids", Usage: "Comma-separated attachment IDs"},
			{Name: "receivers", Short: "r", Usage: "Comma-separated @ receiver login names"},
			{Name: "dry-run", Usage: "Preview the request body without updating the comment", Bool: true, Default: "false"},
		},
		Run: runIssueCommentUpdate,
	}
}

func newIssueCommentDeleteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "comment-delete",
		Description: "Delete an issue comment",
		Flags: []common.Flag{
			{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
			{Name: "comment-id", Short: "i", Usage: "Issue comment/journal ID", Required: true},
			{Name: "dry-run", Usage: "Preview the delete request without deleting the comment", Bool: true, Default: "false"},
		},
		Run: runIssueCommentDelete,
	}
}

func newIssueCommentChildrenShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "comment-children",
		Description: "List child comments for an issue comment",
		Flags: []common.Flag{
			{Name: "number", Short: "n", Usage: "Issue number (as shown in the web URL)", Required: true},
			{Name: "comment-id", Short: "i", Usage: "Parent issue comment/journal ID", Required: true},
			{Name: "keyword", Short: "k", Usage: "Search keyword"},
			{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
		},
		Run: runIssueCommentChildren,
	}
}

func runIssueComments(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	category, err := normalizeIssueJournalCategory(ctx.Arg("category"))
	if err != nil {
		return err
	}
	q := url.Values{}
	if category != "" {
		q.Set("category", category)
	}
	addStringQuery(q, "keyword", ctx.Arg("keyword"))
	addStringQuery(q, "sort_by", ctx.Arg("sort-by"))
	addStringQuery(q, "sort_direction", ctx.Arg("sort-direction"))
	q.Set("page", defaultValue(ctx.Arg("page"), "1"))
	q.Set("limit", defaultValue(ctx.Arg("limit"), "20"))

	env, err := ctx.CallAPIWithQuery("GET", issueJournalPath(ctx, number), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueComment(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	body, err := ctx.RequireArg("body")
	if err != nil {
		return err
	}
	payload, err := issueCommentPayload(ctx, body, true)
	if err != nil {
		return err
	}
	path := issueJournalPath(ctx, number)
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(issueJournalDryRun("create_comment", "POST", path, payload))
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return err
	}
	body, err := ctx.RequireArg("body")
	if err != nil {
		return err
	}
	payload, err := issueCommentPayload(ctx, body, false)
	if err != nil {
		return err
	}
	path := issueJournalItemPath(ctx, number, commentID)
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(issueJournalDryRun("update_comment", "PATCH", path, payload))
	}
	env, err := ctx.CallAPI("PATCH", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return err
	}
	path := issueJournalItemPath(ctx, number, commentID)
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(issueJournalDryRun("delete_comment", "DELETE", path, nil))
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runIssueCommentChildren(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	number, err := issueNumberArg(ctx)
	if err != nil {
		return err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return err
	}
	q := url.Values{}
	addStringQuery(q, "keyword", ctx.Arg("keyword"))
	q.Set("page", defaultValue(ctx.Arg("page"), "1"))
	q.Set("limit", defaultValue(ctx.Arg("limit"), "20"))
	env, err := ctx.CallAPIWithQuery("GET", issueJournalChildrenPath(ctx, number, commentID), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func issueCommentPayload(ctx *common.RuntimeContext, body string, includeReplyFields bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{"notes": body}
	if includeReplyFields {
		if parentID := ctx.Arg("parent-id"); parentID != "" {
			id, err := parsePositiveIntValue(parentID, "parent-id")
			if err != nil {
				return nil, err
			}
			payload["parent_id"] = id
		}
		if replyID := ctx.Arg("reply-id"); replyID != "" {
			id, err := parsePositiveIntValue(replyID, "reply-id")
			if err != nil {
				return nil, err
			}
			payload["reply_id"] = id
		}
	}
	if value := ctx.Arg("attachment-ids"); value != "" {
		ids, err := parseIntCSV(value, "attachment-ids")
		if err != nil {
			return nil, err
		}
		payload["attachment_ids"] = ids
	}
	if value := ctx.Arg("receivers"); value != "" {
		payload["receivers_login"] = parseStringCSV(value)
	}
	return payload, nil
}

func issueJournalPath(ctx *common.RuntimeContext, number string) string {
	return fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), number)
}

func issueJournalItemPath(ctx *common.RuntimeContext, number, commentID string) string {
	return fmt.Sprintf("%s/%s", issueJournalPath(ctx, number), commentID)
}

func issueJournalChildrenPath(ctx *common.RuntimeContext, number, commentID string) string {
	return fmt.Sprintf("%s/children_journals", issueJournalItemPath(ctx, number, commentID))
}

func normalizeIssueJournalCategory(value string) (string, error) {
	category := strings.ToLower(strings.TrimSpace(value))
	if category == "" {
		return "", nil
	}
	switch category {
	case "all", "comment", "operate":
		return category, nil
	default:
		return "", fmt.Errorf("invalid --category %q: use all, comment, or operate", value)
	}
}

func parsePositiveIntValue(value, flagName string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", flagName, value)
	}
	return id, nil
}

func parseIntCSV(value, flagName string) ([]int, error) {
	values := make([]int, 0)
	seen := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := parsePositiveIntValue(part, flagName)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		values = append(values, id)
	}
	return values, nil
}

func parseStringCSV(value string) []string {
	values := make([]string, 0)
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		values = append(values, part)
	}
	return values
}

func addStringQuery(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func issueJournalDryRun(action, method, path string, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"dry_run": true,
		"action":  action,
		"method":  method,
		"path":    path,
	}
	if payload != nil {
		data["body"] = payload
	}
	return data
}
