// Package dataset implements shortcuts for querying GitLink research datasets.
//
// GitLink exposes dataset metadata (title, description, paper_content, license,
// owning project) through the platform-wide query endpoint
// GET /api/v1/project_datasets. The per-repository dataset CRUD routes
// documented under /api/v1/{owner}/{repo}/dataset are not deployed on the
// production gitlink.org.cn host, so this package wraps the query endpoint and
// resolves a repository's project ID to offer both a list and a per-repo view.
package dataset

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns dataset query shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)

	return []*common.Shortcut{
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
				return queryDatasets(ctx, normalized)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.dataset.view.short"),
			Long:        tr.T("cmd.dataset.view.long"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.dataset.project_id")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				projectID := strings.TrimSpace(ctx.Arg("project-id"))
				if projectID == "" {
					if err := ctx.ResolveOwnerRepo(); err != nil {
						return err
					}
					resolved, err := resolveProjectID(ctx)
					if err != nil {
						return err
					}
					projectID = strconv.FormatInt(resolved, 10)
				} else if _, err := strconv.ParseInt(projectID, 10, 64); err != nil {
					return fmt.Errorf("invalid --project-id %q: use a numeric project ID", projectID)
				}
				return queryDatasets(ctx, projectID)
			},
		},
	}
}

// queryDatasets calls the platform dataset query endpoint with a comma-separated
// list of project IDs.
func queryDatasets(ctx *common.RuntimeContext, ids string) error {
	q := url.Values{}
	q.Set("ids", ids)
	env, err := ctx.CallAPIWithQuery("GET", "/v1/project_datasets", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// normalizeIDs validates a comma-separated list of positive integer project IDs
// and returns it without surrounding whitespace.
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

// resolveProjectID resolves the numeric GitLink project ID from the current
// owner/repo via the repository info endpoint.
func resolveProjectID(ctx *common.RuntimeContext) (int64, error) {
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"id", "project_id"} {
		if id, ok := projectIDValue(data[key]); ok {
			return id, nil
		}
	}
	return 0, fmt.Errorf("resolve project id: repository response did not include id")
}

func projectIDValue(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return int64(v), true
		}
	case int:
		if v > 0 {
			return int64(v), true
		}
	case int64:
		if v > 0 {
			return v, true
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
