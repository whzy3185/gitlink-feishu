package compare

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "view",
			Description: "Compare two branches, tags, or commits",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, err := ctx.RequireArg("head")
				if err != nil {
					return err
				}
				base, err := ctx.RequireArg("base")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", comparePath(ctx, head, base), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: "List changed files between two refs",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
				{Name: "file", Short: "f", Usage: "Filter by file path"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, err := ctx.RequireArg("head")
				if err != nil {
					return err
				}
				base, err := ctx.RequireArg("base")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if file := ctx.Arg("file"); file != "" {
					q.Set("filepath", file)
				}
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+comparePath(ctx, head, base)+"/files", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func comparePath(ctx *common.RuntimeContext, head, base string) string {
	return fmt.Sprintf("%s/compare/%s...%s", ctx.RepoPath(), encodeRef(head), encodeRef(base))
}

func encodeRef(ref string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}
