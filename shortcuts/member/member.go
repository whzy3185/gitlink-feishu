package member

import (
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for project member management.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List project members",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/collaborators", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "add",
			Description: "Add a project member",
			Flags: []common.Flag{
				{Name: "user-id", Short: "u", Usage: "User ID to add", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				userID, err := ctx.RequireArg("user-id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"user_id": userID,
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/collaborators", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "remove",
			Description: "Remove a project member",
			Flags: []common.Flag{
				{Name: "user-id", Short: "u", Usage: "User ID to remove", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				userID, err := ctx.RequireArg("user-id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"user_id": userID,
				}
				env, err := ctx.CallAPI("DELETE", ctx.RepoPath()+"/collaborators/remove", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
