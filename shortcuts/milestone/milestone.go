package milestone

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List milestones",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
				{Name: "category", Short: "c", Usage: "Filter by category: opening, closed"},
				{Name: "only-name", Usage: "Return only milestone id and name: true or false"},
				{Name: "sort-by", Usage: "Sort field: created_on, updated_on, effective_date, issues_count, percent"},
				{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
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
			Description: "Create a milestone",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Milestone name", Required: true},
				{Name: "description", Short: "d", Usage: "Milestone description", Required: true},
				{Name: "due-date", Usage: "Due date in YYYY-MM-DD format", Required: true},
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
			Description: "View milestone details and linked issues",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Milestone ID", Required: true},
				{Name: "category", Short: "c", Usage: "Filter issues by category: all, opened, closed"},
				{Name: "author-id", Usage: "Filter issues by author ID"},
				{Name: "assigner-id", Usage: "Filter issues by assignee ID"},
				{Name: "issue-tag-ids", Usage: "Comma-separated issue tag IDs"},
				{Name: "sort-by", Usage: "Sort field: issues.created_on, issues.updated_on, issue_priorities.position"},
				{Name: "sort-direction", Usage: "Sort direction: asc or desc"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
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
			Description: "Update a milestone",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Milestone ID", Required: true},
				{Name: "name", Short: "n", Usage: "Milestone name"},
				{Name: "description", Short: "d", Usage: "Milestone description"},
				{Name: "due-date", Usage: "Due date in YYYY-MM-DD format"},
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
			Description: "Delete a milestone",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Milestone ID", Required: true},
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
		newStatusShortcut("close", "Close a milestone", "closed"),
		newStatusShortcut("reopen", "Reopen a milestone", "open"),
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

func newStatusShortcut(name, description, status string) *common.Shortcut {
	return &common.Shortcut{
		Name:        name,
		Description: description,
		Flags: []common.Flag{
			{Name: "id", Short: "i", Usage: "Milestone ID", Required: true},
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
