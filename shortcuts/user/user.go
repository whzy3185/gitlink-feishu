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
			Name:        "heatmap",
			Description: "Show user contribution heatmap",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "year", Short: "y", Usage: "Year (e.g. 2026)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/users/%s/headmaps", login)
				if year := ctx.Arg("year"); year != "" {
					q := url.Values{}
					q.Set("year", year)
					env, err := ctx.CallAPIWithQuery("GET", path, q)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				}
				env, err := ctx.CallAPI("GET", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "stats",
			Description: "Show user development statistics",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "start-time", Usage: "Start date (YYYY-MM-DD)"},
				{Name: "end-time", Usage: "End date (YYYY-MM-DD)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/users/%s/statistics/develop", login)
				q := url.Values{}
				if st := ctx.Arg("start-time"); st != "" {
					q.Set("start_time", st)
				}
				if et := ctx.Arg("end-time"); et != "" {
					q.Set("end_time", et)
				}
				if len(q) > 0 {
					env, err := ctx.CallAPIWithQuery("GET", path, q)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				}
				env, err := ctx.CallAPI("GET", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "trends",
			Description: "Show user project trends",
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: "User login name", Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				query := url.Values{}
				query.Set("page", firstValue(ctx.Arg("page"), "1"))
				query.Set("limit", firstValue(ctx.Arg("limit"), "20"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/project_trends", login), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func newStatsShortcut(tr *i18n.Translator, name, description, subPath string) *common.Shortcut {
	return &common.Shortcut{
		Name:        name,
		Description: description,
		Flags: []common.Flag{
			{Name: "login", Short: "l", Usage: tr.T("flag.user.login"), Required: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			login, err := ctx.RequireArg("login")
			if err != nil {
				return err
			}
			env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/statistics/%s", login, subPath), nil)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func firstValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
