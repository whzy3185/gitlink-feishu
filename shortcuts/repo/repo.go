package repo

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
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
			Name:        "clone",
			Description: tr.T("cmd.repo.clone.short"),
			Long:        tr.T("cmd.repo.clone.long"),
			Flags: []common.Flag{
				{Name: "dir", Short: "d", Usage: tr.T("flag.repo.clone_dir")},
				{Name: "branch", Short: "b", Usage: tr.T("flag.repo.clone_branch")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
				if err != nil {
					return err
				}
				data, _ := env.Data.(map[string]interface{})
				cloneURL, _ := data["clone_url"].(string)
				if cloneURL == "" {
					return errors.New(tr.T("error.repo.clone_url_missing"))
				}
				args := []string{"clone", cloneURL}
				if branch := ctx.Arg("branch"); branch != "" {
					args = append(args, "--branch", branch)
				}
				if dir := ctx.Arg("dir"); dir != "" {
					args = append(args, dir)
				}
				gitCmd := exec.Command("git", args...)
				gitCmd.Stdout = os.Stderr
				gitCmd.Stderr = os.Stderr
				if err := gitCmd.Run(); err != nil {
					return fmt.Errorf("git clone failed: %w", err)
				}
				dest := ctx.Arg("dir")
				if dest == "" {
					dest = strings.TrimSuffix(cloneURL[strings.LastIndex(cloneURL, "/")+1:], ".git")
				}
				return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
					"message":   "cloned",
					"clone_url": cloneURL,
					"dir":       dest,
				}, nil))
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
			Name:        "edit",
			Description: tr.T("cmd.repo.edit.short"),
			Flags: []common.Flag{
				{Name: "description", Short: "d", Usage: tr.T("flag.repo.description")},
				{Name: "website", Usage: tr.T("flag.repo.edit.website")},
				{Name: "private", Usage: tr.T("flag.repo.private")},
				{Name: "default-branch", Usage: tr.T("flag.repo.edit.default_branch")},
				{Name: "category-id", Usage: tr.T("flag.repo.edit.category_id")},
				{Name: "language-id", Usage: tr.T("flag.repo.edit.language_id")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				private := ctx.Arg("private")
				if private != "" && private != "true" && private != "false" {
					return fmt.Errorf("--private must be true or false, got %q", private)
				}
				ids := map[string]interface{}{}
				for flag, key := range map[string]string{"category-id": "project_category_id", "language-id": "project_language_id"} {
					if v := ctx.Arg(flag); v != "" {
						id, err := strconv.Atoi(v)
						if err != nil {
							return fmt.Errorf("--%s must be an integer, got %q", flag, v)
						}
						ids[key] = id
					}
				}
				description := ctx.Arg("description")
				website := ctx.Arg("website")
				defaultBranch := ctx.Arg("default-branch")
				if description == "" && website == "" && defaultBranch == "" && private == "" && len(ids) == 0 {
					return fmt.Errorf("nothing to update: pass at least one of --description, --website, --private, --default-branch, --category-id, --language-id")
				}

				detail, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
				if err != nil {
					return err
				}
				data, _ := detail.Data.(map[string]interface{})
				name, _ := data["name"].(string)
				identifier, _ := data["identifier"].(string)
				if name == "" || identifier == "" {
					return fmt.Errorf("cannot resolve repository name/identifier from %s", ctx.RepoPath())
				}
				base := func() map[string]interface{} {
					return map[string]interface{}{"name": name, "identifier": identifier}
				}

				// The server dispatches on which key is present (website,
				// default_branch, or general metadata), so each group goes
				// out as its own request.
				if defaultBranch != "" {
					payload := base()
					payload["default_branch"] = defaultBranch
					if _, err := ctx.CallAPI("PATCH", ctx.RepoPath(), payload); err != nil {
						return err
					}
				}
				if website != "" {
					payload := base()
					payload["website"] = website
					if _, err := ctx.CallAPI("PATCH", ctx.RepoPath(), payload); err != nil {
						return err
					}
				}
				if description != "" || private != "" || len(ids) > 0 {
					payload := base()
					if description != "" {
						payload["description"] = description
					}
					if private != "" {
						payload["private"] = private == "true"
					}
					for k, v := range ids {
						payload[k] = v
					}
					if _, err := ctx.CallAPI("PATCH", ctx.RepoPath(), payload); err != nil {
						return err
					}
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
			Description: tr.T("cmd.repo.readme.short"),
			Flags: []common.Flag{
				{Name: "ref", Usage: tr.T("flag.repo.readme_ref")},
				{Name: "path", Usage: tr.T("flag.repo.readme_path")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if ref := strings.TrimSpace(ctx.Arg("ref")); ref != "" {
					q.Set("ref", ref)
				}
				if path := strings.Trim(strings.TrimSpace(ctx.Arg("path")), "/"); path != "" {
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
			Name:        "file",
			Description: "Show repository file content",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "Repository file path", Required: true},
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA", Default: "master"},
				{Name: "content-only", Usage: "Output file content only", Bool: true, Default: "false"},
			},
			Run: runFile,
		},
		{
			Name:        "tree",
			Description: tr.T("cmd.repo.tree.short"),
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: tr.T("flag.repo.tree.path")},
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.tree.ref")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if path := ctx.Arg("path"); path != "" {
					q.Set("filepath", path)
				}
				if ref := ctx.Arg("ref"); ref != "" {
					q.Set("ref", ref)
				}
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "blame",
			Description: tr.T("cmd.repo.blame.short"),
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: tr.T("flag.repo.blame.path"), Required: true},
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.tree.ref"), Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				ref := ctx.Arg("ref")
				if ref == "" {
					ref = "master"
				}
				q := url.Values{}
				q.Set("filepath", path)
				q.Set("sha", ref)
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/blame", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
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
			Name:        "activity",
			Description: tr.T("cmd.repo.activity.short"),
			Flags: []common.Flag{
				{Name: "type", Short: "t", Usage: tr.T("flag.repo.activity.type")},
				{Name: "status", Short: "s", Usage: tr.T("flag.repo.activity.status")},
				{Name: "time", Usage: tr.T("flag.repo.activity.time")},
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
				if trendType := ctx.Arg("type"); trendType != "" {
					q.Set("type", trendType)
				}
				if status := ctx.Arg("status"); status != "" {
					q.Set("status", status)
				}
				if timeDays := ctx.Arg("time"); timeDays != "" {
					if _, err := strconv.Atoi(timeDays); err != nil {
						return fmt.Errorf("--time must be an integer number of days, got %q", timeDays)
					}
					q.Set("time", timeDays)
				}
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/activity", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
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
			Name:        "forks",
			Description: tr.T("cmd.repo.forks.short"),
			Flags:       communityListFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runCommunityList(ctx, "forks")
			},
		},
		{
			Name:        "top-counts",
			Description: tr.T("cmd.repo.top_counts.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/top_counts", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
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
					return fmt.Errorf("failed to get current user: %w", err)
				}
				userData, _ := userEnv.Data.(map[string]interface{})
				login, _ := userData["login"].(string)
				if login == "" {
					return fmt.Errorf("cannot determine current user login")
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
			Name:        "transfer-orgs",
			Description: tr.T("cmd.repo.transfer_orgs.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", repoTransferPath(ctx, "organizations"), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "transfer",
			Description: tr.T("cmd.repo.transfer.short"),
			Flags: []common.Flag{
				{Name: "target-owner", Usage: tr.T("flag.repo.target_owner"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.repo.transfer_dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.repo.transfer_yes"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				targetOwner, err := ctx.RequireArg("target-owner")
				if err != nil {
					return err
				}
				targetOwner = strings.TrimSpace(targetOwner)
				if targetOwner == "" {
					return fmt.Errorf("required flag --target-owner is missing")
				}
				payload := map[string]interface{}{"owner_name": targetOwner}
				path := repoTransferPath(ctx, "")
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"method":  "POST",
						"path":    path,
						"payload": payload,
					})
				}
				if err := requireRepoTransferConfirmation(ctx, "transfer"); err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", path, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "transfer-cancel",
			Description: tr.T("cmd.repo.transfer_cancel.short"),
			Flags: []common.Flag{
				{Name: "dry-run", Usage: tr.T("flag.repo.transfer_cancel_dry_run"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.repo.transfer_yes"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path := repoTransferPath(ctx, "cancel")
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"method":  "POST",
						"path":    path,
					})
				}
				if err := requireRepoTransferConfirmation(ctx, "transfer-cancel"); err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", path, nil)
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

func repoTransferPath(ctx *common.RuntimeContext, action string) string {
	base := ctx.RepoPath() + "/applied_transfer_projects"
	if action == "" {
		return base
	}
	return fmt.Sprintf("%s/%s", base, action)
}

func requireRepoTransferConfirmation(ctx *common.RuntimeContext, shortcut string) error {
	if ctx.Arg("yes") == "true" {
		return nil
	}
	return fmt.Errorf("refusing to run repo +%s without --yes; use --dry-run to preview the request first", shortcut)
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
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

func runFile(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	path, err := normalizeRepoFilePath(ctx)
	if err != nil {
		return err
	}

	ref := strings.TrimSpace(ctx.Arg("ref"))
	if ref == "" {
		ref = "master"
	}

	q := url.Values{}
	q.Set("filepath", path)
	q.Set("ref", ref)

	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
	if err != nil {
		return err
	}

	entry, err := extractRepoFileEntry(env.Data, path)
	if err != nil {
		return err
	}
	if ctx.Arg("content-only") == "true" {
		content, _ := entry["content"].(string)
		if content == "" {
			return fmt.Errorf("file response did not include content for %q", path)
		}
		return ctx.OutputData(content)
	}

	return ctx.OutputData(buildRepoFileResult(entry, path, ref))
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

func normalizeRepoFilePath(ctx *common.RuntimeContext) (string, error) {
	path, err := ctx.RequireArg("path")
	if err != nil {
		return "", err
	}
	path = strings.TrimLeft(strings.TrimSpace(path), "/")
	if path == "" {
		return "", fmt.Errorf("invalid --path %q: provide a repository file path", ctx.Arg("path"))
	}
	return path, nil
}

func extractRepoFileEntry(data interface{}, path string) (map[string]interface{}, error) {
	payload, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected file response format")
	}

	entry, ok := payload["entries"]
	if !ok {
		return nil, fmt.Errorf("unexpected file response format")
	}

	if _, isDir := entry.([]interface{}); isDir {
		return nil, fmt.Errorf("path %q is a directory; use repo +tree instead", path)
	}

	fileEntry, ok := entry.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected file response format")
	}

	if entryType, _ := fileEntry["type"].(string); entryType != "" && entryType != "file" {
		return nil, fmt.Errorf("path %q is not a file; use repo +tree instead", path)
	}

	return fileEntry, nil
}

func buildRepoFileResult(entry map[string]interface{}, path, ref string) map[string]interface{} {
	result := map[string]interface{}{
		"path": path,
		"ref":  ref,
	}
	for _, key := range []string{"name", "type", "size", "sha", "content"} {
		if value, ok := entry[key]; ok {
			result[key] = value
		}
	}
	return result
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
