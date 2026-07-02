// Package pm implements GitLink project management shortcuts.
package pm

import (
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns project management read shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		pmListShortcut(tr, "dashboards", tr.T("cmd.pm.dashboards.short"), "/pm/dashboards"),
		pmListShortcut(tr, "sprint-issues", tr.T("cmd.pm.sprint_issues.short"), "/pm/sprint_issues"),
		pmListShortcut(tr, "weekly-issues", tr.T("cmd.pm.weekly_issues.short"), "/pm/weekly_issues"),
		pmListShortcut(tr, "issue-tags", tr.T("cmd.pm.issue_tags.short"), "/pm/issue_tags"),
		pmListShortcut(tr, "pipelines", tr.T("cmd.pm.pipelines.short"), "/pm/pipelines"),
		pmListShortcut(tr, "action-runs", tr.T("cmd.pm.action_runs.short"), "/pm/action_runs"),
	}
}

func pmListShortcut(tr *i18n.Translator, name, description, path string) *common.Shortcut {
	return &common.Shortcut{
		Name:        name,
		Description: description,
		Flags: []common.Flag{
			{Name: "project-id", Short: "P", Usage: tr.T("flag.pm.project_id"), Required: true},
			{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
			{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
		},
		Run: func(ctx *common.RuntimeContext) error {
			projectID, err := ctx.RequireArg("project-id")
			if err != nil {
				return err
			}
			q := pmQuery(ctx, projectID)
			env, err := ctx.CallAPIWithQuery("GET", path, q)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func pmQuery(ctx *common.RuntimeContext, projectID string) url.Values {
	q := url.Values{}
	q.Set("project_id", strings.TrimSpace(projectID))
	q.Set("page", firstNonEmpty(ctx.Arg("page"), "1"))
	q.Set("limit", firstNonEmpty(ctx.Arg("limit"), "20"))
	return q
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
