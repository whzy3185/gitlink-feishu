package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// switchToGateway overrides the client base URL with the gateway URL from config.
func switchToGateway(ctx *common.RuntimeContext) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = config.DefaultGatewayURL
	}
	ctx.Client.BaseURL = cfg.GatewayURL
	return nil
}

// gatewayFlag returns the common --gateway flag definition.
func gatewayFlag() common.Flag {
	return common.Flag{Name: "gateway", Short: "g", Usage: "Use gateway API endpoint", Bool: true}
}

// Shortcuts returns all wiki shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List wiki pages",
			Flags: []common.Flag{
				{Name: "project-id", Usage: "GitLink project ID", Required: true},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", ctx.Arg("project-id"))
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/wikiPages", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View a wiki page by page name",
			Flags: []common.Flag{
				{Name: "project-id", Usage: "GitLink project ID", Required: true},
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", ctx.Arg("project-id"))
				q.Set("pageName", ctx.Arg("page-name"))
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/getWiki", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a new wiki page",
			Flags: []common.Flag{
				{Name: "project-id", Usage: "GitLink project ID", Required: true},
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
				{Name: "content", Short: "c", Usage: "Wiki page content (markdown)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				content := ctx.Arg("content")
				payload := map[string]interface{}{
					"owner":          ctx.Owner,
					"repo":           ctx.Repo,
					"projectId":      ctx.Arg("project-id"),
					"pageName":       ctx.Arg("page-name"),
					"title":          ctx.Arg("title"),
					"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
					"message":        ctx.Arg("message"),
				}
				env, err := ctx.CallAPI("POST", "/wiki/open/createWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update an existing wiki page",
			Flags: []common.Flag{
				{Name: "project-id", Usage: "GitLink project ID", Required: true},
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
				{Name: "content", Short: "c", Usage: "Wiki page content (markdown)"},
				{Name: "message", Short: "m", Usage: "Commit message"},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title := ctx.Arg("title")
				if title == "" {
					return fmt.Errorf("--title is required")
				}
				content := ctx.Arg("content")
				payload := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": ctx.Arg("project-id"),
					"pageName":  ctx.Arg("page-name"),
					"title":     title,
					"message":   ctx.Arg("message"),
				}
				if content != "" {
					payload["content_base64"] = base64.StdEncoding.EncodeToString([]byte(content))
				}
				env, err := ctx.CallAPI("PUT", "/wiki/open/updateWiki", payload)
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
				{Name: "project-id", Usage: "GitLink project ID", Required: true},
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				payload := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": ctx.Arg("project-id"),
					"pageName":  ctx.Arg("page-name"),
				}
				env, err := ctx.CallAPI("DELETE", "/wiki/open/deleteWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
