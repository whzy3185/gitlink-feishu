package pipeline

import (
	"encoding/json"
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
			Description: tr.T("cmd.pipeline.list.short"),
			Flags: []common.Flag{
				{Name: "owner-id", Usage: tr.T("flag.pipeline.owner_id")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := pageLimitQuery(ctx)
				setQueryIfPresent(q, ctx, "owner-id", "owner_id")
				env, err := ctx.CallAPIWithQuery("GET", "/pm/pipelines", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "runs",
			Description: tr.T("cmd.pipeline.runs.short"),
			Flags:       runFilterFlags(tr),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", pipelineV1RepoPath(ctx)+"/actions/runs", runFilterQuery(ctx))
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "run",
			Description: tr.T("cmd.pipeline.run.short"),
			Flags: append(runFilterFlags(tr),
				common.Flag{Name: "dry-run", Usage: tr.T("flag.pipeline.dry_run"), Bool: true, Default: "false"},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := runFilterQuery(ctx)
				path := pipelineV1RepoPath(ctx) + "/actions/runs"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "run_pipeline",
						"method":     "POST",
						"path":       path,
						"query":      q,
					})
				}
				env, err := ctx.CallAPIWithQuery("POST", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.pipeline.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pipeline.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := requiredPositiveInt(ctx, "id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/pipelines/%d", pipelineV1RepoPath(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.pipeline.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pipeline.id"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.pipeline.dry_run_2"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := requiredPositiveInt(ctx, "id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/pipelines/%d", pipelineV1RepoPath(ctx), id)
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "delete_pipeline",
						"method":     "DELETE",
						"path":       path,
					})
				}
				env, err := ctx.CallAPI("DELETE", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "save-yaml",
			Description: tr.T("cmd.pipeline.save-yaml.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.pipeline.id"), Required: true},
				{Name: "pipeline-json", Usage: tr.T("flag.pipeline.pipeline_json"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.pipeline.dry_run_2"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := requiredPositiveInt(ctx, "id")
				if err != nil {
					return err
				}
				pipelineJSON, err := ctx.RequireArg("pipeline-json")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"id":            id,
					"pipeline_json": parseJSONValue(pipelineJSON),
				}
				path := pipelineV1RepoPath(ctx) + "/pipelines/save_yaml"
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
						"dry_run":    true,
						"action":     "save_pipeline_yaml",
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
			Name:        "enable",
			Description: tr.T("cmd.pipeline.enable.short"),
			Flags:       workflowStateFlags(tr, true),
			Run: func(ctx *common.RuntimeContext) error {
				return runWorkflowState(ctx, "enable")
			},
		},
		{
			Name:        "disable",
			Description: tr.T("cmd.pipeline.disable.short"),
			Flags:       workflowStateFlags(tr, true),
			Run: func(ctx *common.RuntimeContext) error {
				return runWorkflowState(ctx, "disable")
			},
		},
		{
			Name:        "logs",
			Description: tr.T("cmd.pipeline.logs.short"),
			Flags: []common.Flag{
				{Name: "run-id", Short: "r", Usage: tr.T("flag.pipeline.run_id"), Required: true},
				{Name: "id", Short: "i", Usage: tr.T("flag.pipeline.id"), Required: true},
				{Name: "index", Usage: tr.T("flag.pipeline.index"), Required: true},
				{Name: "job", Short: "j", Usage: tr.T("flag.pipeline.job"), Default: "0"},
				{Name: "cursor", Usage: tr.T("flag.pipeline.cursor")},
				{Name: "step", Usage: tr.T("flag.pipeline.step"), Default: "1"},
				{Name: "expanded", Usage: tr.T("flag.pipeline.expanded"), Bool: true, Default: "true"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				runID, err := ctx.RequireArg("run-id")
				if err != nil {
					return err
				}
				id, err := requiredPositiveInt(ctx, "id")
				if err != nil {
					return err
				}
				index, err := ctx.RequireArg("index")
				if err != nil {
					return err
				}
				job, err := requiredNonNegativeIntWithDefault(ctx, "job", 0)
				if err != nil {
					return err
				}
				step, err := requiredPositiveIntWithDefault(ctx, "step", 1)
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"id":    id,
					"index": index,
					"job":   job,
					"owner": ctx.Owner,
					"repo":  ctx.Repo,
					"log_cursors": []map[string]interface{}{
						{
							"cursor":   ctx.Arg("cursor"),
							"expanded": ctx.Arg("expanded") == "true",
							"step":     step,
						},
					},
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/actions/runs/%s/jobs/0", pipelineV1RepoPath(ctx), url.PathEscape(runID)), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "results",
			Description: tr.T("cmd.pipeline.results.short"),
			Flags: []common.Flag{
				{Name: "run-id", Short: "r", Usage: tr.T("flag.pipeline.run_id")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				setQueryIfPresent(q, ctx, "run-id", "run_id")
				env, err := ctx.CallAPIWithQuery("GET", pipelineV1RepoPath(ctx)+"/pipelines/run_results", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func pipelineV1RepoPath(ctx *common.RuntimeContext) string {
	return "/v1" + ctx.RepoPath()
}

func runFilterFlags(tr *i18n.Translator) []common.Flag {
	return []common.Flag{
		{Name: "ref", Short: "r", Usage: tr.T("flag.pipeline.ref")},
		{Name: "workflow", Short: "w", Usage: tr.T("flag.pipeline.workflow")},
	}
}

func runFilterQuery(ctx *common.RuntimeContext) url.Values {
	q := url.Values{}
	setQueryIfPresent(q, ctx, "ref", "ref")
	setQueryIfPresent(q, ctx, "workflow", "workflow")
	return q
}

func workflowStateFlags(tr *i18n.Translator, includeDryRun bool) []common.Flag {
	flags := []common.Flag{
		{Name: "id", Short: "i", Usage: tr.T("flag.pipeline.id"), Required: true},
		{Name: "workflow", Short: "w", Usage: tr.T("flag.pipeline.workflow"), Required: true},
	}
	if includeDryRun {
		flags = append(flags, common.Flag{Name: "dry-run", Usage: tr.T("flag.pipeline.dry_run_2"), Bool: true, Default: "false"})
	}
	return flags
}

func runWorkflowState(ctx *common.RuntimeContext, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := requiredPositiveInt(ctx, "id")
	if err != nil {
		return err
	}
	workflow, err := ctx.RequireArg("workflow")
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"id":       id,
		"workflow": workflow,
	}
	path := fmt.Sprintf("%s/actions/%s", pipelineV1RepoPath(ctx), action)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			"dry_run":    true,
			"action":     action + "_pipeline",
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
}

func pageLimitQuery(ctx *common.RuntimeContext) url.Values {
	q := url.Values{}
	setQueryIfPresent(q, ctx, "page", "page")
	setQueryIfPresent(q, ctx, "limit", "limit")
	return q
}

func setQueryIfPresent(q url.Values, ctx *common.RuntimeContext, flagName, queryName string) {
	if value := ctx.Arg(flagName); value != "" {
		q.Set(queryName, value)
	}
}

func parseJSONValue(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		return parsed
	}
	return value
}

func requiredPositiveInt(ctx *common.RuntimeContext, flagName string) (int, error) {
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

func requiredPositiveIntWithDefault(ctx *common.RuntimeContext, flagName string, defaultValue int) (int, error) {
	value := ctx.Arg(flagName)
	if value == "" {
		return defaultValue, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", flagName)
	}
	return id, nil
}

func requiredNonNegativeIntWithDefault(ctx *common.RuntimeContext, flagName string, defaultValue int) (int, error) {
	value := ctx.Arg(flagName)
	if value == "" {
		return defaultValue, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id < 0 {
		return 0, fmt.Errorf("--%s must be a non-negative integer", flagName)
	}
	return id, nil
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
