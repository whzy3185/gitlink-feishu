// Package profile implements user profile shortcuts that surface GitLink's
// platform statistics (develop ability, role, major, activity, contribution).
// These power "research subject portrait" scenarios for the gitlink-cli skills.
package profile

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns user profile (research subject portrait) shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)

	userFlag := common.Flag{Name: "user", Short: "u", Usage: tr.T("flag.profile.user")}
	timeFlags := []common.Flag{
		{Name: "start-time", Usage: tr.T("flag.profile.start_time")},
		{Name: "end-time", Usage: tr.T("flag.profile.end_time")},
	}

	statFlags := append([]common.Flag{userFlag}, timeFlags...)

	return []*common.Shortcut{
		{
			Name:        "ability",
			Description: tr.T("cmd.profile.ability.short"),
			Long:        tr.T("cmd.profile.ability.long"),
			Flags:       statFlags,
			Run:         statRun("develop"),
		},
		{
			Name:        "role",
			Description: tr.T("cmd.profile.role.short"),
			Long:        tr.T("cmd.profile.role.long"),
			Flags:       statFlags,
			Run:         statRun("role"),
		},
		{
			Name:        "major",
			Description: tr.T("cmd.profile.major.short"),
			Long:        tr.T("cmd.profile.major.long"),
			Flags:       statFlags,
			Run:         statRun("major"),
		},
		{
			Name:        "activity",
			Description: tr.T("cmd.profile.activity.short"),
			Long:        tr.T("cmd.profile.activity.long"),
			Flags:       []common.Flag{userFlag},
			Run: func(ctx *common.RuntimeContext) error {
				user, err := resolveUser(ctx)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/statistics/activity", user), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "contribution",
			Description: tr.T("cmd.profile.contribution.short"),
			Long:        tr.T("cmd.profile.contribution.long"),
			Flags: []common.Flag{
				userFlag,
				{Name: "year", Usage: tr.T("flag.profile.year")},
			},
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
	}
}

// statRun builds a Run that calls /users/{user}/statistics/{kind} with an
// optional start_time/end_time window (Unix timestamps).
func statRun(kind string) func(ctx *common.RuntimeContext) error {
	return func(ctx *common.RuntimeContext) error {
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
		env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/statistics/%s", user, kind), q)
		if err != nil {
			return err
		}
		return ctx.Output(env)
	}
}

// resolveUser returns the target user identifier from the --user flag, falling
// back to the currently authenticated user (/users/me) when the flag is omitted.
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
	return "", fmt.Errorf("%s", ctx.Tr.T("error.profile.user_required"))
}

// extractLogin pulls the "login" identifier out of a /users/me envelope.
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
