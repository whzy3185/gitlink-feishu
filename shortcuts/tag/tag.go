package tag

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns git tag shortcuts.
//
// Tags previously had no first-class command even though the platform
// exposes a paginated v1 endpoint; releases only cover annotated releases,
// while lightweight tags were reachable through the raw API alone.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository git tags",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
				{Name: "all", Usage: "Fetch all pages automatically (ignores --page)", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path := fmt.Sprintf("/v1/%s/%s/tags", ctx.Owner, ctx.Repo)
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey(path, q, "tags")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("tags", items))
				}
				env, err := ctx.CallAPIWithQuery("GET", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
