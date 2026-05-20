package pr

import (
	"encoding/base64"
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
				base := ctx.Arg("base")
				if base == "" {
					base = "master"
				}
				payload, err := buildCreatePRPayload(ctx, title, head, base, ctx.Arg("body"))
				if err != nil {
					return err
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

type prHeadSpec struct {
	Branch      string
	ForkOwner   string
	ForkRepo    string
	IsFork      bool
	CompareHead string
}

func buildCreatePRPayload(ctx *common.RuntimeContext, title, head, base, body string) (map[string]interface{}, error) {
	spec, err := parsePRHead(head)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"title":            title,
		"head":             spec.Branch,
		"base":             base,
		"body":             body,
		"assigned_to_id":   "",
		"fixed_version_id": "",
		"issue_tag_ids":    []string{},
		"reviewer_ids":     []string{},
		"receivers_login":  []string{},
		"priority_id":      "2",
		"is_original":      spec.IsFork,
	}

	if spec.IsFork {
		repoInfo, err := fetchProjectInfo(ctx, spec.ForkOwner, spec.ForkRepo)
		if err != nil {
			return nil, err
		}
		projectID, err := extractFloatField(repoInfo, "project_id", "id")
		if err != nil {
			return nil, fmt.Errorf("resolve fork project id: %w", err)
		}
		identifier, err := extractStringField(repoInfo, "project_identifier", "identifier")
		if err != nil {
			return nil, fmt.Errorf("resolve fork project identifier: %w", err)
		}
		payload["merge_user_login"] = spec.ForkOwner
		payload["merge_project_identifier"] = identifier
		payload["fork_project_id"] = int(projectID)
	}

	compareCounts, err := fetchPRCompareCounts(ctx, spec.CompareHead, base)
	if err == nil {
		if commits, ok := compareCounts["commits_count"]; ok {
			payload["commits_count"] = commits
		}
		if files, ok := compareCounts["files_count"]; ok {
			payload["files_count"] = files
		}
	}

	return payload, nil
}

func parsePRHead(head string) (*prHeadSpec, error) {
	if head == "" {
		return nil, fmt.Errorf("source branch cannot be empty")
	}
	if !strings.Contains(head, ":") {
		return &prHeadSpec{
			Branch:      head,
			IsFork:      false,
			CompareHead: head,
		}, nil
	}

	parts := strings.SplitN(head, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid --head %q, expected owner/repo:branch", head)
	}

	repoParts := strings.Split(parts[0], "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return nil, fmt.Errorf("invalid --head %q, expected owner/repo:branch", head)
	}

	return &prHeadSpec{
		Branch:      parts[1],
		ForkOwner:   repoParts[0],
		ForkRepo:    repoParts[1],
		IsFork:      true,
		CompareHead: repoParts[0] + ":" + parts[1],
	}, nil
}

func fetchProjectInfo(ctx *common.RuntimeContext, owner, repo string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", fmt.Sprintf("/%s/%s", owner, repo), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch project info: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected project info response format")
	}
	return data, nil
}

func fetchPRCompareCounts(ctx *common.RuntimeContext, head, base string) (map[string]int, error) {
	encodedHead := base64.RawURLEncoding.EncodeToString([]byte(head))
	encodedBase := base64.RawURLEncoding.EncodeToString([]byte(base))
	env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/compare/%s...%s", ctx.RepoPath(), encodedHead, encodedBase), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected compare response format")
	}
	result := map[string]int{}
	if v, ok := data["commits_count"].(float64); ok {
		result["commits_count"] = int(v)
	}
	if v, ok := data["files_count"].(float64); ok {
		result["files_count"] = int(v)
	}
	return result, nil
}

func extractFloatField(data map[string]interface{}, keys ...string) (float64, error) {
	for _, key := range keys {
		if v, ok := data[key].(float64); ok {
			return v, nil
		}
	}
	return 0, fmt.Errorf("missing numeric field %v", keys)
}

func extractStringField(data map[string]interface{}, keys ...string) (string, error) {
	for _, key := range keys {
		if v, ok := data[key].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("missing string field %v", keys)
}
