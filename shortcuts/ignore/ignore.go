package ignore

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns ignore-file management shortcuts.
//
// These shortcuts provide access to the GitLink ignore-file registry,
// which lists all available .gitignore templates supported by the platform.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List available ignore-file templates",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Filter ignore templates by name"},
			},
			Run: runList,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	q := url.Values{}
	if name := ctx.Arg("name"); name != "" {
		q.Set("name", name)
	}
	env, err := ctx.CallAPIWithQuery("GET", "/ignores", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}
