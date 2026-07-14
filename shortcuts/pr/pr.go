package pr

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
				{Name: "login", Usage: "Filter pull requests by issue author login"},
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
				if login := ctx.Arg("login"); login != "" {
					filterPullsByLogin(env, login)
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
			Name:        "review-comments",
			Description: "List inline review comments on a pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "review-id", Usage: "Review ID"},
				{Name: "need-respond", Usage: "Filter by whether comments still need a response: true or false"},
				{Name: "state", Short: "s", Usage: "Comment state: opened, resolved, or disabled"},
				{Name: "parent-id", Usage: "Parent comment ID"},
				{Name: "path", Short: "f", Usage: "Filter by file path"},
				{Name: "is-full", Usage: "Whether to include reply comments: true or false"},
				{Name: "sort-by", Usage: "Sort field: created_on or updated_on"},
				{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
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
				if reviewID := ctx.Arg("review-id"); reviewID != "" {
					q.Set("review_id", reviewID)
				}
				if needRespond := ctx.Arg("need-respond"); needRespond != "" {
					q.Set("need_respond", needRespond)
				}
				if state := ctx.Arg("state"); state != "" {
					normalizedState, err := normalizePRReviewCommentState(state)
					if err != nil {
						return err
					}
					q.Set("state", normalizedState)
				}
				if parentID := ctx.Arg("parent-id"); parentID != "" {
					q.Set("parent_id", parentID)
				}
				if path := ctx.Arg("path"); path != "" {
					q.Set("path", path)
				}
				if isFull := ctx.Arg("is-full"); isFull != "" {
					q.Set("is_full", isFull)
				}
				if sortBy := ctx.Arg("sort-by"); sortBy != "" {
					q.Set("sort_by", sortBy)
				}
				if sortDirection := ctx.Arg("sort-direction"); sortDirection != "" {
					q.Set("sort_direction", sortDirection)
				}
				env, err := ctx.CallAPIWithQuery("GET", prV1Path(ctx, id)+"/journals", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "review-comment",
			Description: "Create an inline review comment on a pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "review-id", Usage: "Review ID", Required: true},
				{Name: "path", Short: "f", Usage: "File path to comment on", Required: true},
				{Name: "line-code", Usage: "Line code returned by the diff API", Required: true},
				{Name: "note", Short: "n", Usage: "Comment body", Required: true},
				{Name: "type", Short: "t", Usage: "Comment type: comment or problem", Default: "comment"},
				{Name: "commit", Short: "m", Usage: "Commit SHA for the comment"},
				{Name: "parent-id", Usage: "Parent comment ID for replies"},
				{Name: "diff-file", Usage: "Load diff JSON from a file instead of fetching PR files automatically"},
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
				reviewID, err := ctx.RequireArg("review-id")
				if err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				lineCode, err := ctx.RequireArg("line-code")
				if err != nil {
					return err
				}
				note, err := ctx.RequireArg("note")
				if err != nil {
					return err
				}
				commentType, err := normalizePRReviewCommentType(ctx.Arg("type"))
				if err != nil {
					return err
				}
				diff, diffSource, err := loadPRReviewCommentDiff(ctx, id, path, ctx.Arg("diff-file"))
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"type":      commentType,
					"note":      note,
					"review_id": reviewID,
					"line_code": lineCode,
					"path":      path,
					"diff":      diff,
				}
				if commit := ctx.Arg("commit"); commit != "" {
					payload["commit_id"] = commit
				}
				if parentID := ctx.Arg("parent-id"); parentID != "" {
					payload["parent_id"] = parentID
				}
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"pull_request": id,
						"dry_run":      true,
						"action":       "create_review_comment",
						"diff_source":  diffSource,
						"payload":      payload,
					})
				}
				env, err := ctx.CallAPI("POST", prV1Path(ctx, id)+"/journals", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update-review-comment",
			Description: "Update an inline review comment on a pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "comment-id", Usage: "Review comment ID", Required: true},
				{Name: "note", Short: "n", Usage: "Updated comment body"},
				{Name: "state", Short: "s", Usage: "Comment state: opened, resolved, or disabled"},
				{Name: "commit", Short: "m", Usage: "Commit SHA to attach to the update"},
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
				commentID, err := ctx.RequireArg("comment-id")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{}
				if note := ctx.Arg("note"); note != "" {
					payload["note"] = note
				}
				if state := ctx.Arg("state"); state != "" {
					normalizedState, err := normalizePRReviewCommentState(state)
					if err != nil {
						return err
					}
					payload["state"] = normalizedState
				}
				if commit := ctx.Arg("commit"); commit != "" {
					payload["commit_id"] = commit
				}
				if len(payload) == 0 {
					return fmt.Errorf("at least one of --note, --state, or --commit is required")
				}
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"pull_request": id,
						"dry_run":      true,
						"action":       "update_review_comment",
						"comment_id":   commentID,
						"payload":      payload,
					})
				}
				env, err := ctx.CallAPI("PUT", prV1Path(ctx, id)+"/journals/"+commentID, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete-review-comment",
			Description: "Delete an inline review comment from a pull request",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pr.id"), Required: true},
				{Name: "comment-id", Usage: "Review comment ID", Required: true},
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
				commentID, err := ctx.RequireArg("comment-id")
				if err != nil {
					return err
				}
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository":   fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"pull_request": id,
						"dry_run":      true,
						"action":       "delete_review_comment",
						"comment_id":   commentID,
					})
				}
				env, err := ctx.CallAPI("DELETE", prV1Path(ctx, id)+"/journals/"+commentID, nil)
				if err != nil {
					return err
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

func validatePRReviewStatus(status string) error {
	switch status {
	case "common", "approved", "rejected":
		return nil
	default:
		return fmt.Errorf("invalid --status value %q: use common, approved, or rejected", status)
	}
}

func normalizePRReviewCommentType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "comment":
		return "comment", nil
	case "problem":
		return "problem", nil
	default:
		return "", fmt.Errorf("invalid --type value %q: use comment or problem", value)
	}
}

