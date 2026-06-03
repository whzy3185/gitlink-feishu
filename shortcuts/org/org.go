package org

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.org.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", "/organizations", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.org.info.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id_or_login"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/organizations/%s", id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "members",
			Description: tr.T("cmd.org.members.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/organizations/%s/organization_users", id), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.org.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.org.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.description")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, _ := ctx.RequireArg("name")
				payload := map[string]interface{}{
					"name": name,
				}
				if d := ctx.Arg("description"); d != "" {
					payload["description"] = d
				}
				env, err := ctx.CallAPI("POST", "/organizations", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
			{
				Name:        "teams",
				Description: "列出组织下的所有团队",
				Flags: []common.Flag{
					{Name: "id", Usage: "组织 ID", Required: true},
				},
				Run: func(ctx *common.RuntimeContext) error {
					id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
					env, err := ctx.CallAPI("GET", fmt.Sprintf("/organizations/%s/teams", id), nil)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				},
			},
			{
				Name:        "create-team",
				Description: "在组织下创建新团队",
				Flags: []common.Flag{
					{Name: "id", Usage: "组织 ID", Required: true},
					{Name: "name", Short: "n", Usage: "团队名称", Required: true},
				},
				Run: func(ctx *common.RuntimeContext) error {
					id, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
					name, err := ctx.RequireArg("name")
					if err != nil {
						return err
					}
					body := map[string]interface{}{"name": name}
					env, err := ctx.CallAPI("POST", fmt.Sprintf("/organizations/%s/teams", id), body)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				},
			},
			{
				Name:        "remove-user",
				Description: "从组织中移除成员",
				Flags: []common.Flag{
					{Name: "id", Usage: "组织 ID", Required: true},
					{Name: "user", Short: "u", Usage: "要移除的用户 ID", Required: true},
				},
				Run: func(ctx *common.RuntimeContext) error {
					orgID, err := ctx.RequireArg("id")
					if err != nil {
						return err
					}
					userID, err := ctx.RequireArg("user")
					if err != nil {
						return err
					}
					path := fmt.Sprintf("/organizations/%s/organization_users/%s", orgID, userID)
					env, err := ctx.CallAPI("DELETE", path, nil)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				},
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
