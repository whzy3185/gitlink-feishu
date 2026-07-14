package pr

import (
	"fmt"
	"net/url"
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
				payload := map[string]interface{}{
					"title": title,
					"head":  head,
					"base":  ctx.Arg("base"),
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
				{Name: "id", Short: "i", Usage: "PR number", Required: true},
				{Name: "file", Short: "f", Usage: "Filter diff to a specific file path"},
				{Name: "stat", Usage: "Show only diff stat summary", Bool: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")

				// v0 API：返回变更文件列表及 patch 内容。
				q := url.Values{}
				if f := ctx.Arg("file"); f != "" {
					q.Set("filepath", f)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("%s/pulls/%s/files", ctx.RepoPath(), id), q)
				if err != nil {
					return err
				}

				if ctx.Arg("stat") == "true" {
					return ctx.Output(formatDiffStat(env))
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
				{Name: "content", Short: "c", Usage: tr.T("flag.pr.review_content")},
				{Name: "body", Short: "b", Usage: tr.T("flag.pr.review_content")},
				{Name: "status", Short: "s", Usage: tr.T("flag.pr.review_status"), Default: "common"},
				{Name: "commit-id", Usage: tr.T("flag.pr.review_commit")},
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
				// --content 和 --body 互为别名，至少传一个。
				content := ctx.Arg("content")
				if content == "" {
					content, err = ctx.RequireArg("body")
					if err != nil {
						return err
					}
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
				if commit := ctx.Arg("commit-id"); commit != "" {
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
					return fmt.Errorf("获取 PR 详情失败: %w", err)
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
			Name:        "check-merge",
			Description: "Check if branches can be merged",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch", Required: true},
				{Name: "base", Short: "b", Usage: "Target branch", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, _ := ctx.RequireArg("head")
				base, _ := ctx.RequireArg("base")
				payload := map[string]interface{}{
					"head": head,
					"base": base,
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/pulls/check_can_merge", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "branches",
			Description: "List available branches for PR",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/pulls/get_branches", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

// getLatestVersionID calls the versions API and returns the latest version's ID.
// GitLink returns versions in reverse chronological order, so the first is latest.
func getLatestVersionID(ctx *common.RuntimeContext, prID string) (string, error) {
	env, err := ctx.CallAPI("GET",
		fmt.Sprintf("/v1%s/pulls/%s/versions", ctx.RepoPath(), prID), nil)
	if err != nil {
		return "", err
	}

	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected versions response format")
	}

	versions, ok := data["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("PR #%s 没有找到版本信息", prID)
	}

	latest, ok := versions[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected version format")
	}

	idFloat, ok := latest["id"].(float64)
	if !ok {
		return "", fmt.Errorf("version missing id field")
	}

	return fmt.Sprintf("%d", int64(idFloat)), nil
}

// formatDiffStat extracts add/delete statistics from the diff response.
func formatDiffStat(env *output.Envelope) *output.Envelope {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return env
	}

	stat := map[string]interface{}{
		"file_nums":      data["file_nums"],
		"total_addition": data["total_addition"],
		"total_deletion": data["total_deletion"],
	}

	if files, ok := data["files"].([]interface{}); ok {
		var fileStats []map[string]interface{}
		for _, f := range files {
			if fm, ok := f.(map[string]interface{}); ok {
				fileStats = append(fileStats, map[string]interface{}{
					"name":     fm["name"],
					"addition": fm["addition"],
					"deletion": fm["deletion"],
					"type":     fm["type"],
				})
			}
		}
		stat["files"] = fileStats
	}

	return output.SuccessEnvelope(stat, nil)
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

// prV1Path returns the v1 API path for a specific PR.
func prV1Path(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("/v1/%s/%s/pulls/%s", ctx.Owner, ctx.Repo, id)
}

// validatePRReviewStatus validates the review status value.
func validatePRReviewStatus(status string) error {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "common", "approved", "rejected", "":
		return nil
	default:
		return fmt.Errorf("invalid review status %q: use common, approved, or rejected", status)
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
