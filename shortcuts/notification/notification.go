package notification

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns notification management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List notifications",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := resolveLogin()
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/messages", login), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View notification details",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Notification ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := resolveLogin()
				if err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/messages/%s", login, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "read",
			Description: "Mark a notification as read",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Notification ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := resolveLogin()
				if err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PATCH", fmt.Sprintf("/users/%s/messages/%s", login, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a notification",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Notification ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := resolveLogin()
				if err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("/users/%s/messages/%s", login, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

// resolveLogin returns the current user login from global flags.
func resolveLogin() (string, error) {
	login := cmdutil.Owner
	if login == "" {
		return "", fmt.Errorf("provide --owner (your login) or set it via config")
	}
	return login, nil
}
