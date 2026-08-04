package commit

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns commit history shortcuts.
//
// Commit history previously had no first-class command even though the
// platform exposes a paginated v1 endpoint; agents had to fall back to the
// raw api command to read it.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository commits",
			Flags: []common.Flag{
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA to start from (default branch when omitted)"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
				{Name: "all", Usage: "Fetch all pages automatically (ignores --page)", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path := fmt.Sprintf("/v1/%s/%s/commits", ctx.Owner, ctx.Repo)
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ref := ctx.Arg("ref"); ref != "" {
					q.Set("sha", ref)
				}
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey(path, q, "commits")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("commits", items))
				}
				env, err := ctx.CallAPIWithQuery("GET", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View a single commit",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Commit SHA", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/commits/%s", ctx.RepoPath(), sha), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
