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
			Name:        "detail",
			Description: "Show repository detail metadata",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/detail", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "simple",
			Description: "Show simplified repository metadata",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/simple", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "settings",
			Description: "Show repository settings metadata",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/edit", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "units",
			Description: "Show repository navigation units",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/project_units", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "units-update",
			Description: "Update repository navigation units",
			Flags: []common.Flag{
				{Name: "units", Short: "u", Usage: "Comma-separated units, for example: code,issues,pulls,wiki", Required: true},
				{Name: "dry-run", Usage: "Preview the update request without changing repository units", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				units, err := parseCSVArg(ctx.Arg("units"), "--units")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{"unit_types": units}
				path := ctx.RepoPath() + "/project_units"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "update_repository_units",
						"method":     "POST",
						"path":       path,
						"payload":    payload,
					})
				}
				env, err := ctx.CallAPI("POST", path, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "topics",
			Description: "List project topic labels",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Topic keyword filter"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				setQueryIfPresent(q, ctx, "keyword", "keyword")
				env, err := ctx.CallAPIWithQuery("GET", "/v1/project_topics", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "topic-add",
			Description: "Add a topic label to a repository",
			Flags: []common.Flag{
				{Name: "project-id", Usage: "Repository project ID", Required: true},
				{Name: "name", Short: "n", Usage: "Topic name", Required: true},
				{Name: "dry-run", Usage: "Preview the create request without changing topics", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				projectID, err := requiredPositiveIntArg(ctx, "project-id")
				if err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"project_id": projectID,
					"name":       name,
				}
				path := "/v1/project_topics"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"action":  "add_repository_topic",
						"method":  "POST",
						"path":    path,
						"payload": payload,
					})
				}
				env, err := ctx.CallAPI("POST", path, payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "topic-delete",
			Description: "Remove a topic label from a repository",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Project topic ID", Required: true},
				{Name: "project-id", Usage: "Repository project ID", Required: true},
				{Name: "dry-run", Usage: "Preview the delete request without changing topics", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				projectID, err := ctx.RequireArg("project-id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/v1/project_topics/%s", url.PathEscape(id))
				q := url.Values{"project_id": []string{projectID}}
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"action":  "delete_repository_topic",
						"method":  "DELETE",
						"path":    path,
						"query":   q,
					})
				}
				env, err := ctx.CallAPIWithQuery("DELETE", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "transfer-orgs",
			Description: "List organizations available for repository transfer",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/applied_transfer_projects/organizations", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "transfer",
			Description: "Transfer a repository to another owner",
			Flags: []common.Flag{
				{Name: "owner-name", Usage: "Target owner login or organization name", Required: true},
				{Name: "dry-run", Usage: "Preview the transfer request without changing ownership", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				ownerName, err := ctx.RequireArg("owner-name")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{"owner_name": ownerName}
				path := ctx.RepoPath() + "/applied_transfer_projects"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "transfer_repository",
						"method":     "POST",
						"path":       path,
						"payload":    payload,
					})
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
			Description: "Cancel a pending repository transfer",
			Flags: []common.Flag{
				{Name: "dry-run", Usage: "Preview the cancel request without changing transfer state", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path := ctx.RepoPath() + "/applied_transfer_projects/cancel"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "cancel_repository_transfer",
						"method":     "POST",
						"path":       path,
					})
				}
				env, err := ctx.CallAPI("POST", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
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

func parseCSVArg(value, flagName string) ([]string, error) {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("%s must include at least one value", flagName)
	}
	return result, nil
}

func requiredPositiveIntArg(ctx *common.RuntimeContext, flagName string) (int, error) {
	value, err := ctx.RequireArg(flagName)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", flagName)
	}
	return id, nil
}

func setQueryIfPresent(q url.Values, ctx *common.RuntimeContext, flagName, queryName string) {
	if value := ctx.Arg(flagName); value != "" {
		q.Set(queryName, value)
	}
}
