// Package PLACEHOLDER implements shortcuts for PLACEHOLDER management.
//
// To create a new shortcut domain:
// 1. Copy this file to shortcuts/PLACEHOLDER/PLACEHOLDER.go
// 2. Replace all "PLACEHOLDER" with your domain name
// 3. Implement your shortcuts in the Shortcuts() function
// 4. Register in shortcuts/register.go
// 5. Add tests in PLACEHOLDER_test.go
package PLACEHOLDER

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for this domain.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List PLACEHOLDERs",
			Flags: []common.Flag{
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
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/PLACEHOLDERs", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a PLACEHOLDER",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Name", Required: true},
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
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/PLACEHOLDERs", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a PLACEHOLDER",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "PLACEHOLDER ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/PLACEHOLDERs/%s", ctx.RepoPath(), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
