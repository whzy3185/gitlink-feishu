package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
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
func gatewayFlag(tr *i18n.Translator) common.Flag {
	return common.Flag{Name: "gateway", Short: "g", Usage: tr.T("flag.wiki.gateway"), Bool: true}
}

// Shortcuts returns all wiki shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.wiki.list.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id"), Required: true},
				gatewayFlag(tr),
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
			Description: tr.T("cmd.wiki.view.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id"), Required: true},
				{Name: "page-name", Short: "n", Usage: tr.T("flag.wiki.page_name"), Required: true},
				gatewayFlag(tr),
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
			Description: tr.T("cmd.wiki.create.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id"), Required: true},
				{Name: "page-name", Short: "n", Usage: tr.T("flag.wiki.page_name"), Required: true},
				{Name: "title", Short: "t", Usage: tr.T("flag.wiki.title"), Required: true},
				{Name: "content", Short: "c", Usage: tr.T("flag.wiki.content"), Required: true},
				{Name: "message", Short: "m", Usage: tr.T("flag.wiki.message")},
				gatewayFlag(tr),
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
			Description: tr.T("cmd.wiki.update.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id"), Required: true},
				{Name: "page-name", Short: "n", Usage: tr.T("flag.wiki.page_name"), Required: true},
				{Name: "title", Short: "t", Usage: tr.T("flag.wiki.title"), Required: true},
				{Name: "content", Short: "c", Usage: tr.T("flag.wiki.content")},
				{Name: "message", Short: "m", Usage: tr.T("flag.wiki.message")},
				gatewayFlag(tr),
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
			Description: tr.T("cmd.wiki.delete.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id"), Required: true},
				{Name: "page-name", Short: "n", Usage: tr.T("flag.wiki.page_name"), Required: true},
				gatewayFlag(tr),
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

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
