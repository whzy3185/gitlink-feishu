package repomirror

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns mirror repository operations.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "sync",
			Description: "Synchronize a mirror repository",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Repository/project ID", Required: true},
				{Name: "dry-run", Usage: "Preview the request without syncing the mirror", Bool: true, Default: "false"},
			},
			Run: runSync,
		},
	}
}

func runSync(ctx *common.RuntimeContext) error {
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	if _, err := strconv.Atoi(id); err != nil {
		return fmt.Errorf("invalid --id value %q: %w", id, err)
	}
	path := fmt.Sprintf("/repositories/%s/sync_mirror", url.PathEscape(id))
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{"dry_run": true, "action": "sync_mirror", "method": "POST", "path": path})
	}
	env, err := ctx.CallAPI("POST", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}
