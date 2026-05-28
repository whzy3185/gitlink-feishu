package pr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var allowedPRReviewCommentTypes = map[string]bool{"comment": true, "problem": true}
var allowedPRReviewCommentStates = map[string]bool{"opened": true, "resolved": true, "disabled": true}

func newPRReviewCommentsShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-comments",
		Description: "List pull request review comments",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "PR number", Required: true},
			{Name: "keyword", Short: "k", Usage: "Search keyword"},
			{Name: "review-id", Usage: "Filter by review ID"},
			{Name: "need-respond", Usage: "Filter by response requirement: true or false"},
			{Name: "state", Short: "s", Usage: "Filter state: opened, resolved, or disabled"},
			{Name: "parent-id", Usage: "Filter by parent comment ID"},
			{Name: "path", Usage: "Filter by file path"},
			{Name: "is-full", Usage: "Whether to include replies: true or false"},
			{Name: "sort-by", Usage: "Sort field: created_on or updated_on"},
			{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
		},
		Run: runPRReviewComments,
	}
}

func newPRReviewCommentShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-comment",
		Description: "Create a pull request review line comment",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "PR number", Required: true},
			{Name: "note", Short: "n", Usage: "Comment text", Required: true},
			{Name: "review-id", Usage: "Review ID", Required: true},
			{Name: "type", Short: "t", Usage: "Comment type: comment or problem", Default: "comment"},
			{Name: "commit", Short: "m", Usage: "Commit SHA for the review comment", Required: true},
			{Name: "line-code", Usage: "GitLink line_code value for the diff line", Required: true},
			{Name: "path", Short: "p", Usage: "File path for the review comment", Required: true},
			{Name: "parent-id", Usage: "Parent review comment ID when replying"},
			{Name: "diff-json", Usage: "Raw JSON diff object to include in the request body"},
			{Name: "diff-file", Usage: "Read JSON diff object from a file"},
			{Name: "dry-run", Usage: "Preview the request body without creating the review comment", Bool: true, Default: "false"},
		},
		Run: runPRReviewComment,
	}
}

func newPRReviewCommentUpdateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-comment-update",
		Description: "Update a pull request review comment",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "PR number", Required: true},
			{Name: "comment-id", Short: "j", Usage: "Review comment/journal ID", Required: true},
			{Name: "note", Short: "n", Usage: "New comment text"},
			{Name: "commit", Short: "m", Usage: "Commit SHA"},
			{Name: "state", Short: "s", Usage: "Comment state: opened, resolved, or disabled"},
			{Name: "dry-run", Usage: "Preview the request body without updating the review comment", Bool: true, Default: "false"},
		},
		Run: runPRReviewCommentUpdate,
	}
}

func newPRReviewCommentDeleteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-comment-delete",
		Description: "Delete a pull request review comment",
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "PR number", Required: true},
			{Name: "comment-id", Short: "j", Usage: "Review comment/journal ID", Required: true},
			{Name: "dry-run", Usage: "Preview the delete request without deleting the review comment", Bool: true, Default: "false"},
		},
		Run: runPRReviewCommentDelete,
	}
}

