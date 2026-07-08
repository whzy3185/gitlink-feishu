package action

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func actionsPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/actions", ctx.Owner, ctx.Repo)
}

func requireIntArg(ctx *common.RuntimeContext, name string) (string, error) {
	value, err := ctx.RequireArg(name)
	if err != nil {
		return "", err
	}
	if _, err := strconv.Atoi(value); err != nil {
		return "", fmt.Errorf("--%s must be an integer, got %q", name, value)
	}
	return value, nil
}

// Shortcuts returns Gitea Actions (CI workflow) shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.action.list.short"),
			Flags:       []common.Flag{},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", actionsPath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "runs",
			Description: tr.T("cmd.action.runs.short"),
			Flags: []common.Flag{
				{Name: "workflow", Short: "w", Usage: tr.T("flag.action.workflow"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				workflow, err := ctx.RequireArg("workflow")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("workflow", workflow)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", actionsPath(ctx)+"/runs", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "run",
			Description: tr.T("cmd.action.run.short"),
			Flags: []common.Flag{
				{Name: "workflow", Short: "w", Usage: tr.T("flag.action.workflow"), Required: true},
				{Name: "ref", Short: "r", Usage: tr.T("flag.action.ref"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				workflow, err := ctx.RequireArg("workflow")
				if err != nil {
					return err
				}
				ref, err := ctx.RequireArg("ref")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("workflow", workflow)
				q.Set("ref", ref)
				env, err := ctx.CallAPIWithQuery("POST", actionsPath(ctx)+"/runs", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "rerun",
			Description: tr.T("cmd.action.rerun.short"),
			Flags: []common.Flag{
				{Name: "run-id", Short: "i", Usage: tr.T("flag.action.run_id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				runID, err := requireIntArg(ctx, "run-id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/runs/%s/rerun", actionsPath(ctx), runID), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "job-rerun",
			Description: tr.T("cmd.action.job_rerun.short"),
			Flags: []common.Flag{
				{Name: "run-id", Short: "i", Usage: tr.T("flag.action.run_id"), Required: true},
				{Name: "job", Short: "j", Usage: tr.T("flag.action.job"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				runID, err := requireIntArg(ctx, "run-id")
				if err != nil {
					return err
				}
				job, err := ctx.RequireArg("job")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/runs/%s/jobs/%s/rerun", actionsPath(ctx), runID, url.PathEscape(job)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "enable",
			Description: tr.T("cmd.action.enable.short"),
			Flags: []common.Flag{
				{Name: "workflow", Short: "w", Usage: tr.T("flag.action.workflow"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				workflow, err := ctx.RequireArg("workflow")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("workflow", workflow)
				env, err := ctx.CallAPIWithQuery("POST", actionsPath(ctx)+"/enable", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "disable",
			Description: tr.T("cmd.action.disable.short"),
			Flags: []common.Flag{
				{Name: "workflow", Short: "w", Usage: tr.T("flag.action.workflow"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				workflow, err := ctx.RequireArg("workflow")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("workflow", workflow)
				env, err := ctx.CallAPIWithQuery("POST", actionsPath(ctx)+"/disable", q)
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
