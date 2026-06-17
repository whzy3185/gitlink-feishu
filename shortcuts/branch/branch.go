package branch

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
			Description: tr.T("cmd.branch.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
				{Name: "keyword", Short: "k", Usage: "Filter branches by keyword"},
				{Name: "state", Short: "s", Usage: "Branch state: all, deleted, or empty for active branches"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if keyword := strings.TrimSpace(ctx.Arg("keyword")); keyword != "" {
					q.Set("keyword", keyword)
				}
				if state := strings.TrimSpace(ctx.Arg("state")); state != "" {
					if err := validateBranchState(state); err != nil {
						return err
					}
					q.Set("state", state)
				}
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/branches", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "all",
			Description: "List all branches without pagination",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", "/v1"+ctx.RepoPath()+"/branches/all", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.branch.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
				{Name: "from", Short: "f", Usage: tr.T("flag.branch.from"), Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, _ := ctx.RequireArg("name")
				from := ctx.Arg("from")
				if from == "" {
					from = "master"
				}
				payload := map[string]interface{}{
					"new_branch_name": name,
					"old_branch_name": from,
				}
				env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/branches", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.branch.delete.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, _ := ctx.RequireArg("name")
				payload := map[string]interface{}{
					"branch_name": name,
				}
				env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/branches/delete", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "protect",
			Description: tr.T("cmd.branch.protect.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, _ := ctx.RequireArg("name")
				payload := map[string]interface{}{
					"branch_name": name,
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/protected_branches", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "unprotect",
			Description: tr.T("cmd.branch.unprotect.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, _ := ctx.RequireArg("name")
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/protected_branches/%s", ctx.RepoPath(), url.PathEscape(name)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "set-default",
			Description: "Set repository default branch",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
				{Name: "dry-run", Usage: "Preview the request without changing the default branch", Bool: true, Default: "false"},
			},
			Run: runSetDefault,
		},
		{
			Name:        "restore",
			Description: "Restore a deleted branch",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Deleted branch ID", Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
				{Name: "dry-run", Usage: "Preview the request without restoring the branch", Bool: true, Default: "false"},
			},
			Run: runRestore,
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func runSetDefault(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return err
	}
	path := "/v1" + ctx.RepoPath() + "/branches/update_default_branch"
	query := url.Values{}
	query.Set("name", name)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run": true,
			"action":  "set_default_branch",
			"method":  "PATCH",
			"path":    path,
			"query":   query.Encode(),
		})
	}
	env, err := ctx.CallAPIWithQuery("PATCH", path, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRestore(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := parsePositiveInt(ctx.Arg("id"), "id")
	if err != nil {
		return err
	}
	name, err := ctx.RequireArg("name")
	if err != nil {
		return err
	}
	path := "/v1" + ctx.RepoPath() + "/branches/restore"
	body := map[string]interface{}{
		"branch_id":   id,
		"branch_name": name,
	}
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run": true,
			"action":  "restore_branch",
			"method":  "POST",
			"path":    path,
			"body":    body,
		})
	}
	env, err := ctx.CallAPI("POST", path, body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func validateBranchState(state string) error {
	switch state {
	case "all", "deleted":
		return nil
	default:
		return fmt.Errorf("invalid --state %q: use all or deleted", state)
	}
}

func parsePositiveInt(value, name string) (int64, error) {
	value = strings.TrimSpace(value)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", name, value)
	}
	return parsed, nil
}
