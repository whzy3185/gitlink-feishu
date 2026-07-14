package user

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
			Name:        "me",
			Description: tr.T("cmd.user.me.short"),
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/users/me", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.user.info.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
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
		{
			Name:        "headmaps",
			Description: tr.T("cmd.user.headmaps.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/headmaps", login), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		newStatsShortcut(tr, "stats-activity", tr.T("cmd.user.stats_activity.short"), "activity"),
		newStatsShortcut(tr, "stats-develop", tr.T("cmd.user.stats_develop.short"), "develop"),
		newStatsShortcut(tr, "stats-role", tr.T("cmd.user.stats_role.short"), "role"),
		newStatsShortcut(tr, "stats-major", tr.T("cmd.user.stats_major.short"), "major"),
		{
			Name:        "trends",
			Description: tr.T("cmd.user.trends.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/project_trends", login), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

// newStatsShortcut 生成用户统计类 shortcut，消除 stats-activity/develop/role/major 的重复代码。
func newStatsShortcut(tr *i18n.Translator, name, desc, subPath string) *common.Shortcut {
	return &common.Shortcut{
		Name:        name,
		Description: desc,
		Flags: []common.Flag{
			{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			login, err := ctx.RequireArg("login")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("GET",
				fmt.Sprintf("/users/%s/statistics/%s", login, subPath), nil)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
