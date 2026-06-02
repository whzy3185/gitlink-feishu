package wiki

import (
	"encoding/base64"
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
		{
			Name:        "view",
			Description: "View a wiki page",
			Flags: []common.Flag{
				{Name: "slug", Short: "s", Usage: "Wiki page slug", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				slug, err := ctx.RequireArg("slug")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", wikiPagePath(ctx, slug), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a wiki page",
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
				{Name: "body", Short: "b", Usage: "Wiki page content", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title, err := ctx.RequireArg("title")
				if err != nil {
					return err
				}
				body, err := ctx.RequireArg("body")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"title":           title,
					"content_base64":  base64.StdEncoding.EncodeToString([]byte(body)),
				}
				env, err := ctx.CallAPI("POST", wikiPagesPath(ctx), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update a wiki page",
			Flags: []common.Flag{
				{Name: "slug", Short: "s", Usage: "Wiki page slug", Required: true},
				{Name: "title", Short: "t", Usage: "New title"},
				{Name: "body", Short: "b", Usage: "New content"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				slug, err := ctx.RequireArg("slug")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{}
				if title := ctx.Arg("title"); title != "" {
					payload["title"] = title
				}
				if body := ctx.Arg("body"); body != "" {
					payload["content_base64"] = base64.StdEncoding.EncodeToString([]byte(body))
				}
				env, err := ctx.CallAPI("PATCH", wikiPagePath(ctx, slug), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a wiki page",
			Flags: []common.Flag{
				{Name: "slug", Short: "s", Usage: "Wiki page slug", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				slug, err := ctx.RequireArg("slug")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", wikiPagePath(ctx, slug), nil)
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

func wikiPagePath(ctx *common.RuntimeContext, slug string) string {
	return fmt.Sprintf("/%s/%s/wiki/pages/%s", ctx.Owner, ctx.Repo, slug)
}
