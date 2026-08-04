package notification

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{Name: "list", Run: func(ctx *common.RuntimeContext) error {
			query := url.Values{}
			if value := ctx.Arg("page"); value != "" {
				query.Set("page", value)
			}
			if value := ctx.Arg("limit"); value != "" {
				query.Set("limit", value)
			}
			env, err := ctx.CallAPIWithQuery("GET", messagesPath(ctx), query)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		}},
		{Name: "view", Flags: []common.Flag{{Name: "id", Required: true}}, Run: func(ctx *common.RuntimeContext) error {
			id, err := ctx.RequireArg("id")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("GET", messagesPath(ctx)+"/"+url.PathEscape(id), nil)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		}},
		{Name: "read", Flags: []common.Flag{{Name: "id", Required: true}}, Run: func(ctx *common.RuntimeContext) error {
			id, err := ctx.RequireArg("id")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("POST", messagesPath(ctx)+"/read", map[string]interface{}{"id": id})
			if err != nil {
				return err
			}
			return ctx.Output(env)
		}},
		{Name: "delete", Flags: []common.Flag{{Name: "id", Required: true}}, Run: func(ctx *common.RuntimeContext) error {
			id, err := ctx.RequireArg("id")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("DELETE", messagesPath(ctx), map[string]interface{}{"id": id})
			if err != nil {
				return err
			}
			return ctx.Output(env)
		}},
	}
}

func messagesPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/users/%s/messages", ctx.Owner)
}
