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
				{Name: "all", Usage: tr.T("flag.all"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey("/organizations", q, "organizations")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("organizations", items))
				}
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
				{Name: "all", Usage: tr.T("flag.all"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey(fmt.Sprintf("/organizations/%s/organization_users", id), q, "organization_users")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("organization_users", items))
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/organizations/%s/organization_users", id), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "repos",
			Description: tr.T("cmd.org.repos.short"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
				{Name: "all", Usage: tr.T("flag.all"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				path := fmt.Sprintf("/organizations/%s/projects", id)
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey(path, q, "projects")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("projects", items))
				}
				env, err := ctx.CallAPIWithQuery("GET", path, q)
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
					"name":       name,
					"nickname":   name,
					"visibility": "common",
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
		{
			Name:        "teams",
			Description: "List teams in an organization",
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
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/organizations/%s/teams", id), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create-team",
			Description: "Create a team in an organization",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "name", Short: "n", Usage: tr.T("flag.org.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.description")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				name, _ := ctx.RequireArg("name")
				payload := map[string]interface{}{
					"name":     name,
					"nickname": name,
				}
				if d := ctx.Arg("description"); d != "" {
					payload["description"] = d
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("/organizations/%s/teams", id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "remove-member",
			Description: "Remove a member from an organization",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "uid", Short: "u", Usage: "User ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				uid, _ := ctx.RequireArg("uid")
				env, err := ctx.CallAPI("DELETE", fmt.Sprintf("/organizations/%s/organization_users/%s", id, uid), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "nickname",
			Description: "Set or view a member's nickname in an organization",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.org.id"), Required: true},
				{Name: "uid", Short: "u", Usage: "User ID", Required: true},
				{Name: "nickname", Short: "n", Usage: "New nickname (omit to view current)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, _ := ctx.RequireArg("id")
				uid, _ := ctx.RequireArg("uid")
				nickname := ctx.Arg("nickname")
				if nickname != "" {
					payload := map[string]interface{}{
						"nickname": nickname,
					}
					env, err := ctx.CallAPI("PUT", fmt.Sprintf("/organizations/%s/organization_users/%s", id, uid), payload)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/organizations/%s/organization_users/%s", id, uid), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "uid",
			Description: "Look up a user's numeric ID by login name",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s", login), nil)
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