func runPRReviewComments(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	q := url.Values{}
	addPRQuery(q, "keyword", ctx.Arg("keyword"))
	addPRQuery(q, "review_id", ctx.Arg("review-id"))
	if value := ctx.Arg("need-respond"); value != "" {
		if _, err := strconv.ParseBool(strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("invalid --need-respond %q: use true or false", value)
		}
		q.Set("need_respond", value)
	}
	state, err := normalizePRReviewCommentState(ctx.Arg("state"), true)
	if err != nil {
		return err
	}
	addPRQuery(q, "state", state)
	addPRQuery(q, "parent_id", ctx.Arg("parent-id"))
	addPRQuery(q, "path", ctx.Arg("path"))
	if value := ctx.Arg("is-full"); value != "" {
		if _, err := strconv.ParseBool(strings.TrimSpace(value)); err != nil {
			return fmt.Errorf("invalid --is-full %q: use true or false", value)
		}
		q.Set("is_full", value)
	}
	addPRQuery(q, "sort_by", ctx.Arg("sort-by"))
	addPRQuery(q, "sort_direction", ctx.Arg("sort-direction"))

	env, err := ctx.CallAPIWithQuery("GET", prReviewJournalsPath(ctx, id), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewComment(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	payload, err := prReviewCommentCreatePayload(ctx)
	if err != nil {
		return err
	}
	path := prReviewJournalsPath(ctx, id)
	if parsePRBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(prJournalDryRun(ctx, id, "create_review_comment", "POST", path, payload))
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewCommentUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return err
	}
	payload, err := prReviewCommentUpdatePayload(ctx)
	if err != nil {
		return err
	}
	path := prReviewJournalItemPath(ctx, id, commentID)
	if parsePRBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(prJournalDryRun(ctx, id, "update_review_comment", "PUT", path, payload))
	}
	env, err := ctx.CallAPI("PUT", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewCommentDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return err
	}
	path := prReviewJournalItemPath(ctx, id, commentID)
	if parsePRBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(prJournalDryRun(ctx, id, "delete_review_comment", "DELETE", path, nil))
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func prReviewCommentCreatePayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	note, err := ctx.RequireArg("note")
	if err != nil {
		return nil, err
	}
	reviewID, err := ctx.RequireArg("review-id")
	if err != nil {
		return nil, err
	}
	commitID, err := ctx.RequireArg("commit")
	if err != nil {
		return nil, err
	}
	lineCode, err := ctx.RequireArg("line-code")
	if err != nil {
		return nil, err
	}
	filePath, err := ctx.RequireArg("path")
	if err != nil {
		return nil, err
	}
	commentType, err := normalizePRReviewCommentType(defaultPRValue(ctx.Arg("type"), "comment"))
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"type":      commentType,
		"note":      note,
		"review_id": reviewID,
		"line_code": lineCode,
		"commit_id": commitID,
		"path":      filePath,
	}
	if parentID := ctx.Arg("parent-id"); parentID != "" {
		id, err := parsePRPositiveInt(parentID, "parent-id")
		if err != nil {
			return nil, err
		}
		payload["parent_id"] = id
	}
	diff, err := parseJSONFromArgs(ctx.Arg("diff-json"), ctx.Arg("diff-file"))
	if err != nil {
		return nil, err
	}
	if diff != nil {
		payload["diff"] = diff
	}
	return payload, nil
}

func prReviewCommentUpdatePayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	payload := map[string]interface{}{}
	if note := ctx.Arg("note"); note != "" {
		payload["note"] = note
	}
	if commitID := ctx.Arg("commit"); commitID != "" {
		payload["commit_id"] = commitID
	}
	if state := ctx.Arg("state"); state != "" {
		normalized, err := normalizePRReviewCommentState(state, false)
		if err != nil {
			return nil, err
		}
		payload["state"] = normalized
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("at least one of --note, --commit, or --state is required")
	}
	return payload, nil
}

func prReviewJournalsPath(ctx *common.RuntimeContext, id string) string {
	return prV1Path(ctx, id) + "/journals"
}

func prReviewJournalItemPath(ctx *common.RuntimeContext, id, commentID string) string {
	return fmt.Sprintf("%s/%s", prReviewJournalsPath(ctx, id), commentID)
}

func normalizePRReviewCommentType(value string) (string, error) {
	commentType := strings.ToLower(strings.TrimSpace(value))
	if allowedPRReviewCommentTypes[commentType] {
		return commentType, nil
	}
	return "", fmt.Errorf("invalid --type %q: use comment or problem", value)
}

func normalizePRReviewCommentState(value string, allowEmpty bool) (string, error) {
	state := strings.ToLower(strings.TrimSpace(value))
	if state == "" && allowEmpty {
		return "", nil
	}
	if allowedPRReviewCommentStates[state] {
		return state, nil
	}
	return "", fmt.Errorf("invalid --state %q: use opened, resolved, or disabled", value)
}

func parseJSONFromArgs(raw, file string) (interface{}, error) {
	if raw != "" && file != "" {
		return nil, fmt.Errorf("use either --diff-json or --diff-file, not both")
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read --diff-file: %w", err)
		}
		raw = string(data)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse diff JSON: %w", err)
	}
	return parsed, nil
}

func parsePRPositiveInt(value, flagName string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", flagName, value)
	}
	return id, nil
}

func addPRQuery(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func defaultPRValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parsePRBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func prJournalDryRun(ctx *common.RuntimeContext, id, action, method, path string, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"pull_request": id,
		"dry_run":      true,
		"action":       action,
		"method":       method,
		"path":         path,
	}
	if payload != nil {
		data["body"] = payload
	}
	return data
}
