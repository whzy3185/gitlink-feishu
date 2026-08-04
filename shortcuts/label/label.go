package label

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:  "list",
			Flags: []common.Flag{{Name: "page"}, {Name: "limit"}},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				query := url.Values{}
				if value := ctx.Arg("page"); value != "" {
					query.Set("page", value)
				}
				if value := ctx.Arg("limit"); value != "" {
					query.Set("limit", value)
				}
				env, err := ctx.CallAPIWithQuery("GET", labelsPath(ctx), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:  "create",
			Flags: []common.Flag{{Name: "name", Required: true}, {Name: "color"}, {Name: "description"}},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				body := map[string]interface{}{"name": name}
				if value := ctx.Arg("color"); value != "" {
					body["color"] = value
				}
				if value := ctx.Arg("description"); value != "" {
					body["description"] = value
				}
				env, err := ctx.CallAPI("POST", labelsPath(ctx), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:  "update",
			Flags: []common.Flag{{Name: "id", Required: true}, {Name: "name"}, {Name: "color"}, {Name: "description"}},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{}
				for _, key := range []string{"name", "color", "description"} {
					if value := ctx.Arg(key); value != "" {
						body[key] = value
					}
				}
				if len(body) == 0 {
					return fmt.Errorf("at least one label field is required")
				}
				env, err := ctx.CallAPI("PATCH", labelsPath(ctx)+"/"+url.PathEscape(id), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:  "delete",
			Flags: []common.Flag{{Name: "id", Required: true}},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", labelsPath(ctx)+"/"+url.PathEscape(id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func labelsPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/%s/%s/labels", ctx.Owner, ctx.Repo)
}
