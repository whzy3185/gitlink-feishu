package commit

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns repository commit inspection shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository commits",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Branch, tag, or commit SHA"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				setQuery(q, "sha", ctx.Arg("sha"))
				setQuery(q, "page", ctx.Arg("page"))
				setQuery(q, "limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", commitRepoPath(ctx)+"/commits", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: "List changed files for a commit",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Commit SHA", Required: true},
				{Name: "filepath", Short: "f", Usage: "Filter by file path"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				q := url.Values{}
				setQuery(q, "filepath", ctx.Arg("filepath"))
				setQuery(q, "page", ctx.Arg("page"))
				setQuery(q, "limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("%s/commits/%s/files", commitRepoPath(ctx), url.PathEscape(sha)), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "diff",
			Description: "Show diff for a commit",
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
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/commits/%s/diff", commitRepoPath(ctx), url.PathEscape(sha)), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "blame",
			Description: "Show blame information for a file",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Branch, tag, or commit SHA", Required: true},
				{Name: "filepath", Short: "f", Usage: "File path", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				filepath, err := ctx.RequireArg("filepath")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("sha", sha)
				q.Set("filepath", filepath)
				env, err := ctx.CallAPIWithQuery("GET", commitRepoPath(ctx)+"/blame", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func commitRepoPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}

func setQuery(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}
