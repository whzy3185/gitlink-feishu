package trace

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns code trace analysis shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "init",
			Description: tr.T("cmd.trace.init.short"),
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("POST", "/api/traces/trace_users", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "results",
			Description: tr.T("cmd.trace.results.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "15"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				setQueryIfPresent(q, "page", ctx.Arg("page"))
				setQueryIfPresent(q, "limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", traceRepoPath(ctx)+"/task_results", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "start",
			Description: tr.T("cmd.trace.start.short"),
			Flags: []common.Flag{
				{Name: "branch", Short: "b", Usage: tr.T("flag.trace.branch"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runStart,
		},
		{
			Name:        "rescan",
			Description: tr.T("cmd.trace.rescan.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.trace.project_id"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runRescan,
		},
		{
			Name:        "report",
			Description: tr.T("cmd.trace.report.short"),
			Flags: []common.Flag{
				{Name: "task-id", Usage: tr.T("flag.trace.task_id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				taskID, err := ctx.RequireArg("task-id")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("task_id", taskID)
				env, err := ctx.CallAPIWithQuery("GET", traceRepoPath(ctx)+"/task_pdf", q)
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

func runStart(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	branch, err := requiredTrimmedArg(ctx, "branch")
	if err != nil {
		return err
	}
	payload := map[string]interface{}{"branch_name": branch}
	path := traceRepoPath(ctx) + "/tasks"
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(traceDryRun(ctx, "POST", path, payload, nil))
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRescan(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := requiredTrimmedArg(ctx, "project-id")
	if err != nil {
		return err
	}
	path := traceRepoPath(ctx) + "/reload_task"
	q := url.Values{}
	q.Set("project_id", projectID)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(traceDryRun(ctx, "GET", path, nil, q))
	}
	env, err := ctx.CallAPIWithQuery("GET", path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func traceRepoPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/api/traces/%s/%s", ctx.Owner, ctx.Repo)
}

func requiredTrimmedArg(ctx *common.RuntimeContext, name string) (string, error) {
	value, err := ctx.RequireArg(name)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("required flag --%s is missing", name)
	}
	return value, nil
}

func traceDryRun(ctx *common.RuntimeContext, method, path string, payload interface{}, query url.Values) map[string]interface{} {
	result := map[string]interface{}{
		"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"dry_run":    true,
		"method":     method,
		"path":       path,
	}
	if payload != nil {
		result["payload"] = payload
	}
	if len(query) > 0 {
		result["query"] = query.Encode()
	}
	return result
}

func setQueryIfPresent(q url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		q.Set(key, strings.TrimSpace(value))
	}
}
