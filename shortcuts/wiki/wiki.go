package wiki

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns wiki page management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List wiki pages",
			Flags: []common.Flag{
				{Name: "project-id", Short: "p", Usage: "GitLink project ID (required)", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := ctx.RequireArg("project-id")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", projectID)
				env, err := ctx.CallAPIWithQuery("GET", "/api/wiki/wikiPages", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
