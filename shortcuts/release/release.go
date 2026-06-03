package release

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.release.list.short"),
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
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/releases", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.release.create.short"),
			Flags: []common.Flag{
				{Name: "tag", Short: "t", Usage: tr.T("flag.release.tag"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.release.name"), Required: true},
				{Name: "body", Short: "b", Usage: tr.T("flag.release.body")},
				{Name: "target", Usage: tr.T("flag.release.target"), Default: "master"},
				{Name: "prerelease", Usage: tr.T("flag.release.prerelease"), Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				tag, _ := ctx.RequireArg("tag")
				name, _ := ctx.RequireArg("name")
				payload := map[string]interface{}{
					"tag_name": tag,
					"name":     name,
				}
				if b := ctx.Arg("body"); b != "" {
					payload["body"] = b
				}
				if t := ctx.Arg("target"); t != "" {
					payload["target_commitish"] = t
				}
				if ctx.Arg("prerelease") == "true" {
					payload["prerelease"] = true
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/releases", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.release.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id_or_tag"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.release.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.release.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, _ := ctx.RequireArg("id")
				_, delErr := ctx.CallAPI("DELETE", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
				if delErr != nil {
					// GitLink API bug: delete succeeds but returns error status.
					// Verify by checking if the release still exists.
					_, viewErr := ctx.CallAPI("GET", fmt.Sprintf("%s/releases/%s", ctx.RepoPath(), id), nil)
					if viewErr != nil {
						// Release no longer exists — delete actually succeeded
						return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
							"message": "删除成功",
						}, nil))
					}
					// Release still exists — delete truly failed
					return delErr
				}
				return ctx.Output(output.SuccessEnvelope(map[string]interface{}{
					"message": "删除成功",
				}, nil))
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
