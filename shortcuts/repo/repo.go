package repo

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.repo.list.short"),
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: tr.T("flag.user"), Default: ""},
				{Name: "category", Short: "c", Usage: tr.T("flag.repo.category"), Default: "manage"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				user := ctx.Arg("user")
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if cat := ctx.Arg("category"); cat != "" && cat != "all" {
					q.Set("category", cat)
				}

				path := "/projects"
				if user != "" {
					path = fmt.Sprintf("/users/%s/projects", user)
				}
				env, err := ctx.CallAPIWithQuery("GET", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.repo.info.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "readme",
			Description: "Show repository README content",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
				{Name: "path", Usage: "README directory path"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if ref := ctx.Arg("ref"); ref != "" {
					q.Set("ref", ref)
				}
				if path := ctx.Arg("path"); path != "" {
					q.Set("filepath", path)
				}
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/readme", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tree",
			Description: tr.T("cmd.repo.tree.short"),
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: tr.T("flag.repo.tree.path")},
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.tree.ref"), Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				ref := ctx.Arg("ref")
				if ref == "" {
					ref = "master"
				}
				if path := ctx.Arg("path"); path != "" {
					q.Set("filepath", path)
				}
				q.Set("ref", ref)
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: tr.T("cmd.repo.files.short"),
			Flags: []common.Flag{
				{Name: "search", Short: "s", Usage: tr.T("flag.repo.files.search")},
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.ref")},
			},
			Run: runFiles,
		},
		{
			Name:        "commits",
			Description: tr.T("cmd.repo.commits.short"),
			Flags: []common.Flag{
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.ref")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runCommits,
		},
		{
			Name:        "commit-files",
			Description: tr.T("cmd.repo.commit_files.short"),
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: tr.T("flag.repo.commit_sha"), Required: true},
				{Name: "file", Short: "f", Usage: tr.T("flag.repo.file")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runCommitFiles,
		},
		{
			Name:        "commit-diff",
			Description: tr.T("cmd.repo.commit_diff.short"),
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: tr.T("flag.repo.commit_sha"), Required: true},
			},
			Run: runCommitDiff,
		},
		{
			Name:        "tags",
			Description: tr.T("cmd.repo.tags.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.repo.tag_name_filter")},
				{Name: "only-name", Usage: tr.T("flag.repo.only_name"), Default: "false"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runTags,
		},
		{
			Name:        "tag",
			Description: tr.T("cmd.repo.tag.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.repo.tag_name"), Required: true},
			},
			Run: runTag,
		},
		{
			Name:        "delete-tag",
			Description: tr.T("cmd.repo.delete_tag.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.repo.tag_name"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.repo.delete_tag_dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.repo.delete_tag_yes"), Bool: true, Default: "false"},
			},
			Run: runDeleteTag,
		},
		{
			Name:        "batch-commit",
			Description: tr.T("cmd.repo.batch_commit.short"),
			Flags: []common.Flag{
				{Name: "branch", Short: "b", Usage: tr.T("flag.repo.batch_branch"), Required: true},
				{Name: "message", Short: "m", Usage: tr.T("flag.repo.batch_message"), Required: true},
				{Name: "files", Short: "f", Usage: tr.T("flag.repo.batch_files"), Required: true},
				{Name: "new-branch", Usage: tr.T("flag.repo.batch_new_branch")},
				{Name: "encoding", Usage: tr.T("flag.repo.batch_encoding"), Default: "text"},
				{Name: "author-name", Usage: tr.T("flag.repo.author_name")},
				{Name: "author-email", Usage: tr.T("flag.repo.author_email")},
				{Name: "committer-name", Usage: tr.T("flag.repo.committer_name")},
				{Name: "committer-email", Usage: tr.T("flag.repo.committer_email")},
				{Name: "dry-run", Usage: tr.T("flag.repo.batch_dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.repo.batch_yes"), Bool: true, Default: "false"},
			},
			Run: runBatchCommit,
		},
		{
			Name:        "languages",
			Description: "Show repository language statistics",
			Run:         runLanguages,
		},
		{
			Name:        "contributors",
			Description: "List repository contributors",
			Run:         runContributors,
		},
		{
			Name:        "contributor-stats",
			Description: "List contributor statistics with code line counts",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
				{Name: "pass-year", Usage: "Number of past years to include"},
			},
			Run: runContributorStats,
		},
		{
			Name:        "code-stats",
			Description: "Show repository code statistics",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
			},
			Run: runCodeStats,
		},
		{
			Name:        "watchers",
			Description: "List repository watchers",
			Flags:       communityListFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runCommunityList(ctx, "watchers")
			},
		},
		{
			Name:        "stargazers",
			Description: "List repository stargazers",
			Flags:       communityListFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runCommunityList(ctx, "stargazers")
			},
		},
		{
			Name:        "follow",
			Description: "Follow a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoFollowAction(ctx, "POST", "/watchers/follow", "follow")
			},
		},
		{
			Name:        "unfollow",
			Description: "Unfollow a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoFollowAction(ctx, "DELETE", "/watchers/unfollow", "unfollow")
			},
		},
		{
			Name:        "like",
			Description: "Like a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoPraiseAction(ctx, "POST", "like")
			},
		},
		{
			Name:        "unlike",
			Description: "Unlike a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoPraiseAction(ctx, "DELETE", "unlike")
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.repo.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.repo.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.repo.description")},
				{Name: "private", Usage: tr.T("flag.repo.private"), Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				// Get current user login for the create path
				userEnv, err := ctx.CallAPI("GET", "/users/me", nil)
				if err != nil {
					return fmt.Errorf("获取当前用户信息失败: %w", err)
				}
				userData, _ := userEnv.Data.(map[string]interface{})
				login, _ := userData["login"].(string)
				if login == "" {
					return fmt.Errorf("无法确定当前用户")
				}
				userID, _ := userData["user_id"].(float64)
				body := map[string]interface{}{
					"name":            name,
					"repository_name": name,
					"user_id":         int(userID),
				}
				if desc := ctx.Arg("description"); desc != "" {
					body["description"] = desc
				}
				if ctx.Arg("private") == "true" {
					body["private"] = true
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("/%s/%s", login, name), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "fork",
			Description: tr.T("cmd.repo.fork.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/forks", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.repo.delete.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", ctx.RepoPath(), nil)
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

func runFiles(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "search", ctx.Arg("search"))
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/files", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommits(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	q.Set("page", firstRepoValue(ctx.Arg("page"), "1"))
	q.Set("limit", firstRepoValue(ctx.Arg("limit"), "20"))
	setRepoQueryIfPresent(q, "sha", ctx.Arg("ref"))
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/commits", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommitFiles(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	sha, err := ctx.RequireArg("sha")
	if err != nil {
		return err
	}
	q := url.Values{}
	if file := strings.TrimSpace(ctx.Arg("file")); file != "" {
		q.Set("filepath", file)
	} else {
		q.Set("page", firstRepoValue(ctx.Arg("page"), "1"))
		q.Set("limit", firstRepoValue(ctx.Arg("limit"), "20"))
	}
	env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/v1%s/commits/%s/files", ctx.RepoPath(), url.PathEscape(sha)), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommitDiff(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	sha, err := ctx.RequireArg("sha")
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", fmt.Sprintf("/v1%s/commits/%s/diff", ctx.RepoPath(), url.PathEscape(sha)), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runTags(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	if name := strings.TrimSpace(ctx.Arg("name")); name != "" {
		q.Set("name", name)
	}
	onlyName := strings.TrimSpace(ctx.Arg("only-name"))
	if onlyName != "" && onlyName != "false" {
		q.Set("only_name", onlyName)
	}
	if len(q) > 0 {
		env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/tags", q)
		if err != nil {
			return err
		}
		return ctx.Output(env)
	}
	q.Set("page", firstRepoValue(ctx.Arg("page"), "1"))
	q.Set("limit", firstRepoValue(ctx.Arg("limit"), "20"))
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/tags", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runTag(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", fmt.Sprintf("/v1%s/tags/%s", ctx.RepoPath(), url.PathEscape(name)), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDeleteTag(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/v1%s/tags/%s", ctx.RepoPath(), url.PathEscape(name))
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":    true,
			"action":     "delete_tag",
			"method":     "DELETE",
			"path":       path,
			"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"tag":        name,
		})
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("tag deletion is destructive; run --dry-run first, then pass --yes to execute")
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runBatchCommit(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	payload, err := batchCommitPayload(ctx)
	if err != nil {
		return err
	}
	path := "/v1" + ctx.RepoPath() + "/contents/batch"
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":    true,
			"action":     "batch_commit",
			"method":     "POST",
			"path":       path,
			"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"payload":    payload,
		})
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("batch file commit changes repository content; run --dry-run first, then pass --yes to execute")
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runLanguages(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/languages", nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runContributors(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/contributors", nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runContributorStats(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	if passYear := ctx.Arg("pass-year"); strings.TrimSpace(passYear) != "" {
		year, err := parseRepoPositiveInt(passYear, "pass-year")
		if err != nil {
			return err
		}
		q.Set("pass_year", strconv.Itoa(year))
	}
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/contributors/stat", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCodeStats(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/code_stats", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommunityList(ctx *common.RuntimeContext, path string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q, err := communityTimeRangeQuery(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/"+path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func communityListFlags() []common.Flag {
	return []common.Flag{
		{Name: "start-at", Usage: "Start timestamp"},
		{Name: "end-at", Usage: "End timestamp"},
	}
}

func repoInteractionFlags() []common.Flag {
	return []common.Flag{
		{Name: "project-id", Usage: "GitLink project ID. If omitted, it is resolved from --owner/--repo."},
		{Name: "dry-run", Usage: "Preview the action without changing repository state", Bool: true, Default: "false"},
	}
}

func runRepoFollowAction(ctx *common.RuntimeContext, method, path, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveRepoProjectID(ctx)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("target_type", "project")
	query.Set("id", projectID)
	if ctx.Arg("dry-run") == "true" {
		return repoInteractionDryRun(ctx, method, path, action, projectID, query)
	}
	env, err := ctx.CallAPIWithQuery(method, path, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRepoPraiseAction(ctx *common.RuntimeContext, method, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveRepoProjectID(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/projects/%s/praise_tread/%s", projectID, action)
	if ctx.Arg("dry-run") == "true" {
		return repoInteractionDryRun(ctx, method, path, action, projectID, nil)
	}
	env, err := ctx.CallAPI(method, path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func repoInteractionDryRun(ctx *common.RuntimeContext, method, path, action, projectID string, query url.Values) error {
	preview := map[string]interface{}{
		"dry_run":    true,
		"action":     action,
		"method":     method,
		"path":       path,
		"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"project_id": projectID,
	}
	if len(query) > 0 {
		preview["query"] = query.Encode()
	}
	return ctx.OutputData(preview)
}

func resolveRepoProjectID(ctx *common.RuntimeContext) (string, error) {
	if raw := strings.TrimSpace(ctx.Arg("project-id")); raw != "" {
		return normalizeRepoProjectID(raw)
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return "", fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"id", "project_id"} {
		if id := repoProjectIDString(data[key]); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("resolve project id: repository response did not include id")
}

func normalizeRepoProjectID(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return "", fmt.Errorf("invalid --project-id %q: use a positive numeric project ID", value)
	}
	return strconv.FormatInt(parsed, 10), nil
}

func repoProjectIDString(value interface{}) string {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return strconv.FormatInt(int64(v), 10)
		}
	case int:
		if v > 0 {
			return strconv.Itoa(v)
		}
	case int64:
		if v > 0 {
			return strconv.FormatInt(v, 10)
		}
	case string:
		id, err := normalizeRepoProjectID(v)
		if err == nil {
			return id
		}
	}
	return ""
}

func communityTimeRangeQuery(ctx *common.RuntimeContext) (url.Values, error) {
	q := url.Values{}
	start, hasStart, err := parseOptionalRepoNonNegativeInt(ctx.Arg("start-at"), "start-at")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := parseOptionalRepoNonNegativeInt(ctx.Arg("end-at"), "end-at")
	if err != nil {
		return nil, err
	}
	if hasStart {
		q.Set("start_at", strconv.Itoa(start))
	}
	if hasEnd {
		q.Set("end_at", strconv.Itoa(end))
	}
	if hasStart && hasEnd && start > end {
		return nil, fmt.Errorf("--start-at must be less than or equal to --end-at")
	}
	return q, nil
}

func setRepoQueryIfPresent(q url.Values, key, value string) {
	if value := strings.TrimSpace(value); value != "" {
		q.Set(key, value)
	}
}

func firstRepoValue(value, fallback string) string {
	if value := strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func batchCommitPayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	branch, err := ctx.RequireArg("branch")
	if err != nil {
		return nil, err
	}
	message, err := ctx.RequireArg("message")
	if err != nil {
		return nil, err
	}
	rawFiles, err := ctx.RequireArg("files")
	if err != nil {
		return nil, err
	}
	encoding := firstRepoValue(ctx.Arg("encoding"), "text")
	if encoding != "text" && encoding != "base64" {
		return nil, fmt.Errorf("invalid --encoding %q: use text or base64", encoding)
	}
	files, err := parseBatchFileSpecs(rawFiles, encoding)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"branch":  strings.TrimSpace(branch),
		"message": message,
		"files":   files,
	}
	if newBranch := strings.TrimSpace(ctx.Arg("new-branch")); newBranch != "" {
		payload["new_branch"] = newBranch
	}
	setRepoPayloadIfPresent(payload, "author_name", ctx.Arg("author-name"))
	setRepoPayloadIfPresent(payload, "author_email", ctx.Arg("author-email"))
	setRepoPayloadIfPresent(payload, "committer_name", ctx.Arg("committer-name"))
	setRepoPayloadIfPresent(payload, "committer_email", ctx.Arg("committer-email"))
	return payload, nil
}

func parseBatchFileSpecs(raw, encoding string) ([]map[string]interface{}, error) {
	specs := strings.Split(raw, ";")
	files := make([]map[string]interface{}, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid --files item %q: use action:path[:content]", spec)
		}
		action := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		if !isBatchFileAction(action) {
			return nil, fmt.Errorf("invalid file action %q: use create, update, or delete", action)
		}
		if path == "" {
			return nil, fmt.Errorf("invalid --files item %q: file path is required", spec)
		}
		item := map[string]interface{}{
			"action_type": action,
			"file_path":   path,
		}
		if action != "delete" {
			if len(parts) != 3 {
				return nil, fmt.Errorf("file action %q for %q requires content", action, path)
			}
			item["content"] = parts[2]
			item["encoding"] = encoding
		}
		files = append(files, item)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("--files must include at least one file operation")
	}
	return files, nil
}

func isBatchFileAction(action string) bool {
	switch action {
	case "create", "update", "delete":
		return true
	default:
		return false
	}
}

func setRepoPayloadIfPresent(payload map[string]interface{}, key, value string) {
	if value := strings.TrimSpace(value); value != "" {
		payload[key] = value
	}
}

func parseOptionalRepoNonNegativeInt(value, name string) (int, bool, error) {
	if strings.TrimSpace(value) == "" {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, false, fmt.Errorf("invalid --%s %q: use a non-negative integer", name, value)
	}
	return parsed, true, nil
}

func parseRepoPositiveInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", name, value)
	}
	return parsed, nil
}
