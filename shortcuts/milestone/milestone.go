package milestone

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.milestone.list.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword")},
				{Name: "category", Short: "c", Usage: tr.T("flag.milestone.category")},
				{Name: "only-name", Usage: tr.T("flag.milestone.only_name")},
				{Name: "sort-by", Usage: tr.T("flag.milestone.sort_by")},
				{Name: "sort-direction", Usage: tr.T("flag.milestone.sort_direction")},
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
				setQueryIfPresent(q, "keyword", ctx.Arg("keyword"))
				setQueryIfPresent(q, "category", ctx.Arg("category"))
				setQueryIfPresent(q, "only_name", ctx.Arg("only-name"))
				setQueryIfPresent(q, "sort_by", ctx.Arg("sort-by"))
				setQueryIfPresent(q, "sort_direction", ctx.Arg("sort-direction"))
				env, err := ctx.CallAPIWithQuery("GET", milestonePath(ctx), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.milestone.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.milestone.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.milestone.description"), Required: true},
				{Name: "due-date", Usage: tr.T("flag.milestone.due_date"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				payload, err := milestonePayload(ctx, true)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", milestonePath(ctx), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.milestone.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.milestone.id"), Required: true},
				{Name: "category", Short: "c", Usage: tr.T("flag.milestone.category_2")},
				{Name: "author-id", Usage: tr.T("flag.milestone.author_id")},
				{Name: "assigner-id", Usage: tr.T("flag.milestone.assigner_id")},
				{Name: "issue-tag-ids", Usage: tr.T("flag.milestone.issue_tag_ids")},
				{Name: "sort-by", Usage: tr.T("flag.milestone.sort_by_2")},
				{Name: "sort-direction", Usage: tr.T("flag.milestone.sort_direction")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				setQueryIfPresent(q, "category", ctx.Arg("category"))
				setQueryIfPresent(q, "author_id", ctx.Arg("author-id"))
				setQueryIfPresent(q, "assigner_id", ctx.Arg("assigner-id"))
				setQueryIfPresent(q, "issue_tag_ids", normalizeCSV(ctx.Arg("issue-tag-ids")))
				setQueryIfPresent(q, "sort_by", ctx.Arg("sort-by"))
				setQueryIfPresent(q, "sort_direction", ctx.Arg("sort-direction"))
				env, err := ctx.CallAPIWithQuery("GET", milestoneItemPath(ctx, id), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: tr.T("cmd.milestone.update.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.milestone.id"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.milestone.name")},
				{Name: "description", Short: "d", Usage: tr.T("flag.milestone.description")},
				{Name: "due-date", Usage: tr.T("flag.milestone.due_date")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				payload, err := milestonePayload(ctx, false)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PATCH", milestoneItemPath(ctx, id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.milestone.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.milestone.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", milestoneItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		newStatusShortcut(tr, "close", tr.T("cmd.milestone.close.short"), "closed"),
		newStatusShortcut(tr, "reopen", tr.T("cmd.milestone.reopen.short"), "open"),
	}
}

func milestonePath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/milestones", ctx.Owner, ctx.Repo)
}

func milestoneItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", milestonePath(ctx), url.PathEscape(id))
}

func milestoneStatusPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/milestones/%s/update_status", ctx.RepoPath(), url.PathEscape(id))
}

func milestonePayload(ctx *common.RuntimeContext, requireAll bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{}
	if name := ctx.Arg("name"); name != "" {
		payload["name"] = name
	}
	if description := ctx.Arg("description"); description != "" {
		payload["description"] = description
	}
	if dueDate := ctx.Arg("due-date"); dueDate != "" {
		payload["effective_date"] = dueDate
	}

	if requireAll {
		for _, name := range []string{"name", "description", "due-date"} {
			if _, err := ctx.RequireArg(name); err != nil {
				return nil, err
			}
		}
		return payload, nil
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("at least one of --name, --description, or --due-date is required")
	}
	return payload, nil
}

func newStatusShortcut(tr *i18n.Translator, name, description, status string) *common.Shortcut {
	return &common.Shortcut{
		Name:        name,
		Description: description,
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: tr.T("flag.milestone.id"), Required: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			id, err := ctx.RequireArg("id")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("POST", milestoneStatusPath(ctx, id), map[string]interface{}{
				"status": status,
			})
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func setQueryIfPresent(q url.Values, name, value string) {
	if value != "" {
		q.Set(name, value)
	}
}

func normalizeCSV(value string) string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return strings.Join(result, ",")
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
