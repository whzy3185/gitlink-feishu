package label

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for label management.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List issue labels (tags)",
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Search keyword"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
				{Name: "order-by", Usage: "Sort field: updated_on, created_on, issues_count", Default: "created_on"},
				{Name: "order-direction", Usage: "Sort direction: asc, desc", Default: "desc"},
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
				if o := ctx.Arg("order-by"); o != "" {
					q.Set("order_by", o)
				}
				if d := ctx.Arg("order-direction"); d != "" {
					q.Set("order_direction", d)
				}
				env, err := ctx.CallAPIWithQuery("GET", v1Path(ctx)+"/issue_tags", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create an issue label (tag)",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Label name", Required: true},
				{Name: "color", Short: "c", Usage: "Color hex (e.g. #FF0000)"},
				{Name: "description", Short: "d", Usage: "Label description"},
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
				if c := ctx.Arg("color"); c != "" {
					body["color"] = c
				}
				if d := ctx.Arg("description"); d != "" {
					body["description"] = d
				}
				env, err := ctx.CallAPI("POST", v1Path(ctx)+"/issue_tags", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update an issue label (tag)",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Label ID", Required: true},
				{Name: "name", Short: "n", Usage: "New label name"},
				{Name: "color", Short: "c", Usage: "New color hex (e.g. #FF0000)"},
				{Name: "description", Short: "d", Usage: "New description"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{}
				if n := ctx.Arg("name"); n != "" {
					payload["name"] = n
				}
				if c := ctx.Arg("color"); c != "" {
					payload["color"] = c
				}
				if d := ctx.Arg("description"); d != "" {
					payload["description"] = d
				}
				if len(payload) == 0 {
					return fmt.Errorf("至少需要指定 --name, --color 或 --description 之一")
				}
				env, err := ctx.CallAPI("PATCH",
					fmt.Sprintf("%s/issue_tags/%s", v1Path(ctx), id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete an issue label (tag)",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Label ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/issue_tags/%s", v1Path(ctx), id), nil)
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
