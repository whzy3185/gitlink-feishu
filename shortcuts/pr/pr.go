package pr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func v1RepoPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}

func normalizePullRequestListState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "open", "opened":
		return "0"
	case "merged":
		return "1"
	case "closed":
		return "2"
	case "all", "":
		return ""
	default:
		return state
	}
}

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.pr.list.short"),
			Flags: []common.Flag{
				{Name: "state", Short: "s", Usage: tr.T("flag.pr.state"), Default: "open"},
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword")},
				{Name: "priority-id", Usage: tr.T("flag.pr.priority_id")},
				{Name: "tag-id", Usage: tr.T("flag.pr.tag_id")},
				{Name: "milestone-id", Usage: tr.T("flag.pr.milestone_id")},
				{Name: "reviewer-id", Usage: tr.T("flag.pr.reviewer_id")},
				{Name: "assignee-id", Usage: tr.T("flag.pr.assignee_id")},
				{Name: "sort-by", Usage: tr.T("flag.sort_by")},
				{Name: "sort-direction", Usage: tr.T("flag.sort_direction")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if s := normalizePullRequestListState(ctx.Arg("state")); s != "" {
					q.Set("status", s)
				}
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				if priorityID := ctx.Arg("priority-id"); priorityID != "" {
					q.Set("priority_id", priorityID)
				}
				if tagID := ctx.Arg("tag-id"); tagID != "" {
					q.Set("issue_tag_id", tagID)
				}
				if milestoneID := ctx.Arg("milestone-id"); milestoneID != "" {
					q.Set("version_id", milestoneID)
				}
				if reviewerID := ctx.Arg("reviewer-id"); reviewerID != "" {
					q.Set("reviewer_id", reviewerID)
				}
				if assigneeID := ctx.Arg("assignee-id"); assigneeID != "" {
					q.Set("assign_user_id", assigneeID)
				}
				if sortBy := ctx.Arg("sort-by"); sortBy != "" {
					q.Set("sort_by", sortBy)
				}
				if sortDirection := ctx.Arg("sort-direction"); sortDirection != "" {
					q.Set("sort_direction", sortDirection)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/pulls", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.pr.create.short"),
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: tr.T("flag.pr.title"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.pr.body")},
				{Name: "head", Usage: tr.T("flag.pr.head"), Required: true},
				{Name: "base", Usage: tr.T("flag.pr.base"), Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title, _ := ctx.RequireArg("title")
				head, _ := ctx.RequireArg("head")
				base := ctx.Arg("base")
				if base == "" {
					base = "master"
				}
				payload := map[string]interface{}{
					"title": title,
					"head":  head,
					"base":  base,
				}
				if b := ctx.Arg("body"); b != "" {
					payload["body"] = b
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/pulls", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.pr.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				if err := enrichPullRequestClosedAt(ctx, env); err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "merge",
			Description: tr.T("cmd.pr.merge.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "method", Short: "m", Usage: tr.T("flag.pr.merge_method"), Default: "merge"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				method := ctx.Arg("method")
				if method == "" {
					method = "merge"
				}
				payload := map[string]interface{}{
					"do": method,
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/pulls/%s/pr_merge", ctx.RepoPath(), id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "refuse",
			Description: "Refuse and close a pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/pulls/%s/refuse_merge", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "reopen",
			Description: "Reopen a closed pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "PR number", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", prV1Path(ctx, id)+"/reopen", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: tr.T("cmd.pr.files.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s/files", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "diff",
			Description: tr.T("cmd.pr.diff.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s/files", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "versions",
			Description: tr.T("cmd.pr.versions.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", prV1Path(ctx, id)+"/versions", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "version-diff",
			Description: tr.T("cmd.pr.version_diff.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "version-id", Short: "v", Usage: tr.T("flag.pr.version_id"), Required: true},
				{Name: "file", Short: "f", Usage: tr.T("flag.pr.file")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				versionID, err := ctx.RequireArg("version-id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/versions/%s/diff", prV1Path(ctx, id), versionID)
				if file := ctx.Arg("file"); file != "" {
					q := url.Values{}
					q.Set("filepath", file)
					env, err := ctx.CallAPIWithQuery("GET", path, q)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				}
				env, err := ctx.CallAPI("GET", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "reviews",
			Description: tr.T("cmd.pr.reviews.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "status", Short: "s", Usage: tr.T("flag.pr.review_status_filter")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				q := url.Values{}
				if status := ctx.Arg("status"); status != "" {
					if err := validatePRReviewStatus(status); err != nil {
						return err
					}
					q.Set("status", status)
				}
				env, err := ctx.CallAPIWithQuery("GET", prV1Path(ctx, id)+"/reviews", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "review",
			Description: tr.T("cmd.pr.review.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "status", Short: "s", Usage: tr.T("flag.pr.review_status"), Default: "common"},
				{Name: "content", Short: "c", Usage: tr.T("flag.pr.review_content"), Required: true},
				{Name: "commit", Short: "m", Usage: tr.T("flag.pr.review_commit")},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				status := ctx.Arg("status")
				if status == "" {
					status = "common"
				}
				if err := validatePRReviewStatus(status); err != nil {
					return err
				}
				payload := map[string]interface{}{
					"content": content,
					"status":  status,
				}
				if commit := ctx.Arg("commit"); commit != "" {
					payload["commit_id"] = commit
				}
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"pull_request": id,
						"dry_run":      true,
						"action":       "create_review",
						"payload":      payload,
					})
				}
				env, err := ctx.CallAPI("POST", prV1Path(ctx, id)+"/reviews", payload)
				if err != nil {
					return err
				}

				// Also post a journal comment so the review is visible in the PR conversation.
				prEnv, journalErr := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s", ctx.RepoPath(), id), nil)
				if journalErr == nil {
					if issueID, extractErr := extractIssueID(prEnv); extractErr == nil {
						statusLabel := map[string]string{
							"approved": "approved", "rejected": "rejected", "common": "commented",
						}[status]
						summary := fmt.Sprintf("## Review: %s\n\n%s", statusLabel, content)
						ctx.CallAPI("POST", fmt.Sprintf("/v1/%s/%s/issues/%d/journals", ctx.Owner, ctx.Repo, issueID),
							map[string]interface{}{"notes": summary})
					}
				}

				return ctx.Output(env)
			},
		},
		{
			Name:        "comment",
			Description: tr.T("cmd.pr.comment.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				body, _ := ctx.RequireArg("body")

				prEnv, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return fmt.Errorf("fetch PR: %w", err)
				}
				issueID, err := extractIssueID(prEnv)
				if err != nil {
					return err
				}

				payload := map[string]interface{}{
					"notes": body,
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("/v1/%s/%s/issues/%d/journals", ctx.Owner, ctx.Repo, issueID), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "review-comments",
			Description: "List pull request review comments and discussion threads",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "keyword", Short: "k", Usage: "Search review comment content"},
				{Name: "review-id", Usage: "Filter by review ID"},
				{Name: "need-respond", Usage: "Filter comments that need response: true or false"},
				{Name: "state", Short: "s", Usage: "Filter by opened, resolved, or disabled"},
				{Name: "parent-id", Usage: "Filter replies under a parent comment ID"},
				{Name: "path", Usage: "Filter by file path"},
				{Name: "full", Usage: "Include replies in the response", Bool: true, Default: "false"},
				{Name: "sort-by", Usage: "Sort field: created_on or updated_on"},
				{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
			},
			Run: runPRReviewComments,
		},
		{
			Name:        "review-comment",
			Description: "Create a pull request review comment or reply",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.comment.body"), Required: true},
				{Name: "type", Usage: "Comment type: comment or problem", Default: "comment"},
				{Name: "review-id", Usage: "Review ID"},
				{Name: "line-code", Usage: "GitLink diff line code"},
				{Name: "commit", Usage: "Commit SHA for the commented diff"},
				{Name: "path", Usage: "Commented file path"},
				{Name: "parent-id", Usage: "Parent review comment ID for a reply"},
				{Name: "diff-json", Usage: "Raw diff JSON object for line comments"},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runPRReviewCommentCreate,
		},
		{
			Name:        "review-comment-update",
			Description: "Update a pull request review comment note, commit, or state",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "comment-id", Usage: "Review comment ID", Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.comment.body")},
				{Name: "commit", Usage: "Commit SHA"},
				{Name: "state", Short: "s", Usage: "New state: opened, resolved, or disabled"},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runPRReviewCommentUpdate,
		},
		{
			Name:        "review-comment-delete",
			Description: "Delete a pull request review comment",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "comment-id", Usage: "Review comment ID", Required: true},
			},
			Run: runPRReviewCommentDelete,
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func prV1Path(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("/v1/%s/%s/pulls/%s", ctx.Owner, ctx.Repo, id)
}

func prReviewCommentPath(ctx *common.RuntimeContext, prID string) string {
	return fmt.Sprintf("%s/journals", prV1Path(ctx, url.PathEscape(prID)))
}

func prReviewCommentItemPath(ctx *common.RuntimeContext, prID, commentID string) string {
	return fmt.Sprintf("%s/%s", prReviewCommentPath(ctx, prID), url.PathEscape(commentID))
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
	setPRQueryIfPresent(q, "keyword", ctx.Arg("keyword"))
	if reviewID := ctx.Arg("review-id"); reviewID != "" {
		if _, err := parsePRPositiveID(reviewID, "review-id"); err != nil {
			return err
		}
		q.Set("review_id", strings.TrimSpace(reviewID))
	}
	if needRespond := ctx.Arg("need-respond"); needRespond != "" {
		if err := validatePRBoolString("need-respond", needRespond); err != nil {
			return err
		}
		q.Set("need_respond", strings.ToLower(strings.TrimSpace(needRespond)))
	}
	if state := ctx.Arg("state"); state != "" {
		if err := validatePRReviewCommentState(state); err != nil {
			return err
		}
		q.Set("state", strings.TrimSpace(state))
	}
	if parentID := ctx.Arg("parent-id"); parentID != "" {
		if _, err := parsePRPositiveID(parentID, "parent-id"); err != nil {
			return err
		}
		q.Set("parent_id", strings.TrimSpace(parentID))
	}
	setPRQueryIfPresent(q, "path", ctx.Arg("path"))
	if ctx.Arg("full") == "true" {
		q.Set("is_full", "true")
	}
	setPRQueryIfPresent(q, "sort_by", ctx.Arg("sort-by"))
	setPRQueryIfPresent(q, "sort_direction", ctx.Arg("sort-direction"))
	env, err := ctx.CallAPIWithQuery("GET", prReviewCommentPath(ctx, id), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewCommentCreate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	body, err := ctx.RequireArg("body")
	if err != nil {
		return err
	}
	payload, err := prReviewCommentCreatePayload(ctx, body)
	if err != nil {
		return err
	}
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"pull_request": id,
			"dry_run":      true,
			"action":       "create_review_comment",
			"payload":      payload,
		})
	}
	env, err := ctx.CallAPI("POST", prReviewCommentPath(ctx, id), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewCommentUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	prID, commentID, err := prReviewCommentTarget(ctx)
	if err != nil {
		return err
	}
	payload, err := prReviewCommentUpdatePayload(ctx)
	if err != nil {
		return err
	}
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"repository":     fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"pull_request":   prID,
			"review_comment": commentID,
			"dry_run":        true,
			"action":         "update_review_comment",
			"payload":        payload,
		})
	}
	env, err := ctx.CallAPI("PUT", prReviewCommentItemPath(ctx, prID, commentID), payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runPRReviewCommentDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	prID, commentID, err := prReviewCommentTarget(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("DELETE", prReviewCommentItemPath(ctx, prID, commentID), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func prReviewCommentCreatePayload(ctx *common.RuntimeContext, body string) (map[string]interface{}, error) {
	commentType := firstPRNonEmpty(ctx.Arg("type"), "comment")
	if err := validatePRReviewCommentType(commentType); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"type": commentType,
		"note": body,
	}
	for _, field := range []struct {
		flag string
		key  string
	}{
		{"review-id", "review_id"},
		{"parent-id", "parent_id"},
	} {
		if value := ctx.Arg(field.flag); value != "" {
			id, err := parsePRPositiveID(value, field.flag)
			if err != nil {
				return nil, err
			}
			payload[field.key] = id
		}
	}
	setPayloadStringIfPresent(payload, "line_code", ctx.Arg("line-code"))
	setPayloadStringIfPresent(payload, "commit_id", ctx.Arg("commit"))
	setPayloadStringIfPresent(payload, "path", ctx.Arg("path"))
	if rawDiff := strings.TrimSpace(ctx.Arg("diff-json")); rawDiff != "" {
		var diff map[string]interface{}
		if err := json.Unmarshal([]byte(rawDiff), &diff); err != nil {
			return nil, fmt.Errorf("invalid --diff-json: %w", err)
		}
		payload["diff"] = diff
	}
	return payload, nil
}

func prReviewCommentUpdatePayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	payload := map[string]interface{}{}
	setPayloadStringIfPresent(payload, "note", ctx.Arg("body"))
	setPayloadStringIfPresent(payload, "commit_id", ctx.Arg("commit"))
	if state := ctx.Arg("state"); state != "" {
		if err := validatePRReviewCommentState(state); err != nil {
			return nil, err
		}
		payload["state"] = strings.TrimSpace(state)
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("at least one of --body, --commit, or --state is required")
	}
	return payload, nil
}

func prReviewCommentTarget(ctx *common.RuntimeContext) (string, string, error) {
	prID, err := ctx.RequireArg("id")
	if err != nil {
		return "", "", err
	}
	commentID, err := ctx.RequireArg("comment-id")
	if err != nil {
		return "", "", err
	}
	if _, err := parsePRPositiveID(commentID, "comment-id"); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(prID), strings.TrimSpace(commentID), nil
}

func validatePRReviewCommentType(value string) error {
	switch strings.TrimSpace(value) {
	case "comment", "problem":
		return nil
	default:
		return fmt.Errorf("invalid --type value %q: use comment or problem", value)
	}
}

func validatePRReviewCommentState(value string) error {
	switch strings.TrimSpace(value) {
	case "opened", "resolved", "disabled":
		return nil
	default:
		return fmt.Errorf("invalid --state value %q: use opened, resolved, or disabled", value)
	}
}

func validatePRBoolString(flagName, value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "false":
		return nil
	default:
		return fmt.Errorf("invalid --%s value %q: use true or false", flagName, value)
	}
}

func parsePRPositiveID(value, flagName string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("--%s contains an empty ID", flagName)
	}
	id, err := strconv.Atoi(trimmed)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("--%s must be a positive numeric ID", flagName)
	}
	return id, nil
}

func setPRQueryIfPresent(q url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		q.Set(key, strings.TrimSpace(value))
	}
}

func setPayloadStringIfPresent(payload map[string]interface{}, key, value string) {
	if strings.TrimSpace(value) != "" {
		payload[key] = strings.TrimSpace(value)
	}
}

func firstPRNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func validatePRReviewStatus(status string) error {
	switch status {
	case "common", "approved", "rejected":
		return nil
	default:
		return fmt.Errorf("invalid --status value %q: use common, approved, or rejected", status)
	}
}

func extractIssueID(env *output.Envelope) (int64, error) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected PR response format")
	}
	issue, ok := data["issue"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("PR response missing issue field")
	}
	idFloat, ok := issue["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("PR response missing issue.id field")
	}
	return int64(idFloat), nil
}

func enrichPullRequestClosedAt(ctx *common.RuntimeContext, env *output.Envelope) error {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil
	}
	pr, ok := data["pull_request"].(map[string]interface{})
	if !ok || !isClosedPullRequest(pr) || stringField(pr, "closed_at") != "" {
		return nil
	}
	issue, ok := data["issue"].(map[string]interface{})
	if !ok {
		return nil
	}
	issueID, ok := numberField(issue, "id")
	if !ok {
		return nil
	}
	journalsEnv, err := ctx.CallAPI("GET", fmt.Sprintf("/v1/%s/%s/issues/%d/journals", ctx.Owner, ctx.Repo, int64(issueID)), nil)
	if err != nil {
		return err
	}
	closedAt := extractPullRequestClosedAt(journalsEnv)
	if closedAt == "" {
		return nil
	}
	pr["closed_at"] = closedAt
	data["closed_at"] = closedAt
	return nil
}

func isClosedPullRequest(pr map[string]interface{}) bool {
	if stringField(pr, "pull_request_staus") == "closed" || stringField(pr, "state") == "closed" {
		return true
	}
	status, ok := numberField(pr, "status")
	return ok && int(status) == 2
}

func extractPullRequestClosedAt(env *output.Envelope) string {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return ""
	}
	rawJournals, ok := data["journals"].([]interface{})
	if !ok {
		return ""
	}
	for i := len(rawJournals) - 1; i >= 0; i-- {
		journal, ok := rawJournals[i].(map[string]interface{})
		if !ok || stringField(journal, "operate_category") != "status" {
			continue
		}
		content := stringField(journal, "operate_content")
		if !isPullRequestCloseOperation(content) {
			continue
		}
		if updatedAt := stringField(journal, "updated_at"); updatedAt != "" {
			return updatedAt
		}
		if createdAt := stringField(journal, "created_at"); createdAt != "" {
			return createdAt
		}
	}
	return ""
}

func isPullRequestCloseOperation(content string) bool {
	content = strings.ToLower(content)
	return strings.Contains(content, "合并请求") &&
		(strings.Contains(content, "拒绝") || strings.Contains(content, "关闭") || strings.Contains(content, "closed"))
}

func stringField(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

func numberField(m map[string]interface{}, key string) (float64, bool) {
	switch v := m[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
