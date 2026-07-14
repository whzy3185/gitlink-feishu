package webhook

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for webhook management.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.webhook.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", v1Path(ctx)+"/webhooks", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.webhook.create.short"),
			Flags: []common.Flag{
				{Name: "url", Short: "u", Usage: tr.T("flag.webhook.url"), Required: true},
				{Name: "content-type", Usage: tr.T("flag.webhook.content_type"), Default: "json"},
				{Name: "secret", Short: "s", Usage: tr.T("flag.webhook.secret")},
				{Name: "events", Short: "e", Usage: tr.T("flag.webhook.events")},
				{Name: "branch-filter", Usage: tr.T("flag.webhook.branch_filter")},
				{Name: "active", Usage: tr.T("flag.webhook.active"), Default: "true"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				webhookURL, err := ctx.RequireArg("url")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"url":          webhookURL,
					"content_type": ctx.Arg("content-type"),
					"http_method":  "POST",
					"active":       true,
				}
				if secret := ctx.Arg("secret"); secret != "" {
					body["secret"] = secret
				}
				if events := ctx.Arg("events"); events != "" {
					body["events"] = strings.Split(events, ",")
				} else {
					body["events"] = []string{"push"}
				}
				if bf := ctx.Arg("branch-filter"); bf != "" {
					body["branch_filter"] = bf
				}
				env, err := ctx.CallAPI("POST", v1Path(ctx)+"/webhooks", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.webhook.view.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/webhooks/%s", v1Path(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: tr.T("cmd.webhook.update.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
				{Name: "url", Short: "u", Usage: tr.T("flag.webhook.url")},
				{Name: "content-type", Usage: tr.T("flag.webhook.content_type"), Default: "json"},
				{Name: "secret", Short: "s", Usage: tr.T("flag.webhook.secret")},
				{Name: "events", Short: "e", Usage: tr.T("flag.webhook.events")},
				{Name: "branch-filter", Usage: tr.T("flag.webhook.branch_filter")},
				{Name: "active", Usage: tr.T("flag.webhook.active"), Default: "true"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"content_type":  ctx.Arg("content-type"),
					"http_method":   "POST",
					"active":        true,
					"branch_filter": ctx.Arg("branch-filter"),
					"secret":        ctx.Arg("secret"),
				}
				if webhookURL := ctx.Arg("url"); webhookURL != "" {
					body["url"] = webhookURL
				}
				if events := ctx.Arg("events"); events != "" {
					body["events"] = strings.Split(events, ",")
				} else {
					body["events"] = []string{"push"}
				}
				env, err := ctx.CallAPI("PUT", fmt.Sprintf("%s/webhooks/%s", v1Path(ctx), id), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "history",
			Description: tr.T("cmd.webhook.tasks.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("%s/webhooks/%s/hooktasks", v1Path(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "test",
			Description: tr.T("cmd.webhook.test.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("%s/webhooks/%s/tests", v1Path(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.webhook.delete.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.webhook.id"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("%s/webhooks/%s", v1Path(ctx), id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func v1Path(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}
