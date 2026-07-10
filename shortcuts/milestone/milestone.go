package milestone

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for milestone management.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List milestones",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
				{Name: "status", Short: "s", Usage: "Filter by status: open, closed, all", Default: "all"},
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
				if k := ctx.Arg("keyword"); k != "" {
					q.Set("keyword", k)
				}
				if s := ctx.Arg("status"); s != "all" {
					q.Set("status", s)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1Path(ctx)+"/milestones", q)
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
				{Name: "description", Short: "d", Usage: "Milestone description"},
				{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"name": name,
				}
				if desc := ctx.Arg("description"); desc != "" {
					body["description"] = desc
				}
				if due := ctx.Arg("due-date"); due != "" {
					body["effective_date"] = due
				}
				env, err := ctx.CallAPI("POST", v1Path(ctx)+"/milestones", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View milestone details with associated issues",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Milestone ID", Required: true},
				{Name: "category", Short: "c", Usage: "Issue filter: all, opened, closed", Default: "all"},
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
				if c := ctx.Arg("category"); c != "all" {
					q.Set("category", c)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("%s/milestones/%s", v1Path(ctx), id), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "close",
			Description: "Close a milestone",
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
				body := map[string]interface{}{
					"status": "closed",
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/milestones/%s/update_status", v1Path(ctx), id), body)
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
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/milestones/%s", v1Path(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func v1Path(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}