func normalizePRReviewCommentState(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "opened", "resolved", "disabled":
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("invalid --state value %q: use opened, resolved, or disabled", value)
	}
}

func loadPRReviewCommentDiff(ctx *common.RuntimeContext, prID string, path string, diffFile string) (map[string]interface{}, string, error) {
	if diffFile != "" {
		raw, err := os.ReadFile(diffFile)
		if err != nil {
			return nil, "", fmt.Errorf("read --diff-file %q: %w", diffFile, err)
		}
		var payload interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, "", fmt.Errorf("parse --diff-file %q: %w", diffFile, err)
		}
		diff, err := extractPRReviewCommentDiff(payload, path)
		if err != nil {
			return nil, "", err
		}
		return diff, diffFile, nil
	}

	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pulls/%s/files", ctx.RepoPath(), prID), nil)
	if err != nil {
		return nil, "", err
	}
	diff, err := extractPRReviewCommentDiff(env.Data, path)
	if err != nil {
		return nil, "", err
	}
	return diff, "pr_files_api", nil
}

func extractPRReviewCommentDiff(data interface{}, path string) (map[string]interface{}, error) {
	diff, ok := findPRReviewCommentDiff(data, path)
	if !ok {
		return nil, fmt.Errorf("no diff found for path %q; use --diff-file to provide the exact diff JSON", path)
	}
	return normalizePRReviewCommentDiff(diff, path), nil
}

func findPRReviewCommentDiff(data interface{}, path string) (map[string]interface{}, bool) {
	switch v := data.(type) {
	case *output.Envelope:
		return findPRReviewCommentDiff(v.Data, path)
	case map[string]interface{}:
		if nested, ok := v["data"]; ok {
			if diff, found := findPRReviewCommentDiff(nested, path); found {
				return diff, true
			}
		}
		if diff, ok := v["diff"].(map[string]interface{}); ok {
			if files, ok := diff["files"]; ok {
				if found, ok := findPRReviewCommentDiff(files, path); ok {
					return found, true
				}
			}
		}
		if files, ok := v["files"]; ok {
			if found, ok := findPRReviewCommentDiff(files, path); ok {
				return found, true
			}
		}
		if looksLikePRReviewCommentDiff(v) && matchesPRReviewCommentPath(v, path) {
			return v, true
		}
	case []interface{}:
		for _, item := range v {
			diff, ok := findPRReviewCommentDiff(item, path)
			if ok {
				return diff, true
			}
		}
	}
	return nil, false
}

func looksLikePRReviewCommentDiff(diff map[string]interface{}) bool {
	_, hasName := diff["name"]
	_, hasSections := diff["sections"]
	_, hasAddition := diff["addition"]
	return (hasName || hasAddition) && hasSections
}

func matchesPRReviewCommentPath(diff map[string]interface{}, path string) bool {
	for _, key := range []string{"name", "path", "filename", "fileName", "old_name", "oldname"} {
		if stringField(diff, key) == path {
			return true
		}
	}
	return false
}

func normalizePRReviewCommentDiff(diff map[string]interface{}, path string) map[string]interface{} {
	normalized := cloneMap(diff)
	normalized["name"] = firstStringField(diff, "name", "path", "filename", "fileName")
	normalized["oldname"] = firstStringField(diff, "oldname", "old_name")
	if normalized["oldname"] == "" {
		normalized["oldname"] = normalized["name"]
	}
	copyBoolAlias(normalized, diff, "is_created", "isCreated", "is_created")
	copyBoolAlias(normalized, diff, "is_deleted", "isDeleted", "is_deleted")
	copyBoolAlias(normalized, diff, "is_bin", "isBin", "is_bin")
	copyBoolAlias(normalized, diff, "is_lfs_file", "isLFSFile", "is_lfs_file")
	copyBoolAlias(normalized, diff, "is_renamed", "isRenamed", "is_renamed")
	copyBoolAlias(normalized, diff, "is_ambiguous", "isAmbiguous", "is_ambiguous")
	copyBoolAlias(normalized, diff, "is_submodule", "isSubmodule", "is_submodule")
	if path != "" {
		normalized["path"] = path
	}
	if sections, ok := diff["sections"].([]interface{}); ok {
		normalized["sections"] = normalizePRReviewCommentSections(sections, normalized["name"])
	}
	return normalized
}

