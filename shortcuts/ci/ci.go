package ci

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "builds",
			Description: tr.T("cmd.ci.builds.short"),
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
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/builds", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "logs",
			Description: tr.T("cmd.ci.logs.short"),
			Flags: []common.Flag{
				{Name: "build", Short: "b", Usage: tr.T("flag.ci.build"), Required: true},
				{Name: "stage", Short: "s", Usage: tr.T("flag.ci.stage"), Default: "1"},
				{Name: "step", Usage: tr.T("flag.ci.step"), Default: "1"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				build, _ := ctx.RequireArg("build")
				stage := ctx.Arg("stage")
				step := ctx.Arg("step")
				if stage == "" {
					stage = "1"
				}
				if step == "" {
					step = "1"
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/builds/%s/logs/%s/%s", ctx.RepoPath(), build, stage, step), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "restart",
			Description: tr.T("cmd.ci.restart.short"),
			Flags: []common.Flag{
				{Name: "build", Short: "b", Usage: tr.T("flag.ci.build"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				build, _ := ctx.RequireArg("build")
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/builds/%s/restart", ctx.RepoPath(), build), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "stop",
			Description: tr.T("cmd.ci.stop.short"),
			Flags: []common.Flag{
				{Name: "build", Short: "b", Usage: tr.T("flag.ci.build"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				build, _ := ctx.RequireArg("build")
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/builds/%s/stop", ctx.RepoPath(), build), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "activate",
			Description: tr.T("cmd.ci.activate.short"),
			Flags:       ciControlFlags(tr, "flag.ci.activate_dry_run", "flag.ci.activate_yes"),
			Run: func(ctx *common.RuntimeContext) error {
				return runCIControl(ctx, "activate_ci", "POST", "activate", "activating CI changes repository CI state; run --dry-run first, then pass --yes to execute")
			},
		},
		{
			Name:        "deactivate",
			Description: tr.T("cmd.ci.deactivate.short"),
			Flags:       ciControlFlags(tr, "flag.ci.deactivate_dry_run", "flag.ci.deactivate_yes"),
			Run: func(ctx *common.RuntimeContext) error {
				return runCIControl(ctx, "deactivate_ci", "DELETE", "deactivate", "deactivating CI changes repository CI state; run --dry-run first, then pass --yes to execute")
			},
		},
		{
			Name:        "authorize",
			Description: tr.T("cmd.ci.authorize.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/ci_authorize", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		newCIToggleShortcut("enable"),
		newCIToggleShortcut("disable"),
	}
}

func newCIToggleShortcut(action string) *common.Shortcut {
	return &common.Shortcut{
		Name:        action,
		Description: action + " repository actions",
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			env, err := ctx.CallAPI("POST", fmt.Sprintf("/v1/%s/%s/actions/%s", ctx.Owner, ctx.Repo, action), nil)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func ciControlFlags(tr *i18n.Translator, dryRunKey, yesKey string) []common.Flag {
	return []common.Flag{
		{Name: "dry-run", Usage: tr.T(dryRunKey), Bool: true, Default: "false"},
		{Name: "yes", Usage: tr.T(yesKey), Bool: true, Default: "false"},
	}
}

func runCIControl(ctx *common.RuntimeContext, action, method, suffix, confirmMessage string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	path := ctx.RepoPath() + "/" + suffix
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":    true,
			"action":     action,
			"method":     method,
			"path":       path,
			"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		})
	}
	if ctx.Arg("yes") != "true" {
		return errors.New(confirmMessage)
	}
	env, err := ctx.CallAPI(method, path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
