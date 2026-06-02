package wiki

import (
	"fmt"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns wiki page management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List wiki pages",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", wikiPagesPath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func wikiPagesPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/%s/%s/wiki/pages", ctx.Owner, ctx.Repo)
}