func normalizePRReviewCommentSections(sections []interface{}, fallbackPath interface{}) []interface{} {
	normalized := make([]interface{}, 0, len(sections))
	for _, rawSection := range sections {
		section, ok := rawSection.(map[string]interface{})
		if !ok {
			continue
		}
		next := cloneMap(section)
		next["file_name"] = firstStringField(section, "file_name", "fileName")
		if next["file_name"] == "" {
			next["file_name"] = fallbackPath
		}
		if lines, ok := section["lines"].([]interface{}); ok {
			next["lines"] = normalizePRReviewCommentLines(lines, next["file_name"])
		}
		normalized = append(normalized, next)
	}
	return normalized
}

func normalizePRReviewCommentLines(lines []interface{}, fallbackPath interface{}) []interface{} {
	normalized := make([]interface{}, 0, len(lines))
	for _, rawLine := range lines {
		line, ok := rawLine.(map[string]interface{})
		if !ok {
			continue
		}
		next := cloneMap(line)
		if left, ok := firstNumberField(line, "left_index", "leftIdx"); ok {
			next["left_index"] = left
		}
		if right, ok := firstNumberField(line, "right_index", "rightIdx"); ok {
			next["right_index"] = right
		}
		if _, ok := next["match"]; !ok {
			next["match"] = inferPRReviewCommentLineMatch(line)
		}
		if sectionInfo, ok := line["sectionInfo"].(map[string]interface{}); ok {
			next["section_path"] = firstStringField(sectionInfo, "section_path", "path")
			if next["section_path"] == "" {
				next["section_path"] = fallbackPath
			}
			if v, ok := firstNumberField(sectionInfo, "section_last_left_index", "lastLeftIdx"); ok {
				next["section_last_left_index"] = v
			}
			if v, ok := firstNumberField(sectionInfo, "section_last_right_index", "lastRightIdx"); ok {
				next["section_last_right_index"] = v
			}
			if v, ok := firstNumberField(sectionInfo, "section_left_index", "leftIdx"); ok {
				next["section_left_index"] = v
			}
			if v, ok := firstNumberField(sectionInfo, "section_right_index", "rightIdx"); ok {
				next["section_right_index"] = v
			}
			if v, ok := firstNumberField(sectionInfo, "section_left_hunk_size", "leftHunkSize"); ok {
				next["section_left_hunk_size"] = v
			}
			if v, ok := firstNumberField(sectionInfo, "section_right_hunk_size", "rightHunkSize"); ok {
				next["section_right_hunk_size"] = v
			}
		}
		normalized = append(normalized, next)
	}
	return normalized
}

func inferPRReviewCommentLineMatch(line map[string]interface{}) float64 {
	if match, ok := numberField(line, "match"); ok {
		return match
	}
	lineType, _ := numberField(line, "type")
	switch int(lineType) {
	case 2:
		return 1
	case 3:
		return 3
	default:
		return 0
	}
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyBoolAlias(dst map[string]interface{}, src map[string]interface{}, dstKey string, candidates ...string) {
	for _, key := range candidates {
		if v, ok := src[key].(bool); ok {
			dst[dstKey] = v
			return
		}
	}
}

func firstStringField(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v := stringField(m, key); v != "" {
			return v
		}
	}
	return ""
}

func firstNumberField(m map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		if v, ok := numberField(m, key); ok {
			return v, true
		}
	}
	return 0, false
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

// filterPullsByLogin filters the pulls list in-place, keeping only items
// whose issue.author.login matches the given login (case-insensitive).
func filterPullsByLogin(env *output.Envelope, login string) {
	if env == nil || env.Data == nil {
		return
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return
	}
	rawPulls, ok := data["pulls"]
	if !ok {
		return
	}
	pulls, ok := rawPulls.([]interface{})
	if !ok {
		return
	}
	loginLower := strings.ToLower(login)
	filtered := make([]interface{}, 0, len(pulls))
	for _, item := range pulls {
		pull, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		issue, ok := pull["issue"].(map[string]interface{})
		if !ok {
			continue
		}
		author, ok := issue["author"].(map[string]interface{})
		if !ok {
			continue
		}
		authorLogin, _ := author["login"].(string)
		if strings.ToLower(authorLogin) == loginLower {
			filtered = append(filtered, item)
		}
	}
	data["pulls"] = filtered
	// Update total_count in data to reflect filtered count
	data["total_count"] = float64(len(filtered))
	if env.Meta != nil {
		env.Meta.TotalCount = len(filtered)
	}
}
