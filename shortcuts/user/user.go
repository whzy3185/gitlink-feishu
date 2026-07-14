package user

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	userFlag := common.Flag{Name: "user", Short: "u", Usage: tr.T("flag.user")}
	yearFlag := common.Flag{Name: "year", Usage: tr.T("flag.user.year")}
	timeFlags := []common.Flag{
		{Name: "start-time", Usage: tr.T("flag.user.start_time")},
		{Name: "end-time", Usage: tr.T("flag.user.end_time")},
	}
	windowFlags := append([]common.Flag{userFlag}, timeFlags...)

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
			Name:        "heatmap",
			Description: tr.T("cmd.user.heatmap.short"),
			Flags:       []common.Flag{userFlag, yearFlag},
			Run: func(ctx *common.RuntimeContext) error {
				user, err := resolveUser(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				if v := ctx.Arg("year"); v != "" {
					q.Set("year", v)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/headmaps", user), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "statistics",
			Description: tr.T("cmd.user.statistics.short"),
			Flags:       windowFlags,
			Run: func(ctx *common.RuntimeContext) error {
				return runWindowedUserGet(ctx, "/users/%s/statistics")
			},
		},
		{
			Name:        "stats",
			Description: tr.T("cmd.user.stats.short"),
			Flags:       windowFlags,
			Run: func(ctx *common.RuntimeContext) error {
				return runWindowedUserGet(ctx, "/users/%s/statistics")
			},
		},
		{
			Name:        "project-trends",
			Description: tr.T("cmd.user.project_trends.short"),
			Flags:       windowFlags,
			Run: func(ctx *common.RuntimeContext) error {
				return runWindowedUserGet(ctx, "/users/%s/project_trends")
			},
		},
		{
			Name:        "trends",
			Description: tr.T("cmd.user.trends.short"),
			Flags:       windowFlags,
			Run: func(ctx *common.RuntimeContext) error {
				return runWindowedUserGet(ctx, "/users/%s/project_trends")
			},
		},
	}
}

func runWindowedUserGet(ctx *common.RuntimeContext, pathFormat string) error {
	user, err := resolveUser(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	if v := ctx.Arg("start-time"); v != "" {
		q.Set("start_time", v)
	}
	if v := ctx.Arg("end-time"); v != "" {
		q.Set("end_time", v)
	}
	env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf(pathFormat, user), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func resolveUser(ctx *common.RuntimeContext) (string, error) {
	if v := ctx.Arg("user"); v != "" {
		return v, nil
	}
	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", err
	}
	if login := extractLogin(env); login != "" {
		return login, nil
	}
	return "", fmt.Errorf("%s", ctx.Tr.T("error.user.required"))
}

func extractLogin(env *output.Envelope) string {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return ""
	}
	if v, ok := data["login"].(string); ok {
		return v
	}
	return ""
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
