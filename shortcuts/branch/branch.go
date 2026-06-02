package branch

import (
	"fmt"
	"net/url"

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
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/branches", q)
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
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
