package meta

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns read-only metadata lookup shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "licenses",
			Description: "List repository license templates",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Filter license templates by name"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				if name := ctx.Arg("name"); name != "" {
					q.Set("name", name)
				}
				env, err := ctx.CallAPIWithQuery("GET", "/licenses", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "ignores",
			Description: "List .gitignore templates",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Filter ignore templates by name"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				if name := ctx.Arg("name"); name != "" {
					q.Set("name", name)
				}
				env, err := ctx.CallAPIWithQuery("GET", "/ignores", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
