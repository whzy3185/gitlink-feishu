package branch

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

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.branch.list.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
				{Name: "state", Short: "s", Usage: "Branch state: all or deleted"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				state, err := normalizeBranchState(ctx.Arg("state"))
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", defaultBranchValue(ctx.Arg("page"), "1"))
				q.Set("limit", defaultBranchValue(ctx.Arg("limit"), "20"))
				if keyword := ctx.Arg("keyword"); keyword != "" {
					q.Set("keyword", keyword)
				}
				if state != "" {
					q.Set("state", state)
				}
				env, err := ctx.CallAPIWithQuery("GET", branchPath(ctx), q)
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
				env, err := ctx.CallAPI("GET", branchPath(ctx)+"/all", nil)
				if err != nil {
					return err
				}
				return outputBranchEnvelope(ctx, env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.branch.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.branch.name"), Required: true},
				{Name: "from", Short: "f", Usage: tr.T("flag.branch.from"), Default: "master"},
				{Name: "dry-run", Usage: "Preview the request body without creating the branch", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				from := defaultBranchValue(ctx.Arg("from"), "master")
				payload := map[string]interface{}{
					"new_branch_name": name,
					"old_branch_name": from,
				}
				if parseBranchBool(ctx.Arg("dry-run")) {
					return ctx.OutputData(branchDryRun("create_branch", "POST", branchPath(ctx), payload, nil))
				}
				env, err := ctx.CallAPI("POST", branchPath(ctx), payload)
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
				{Name: "dry-run", Usage: "Preview the delete request without deleting the branch", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/%s", branchPath(ctx), url.PathEscape(name))
				if parseBranchBool(ctx.Arg("dry-run")) {
					return ctx.OutputData(branchDryRun("delete_branch", "DELETE", path, nil, nil))
				}
				env, err := ctx.CallAPI("DELETE", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "set-default",
			Description: "Set the repository default branch",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Branch name to set as default", Required: true},
				{Name: "dry-run", Usage: "Preview the request without changing the default branch", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				path := branchPath(ctx) + "/update_default_branch"
				q := url.Values{}
				q.Set("name", name)
				if parseBranchBool(ctx.Arg("dry-run")) {
					return ctx.OutputData(branchDryRun("set_default_branch", "PATCH", path, nil, q))
				}
				env, err := ctx.CallAPIWithQuery("PATCH", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "restore",
			Description: "Restore a deleted branch",
			Flags: []common.Flag{
				{Name: "branch-id", Short: "i", Usage: "Deleted branch ID", Required: true},
				{Name: "name", Short: "n", Usage: "Deleted branch name", Required: true},
				{Name: "dry-run", Usage: "Preview the request body without restoring the branch", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				branchIDValue, err := ctx.RequireArg("branch-id")
				if err != nil {
					return err
				}
				branchID, err := parsePositiveBranchInt(branchIDValue, "branch-id")
				if err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"branch_id":   branchID,
					"branch_name": name,
				}
				path := branchPath(ctx) + "/restore"
				if parseBranchBool(ctx.Arg("dry-run")) {
					return ctx.OutputData(branchDryRun("restore_branch", "POST", path, payload, nil))
				}
				env, err := ctx.CallAPI("POST", path, payload)
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
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func branchPath(ctx *common.RuntimeContext) string {
	return "/v1" + ctx.RepoPath() + "/branches"
}

func outputBranchEnvelope(ctx *common.RuntimeContext, env *output.Envelope) error {
	if raw, ok := env.Data.(string); ok {
		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return ctx.OutputData(parsed)
		}
	}
	return ctx.Output(env)
}

func normalizeBranchState(value string) (string, error) {
	state := strings.ToLower(strings.TrimSpace(value))
	if state == "" {
		return "", nil
	}
	switch state {
	case "all", "deleted":
		return state, nil
	default:
		return "", fmt.Errorf("invalid --state %q: use all or deleted", value)
	}
}

func parsePositiveBranchInt(value, flagName string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", flagName, value)
	}
	return id, nil
}

func defaultBranchValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseBranchBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func branchDryRun(action, method, path string, body map[string]interface{}, query url.Values) map[string]interface{} {
	data := map[string]interface{}{
		"dry_run": true,
		"action":  action,
		"method":  method,
		"path":    path,
	}
	if body != nil {
		data["body"] = body
	}
	if len(query) > 0 {
		data["query"] = query.Encode()
	}
	return data
}
