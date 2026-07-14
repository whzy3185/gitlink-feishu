package pr

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var pullRequestJournalHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)

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
				if err := enrichPullRequestTimestamps(ctx, env); err != nil {
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

func enrichPullRequestTimestamps(ctx *common.RuntimeContext, env *output.Envelope) error {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil
	}
	pr, ok := data["pull_request"].(map[string]interface{})
	if !ok {
		return nil
	}
	issue, ok := data["issue"].(map[string]interface{})
	if createdAt := firstNonEmptyString(
		stringField(data, "created_at"),
		stringField(pr, "created_at"),
		stringField(issue, "created_at"),
	); createdAt != "" {
		data["created_at"] = createdAt
	}

	mergedAt := firstNonEmptyString(
		stringField(data, "merged_at"),
		stringField(pr, "merged_at"),
		stringField(issue, "merged_at"),
	)
	closedAt := firstNonEmptyString(
		stringField(data, "closed_at"),
		stringField(data, "closed_on"),
		stringField(pr, "closed_at"),
		stringField(issue, "closed_on"),
	)

	if shouldFetchPullRequestTimestamps(pr, mergedAt, closedAt) {
		issueID, ok := numberField(issue, "id")
		if ok {
			journalsEnv, err := ctx.CallAPI("GET", fmt.Sprintf("/v1/%s/%s/issues/%d/journals", ctx.Owner, ctx.Repo, int64(issueID)), nil)
			if err != nil {
				return err
			}
			journalMergedAt, journalClosedAt := extractPullRequestJournalTimes(journalsEnv)
			mergedAt = firstNonEmptyString(mergedAt, journalMergedAt)
			closedAt = firstNonEmptyString(closedAt, journalClosedAt)
		}
	}

	if closedAt == "" && isMergedPullRequest(pr) {
		closedAt = mergedAt
	}

	if mergedAt != "" {
		pr["merged_at"] = mergedAt
		data["merged_at"] = mergedAt
	}
	if closedAt != "" {
		pr["closed_at"] = closedAt
		data["closed_at"] = closedAt
		data["closed_on"] = closedAt
		if issue != nil {
			issue["closed_on"] = closedAt
		}
	}
	return nil
}

func isClosedPullRequest(pr map[string]interface{}) bool {
	if isMergedPullRequest(pr) {
		return true
	}
	if stringField(pr, "pull_request_staus") == "closed" || stringField(pr, "state") == "closed" {
		return true
	}
	status, ok := numberField(pr, "pull_request_status")
	if ok {
		return int(status) == 2
	}
	status, ok = numberField(pr, "status")
	return ok && int(status) == 2
}

func isMergedPullRequest(pr map[string]interface{}) bool {
	if merged, ok := pr["merged"].(bool); ok && merged {
		return true
	}
	if stringField(pr, "pull_request_staus") == "merged" || stringField(pr, "state") == "merged" {
		return true
	}
	status, ok := numberField(pr, "pull_request_status")
	if ok {
		return int(status) == 1
	}
	status, ok = numberField(pr, "status")
	return ok && int(status) == 1
}

func shouldFetchPullRequestTimestamps(pr map[string]interface{}, mergedAt, closedAt string) bool {
	if !isClosedPullRequest(pr) {
		return false
	}
	if isMergedPullRequest(pr) {
		return mergedAt == "" || closedAt == ""
	}
	return closedAt == ""
}

func extractPullRequestJournalTimes(env *output.Envelope) (string, string) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", ""
	}
	rawJournals, ok := data["journals"].([]interface{})
	if !ok {
		return "", ""
	}
	var mergedAt string
	var closedAt string
	for i := len(rawJournals) - 1; i >= 0; i-- {
		journal, ok := rawJournals[i].(map[string]interface{})
		if !ok || stringField(journal, "operate_category") != "status" {
			continue
		}
		content := stringField(journal, "operate_content")
		timestamp := firstNonEmptyString(
			stringField(journal, "updated_at"),
			stringField(journal, "created_at"),
		)
		if timestamp == "" {
			continue
		}
		if mergedAt == "" && isPullRequestMergeOperation(content) {
			mergedAt = timestamp
		}
		if closedAt == "" && isPullRequestCloseOperation(content) {
			closedAt = timestamp
		}
		if mergedAt != "" && closedAt != "" {
			break
		}
	}
	return mergedAt, closedAt
}

func isPullRequestMergeOperation(content string) bool {
	content = normalizePullRequestJournalContent(content)
	return containsPullRequestMarker(content) &&
		(strings.Contains(content, "\u5408\u5e76\u4e86") ||
			strings.Contains(content, "\u5df2\u5408\u5e76") ||
			strings.Contains(content, "merged"))
}

func isPullRequestCloseOperation(content string) bool {
	content = normalizePullRequestJournalContent(content)
	return containsPullRequestMarker(content) &&
		(strings.Contains(content, "\u62d2\u7edd") ||
			strings.Contains(content, "rejected") ||
			strings.Contains(content, "refused") ||
			strings.Contains(content, "\u5173\u95ed") ||
			strings.Contains(content, "closed"))
}

func normalizePullRequestJournalContent(content string) string {
	content = html.UnescapeString(content)
	content = pullRequestJournalHTMLTagPattern.ReplaceAllString(content, "")
	content = strings.ToLower(strings.TrimSpace(content))
	return strings.Join(strings.Fields(content), "")
}

func containsPullRequestMarker(content string) bool {
	return strings.Contains(content, "\u5408\u5e76\u8bf7\u6c42") || strings.Contains(content, "pullrequest")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringField(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
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
