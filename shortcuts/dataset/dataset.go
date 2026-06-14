// Package dataset implements shortcuts for managing and querying GitLink
// research datasets: per-repository dataset detail/create/update, the
// platform-wide dataset query, and dataset attachment deletion.
//
// Datasets carry research-oriented metadata (title, description, paper_content,
// license, owning project) that is valuable for research/scientometric
// scenarios. Per-repository CRUD wraps /api/v1/{owner}/{repo}/dataset; the
// platform query wraps /api/v1/project_datasets.
package dataset

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns dataset management and query shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)

	writeFlags := []common.Flag{
		{Name: "title", Short: "t", Usage: tr.T("flag.dataset.title"), Required: true},
		{Name: "description", Short: "d", Usage: tr.T("flag.dataset.description"), Required: true},
		{Name: "license-id", Usage: tr.T("flag.dataset.license_id")},
		{Name: "paper-content", Usage: tr.T("flag.dataset.paper_content")},
		{Name: "dry-run", Usage: tr.T("flag.dataset.dry_run"), Bool: true, Default: "false"},
	}

	return []*common.Shortcut{
		{
			Name:        "view",
			Description: tr.T("cmd.dataset.view.short"),
			Long:        tr.T("cmd.dataset.view.long"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.dataset.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.dataset.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				setIfPresent(q, "page", ctx.Arg("page"))
				setIfPresent(q, "limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", repoDatasetPath(ctx), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "list",
			Description: tr.T("cmd.dataset.list.short"),
			Long:        tr.T("cmd.dataset.list.long"),
			Flags: []common.Flag{
				{Name: "ids", Usage: tr.T("flag.dataset.ids"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				ids, err := ctx.RequireArg("ids")
				if err != nil {
					return err
				}
				normalized, err := normalizeIDs(ids)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("ids", normalized)
				env, err := ctx.CallAPIWithQuery("GET", "/v1/project_datasets", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.dataset.create.short"),
			Long:        tr.T("cmd.dataset.create.long"),
			Flags:       writeFlags,
			Run:         runWrite("POST", "create_dataset"),
		},
		{
			Name:        "update",
			Description: tr.T("cmd.dataset.update.short"),
			Long:        tr.T("cmd.dataset.update.long"),
			Flags:       writeFlags,
			Run:         runWrite("PUT", "update_dataset"),
		},
		{
			Name:        "delete-attachment",
			Description: tr.T("cmd.dataset.delete_attachment.short"),
			Long:        tr.T("cmd.dataset.delete_attachment.long"),
			Flags: []common.Flag{
				{Name: "uuid", Short: "u", Usage: tr.T("flag.dataset.uuid"), Required: true},
				{Name: "dry-run", Usage: tr.T("flag.dataset.dry_run_delete"), Bool: true, Default: "false"},
				{Name: "yes", Usage: tr.T("flag.dataset.yes"), Bool: true, Default: "false"},
			},
			Run: runDeleteAttachment,
		},
	}
}

// runWrite builds the create/update handlers, which share the same request body.
func runWrite(method, action string) func(ctx *common.RuntimeContext) error {
	return func(ctx *common.RuntimeContext) error {
		if err := ctx.ResolveOwnerRepo(); err != nil {
			return err
		}
		body, err := datasetBody(ctx)
		if err != nil {
			return err
		}
		path := repoDatasetPath(ctx)
		if ctx.Arg("dry-run") == "true" {
			return ctx.OutputData(map[string]interface{}{
				"dry_run": true, "action": action, "method": method, "path": path, "body": body,
			})
		}
		env, err := ctx.CallAPI(method, path, body)
		if err != nil {
			return err
		}
		return ctx.Output(env)
	}
}

func runDeleteAttachment(ctx *common.RuntimeContext) error {
	uuid, err := ctx.RequireArg("uuid")
	if err != nil {
		return err
	}
	uuid = strings.TrimSpace(uuid)
	path := fmt.Sprintf("/attachments/%s", uuid)
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run": true, "action": "delete_dataset_attachment", "method": "DELETE", "path": path, "uuid": uuid,
		})
	}
	if ctx.Arg("yes") != "true" {
		return fmt.Errorf("%s", ctx.Tr.T("error.dataset.delete_confirm"))
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// datasetBody builds the create/update request body and validates inputs.
func datasetBody(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	title, err := ctx.RequireArg("title")
	if err != nil {
		return nil, err
	}
	description, err := ctx.RequireArg("description")
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"title":       strings.TrimSpace(title),
		"description": strings.TrimSpace(description),
	}
	if v := strings.TrimSpace(ctx.Arg("license-id")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid --license-id %q: use a positive integer", v)
		}
		body["license_id"] = n
	}
	if v := strings.TrimSpace(ctx.Arg("paper-content")); v != "" {
		body["paper_content"] = v
	}
	return body, nil
}

func repoDatasetPath(ctx *common.RuntimeContext) string {
	return "/v1" + ctx.RepoPath() + "/dataset"
}

// normalizeIDs validates a comma-separated list of positive integer project IDs.
func normalizeIDs(raw string) (string, error) {
	parts := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil || n <= 0 {
			return "", fmt.Errorf("invalid --ids %q: use comma-separated positive project IDs", raw)
		}
		cleaned = append(cleaned, strconv.FormatInt(n, 10))
	}
	if len(cleaned) == 0 {
		return "", fmt.Errorf("invalid --ids %q: provide at least one project ID", raw)
	}
	return strings.Join(cleaned, ","), nil
}

func setIfPresent(q url.Values, key, value string) {
	if v := strings.TrimSpace(value); v != "" {
		q.Set(key, v)
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
