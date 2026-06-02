package label

import (
	"fmt"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository labels",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", labelPath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a repository label",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Label name", Required: true},
				{Name: "color", Short: "c", Usage: "Label color (hex, e.g. #ff0000)", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				color, err := ctx.RequireArg("color")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", labelPath(ctx), map[string]interface{}{
					"name":  name,
					"color": color,
				})
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update a repository label",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Label ID", Required: true},
				{Name: "name", Short: "n", Usage: "New label name"},
				{Name: "color", Short: "c", Usage: "New label color (hex)"},
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
				if v := ctx.Arg("name"); v != "" {
					payload["name"] = v
				}
				if v := ctx.Arg("color"); v != "" {
					payload["color"] = v
				}
				env, err := ctx.CallAPI("PATCH", labelItemPath(ctx, id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a repository label",
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
				env, err := ctx.CallAPI("DELETE", labelItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func labelPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/%s/%s/labels", ctx.Owner, ctx.Repo)
}

func labelItemPath(ctx *common.RuntimeContext, id string) string {
	return fmt.Sprintf("%s/%s", labelPath(ctx), id)
}
