package user

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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
			Name:        "activity",
			Description: "Show recent user activity statistics",
			Flags:       userLoginFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runUserStats(ctx, "statistics/activity", false)
			},
		},
		{
			Name:        "headmap",
			Description: "Show user contribution heatmap data",
			Flags: append(userLoginFlags(), common.Flag{
				Name: "year", Short: "y", Usage: "Contribution year, for example 2026",
			}),
			Run: runHeadmap,
		},
		{
			Name:        "develop",
			Description: "Show user development capability statistics",
			Flags:       userStatisticsRangeFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runUserStats(ctx, "statistics/develop", true)
			},
		},
		{
			Name:        "role",
			Description: "Show user role distribution statistics",
			Flags:       userStatisticsRangeFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runUserStats(ctx, "statistics/role", true)
			},
		},
		{
			Name:        "major",
			Description: "Show user professional category statistics",
			Flags:       userStatisticsRangeFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runUserStats(ctx, "statistics/major", true)
			},
		},
	}
}

func userLoginFlags() []common.Flag {
	return []common.Flag{{Name: "login", Short: "l", Usage: "User login; defaults to --owner or current user"}}
}

func userStatisticsRangeFlags() []common.Flag {
	return append(userLoginFlags(),
		common.Flag{Name: "start-time", Usage: "Start Unix timestamp"},
		common.Flag{Name: "end-time", Usage: "End Unix timestamp"},
	)
}

func runHeadmap(ctx *common.RuntimeContext) error {
	login, err := resolveUserLogin(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	if year := strings.TrimSpace(ctx.Arg("year")); year != "" {
		if err := validateYear(year); err != nil {
			return err
		}
		q.Set("year", year)
	}
	env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/headmaps", login), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runUserStats(ctx *common.RuntimeContext, suffix string, withRange bool) error {
	login, err := resolveUserLogin(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	if withRange {
		start, hasStart, err := addTimestampQuery(q, "start_time", "start-time", ctx.Arg("start-time"))
		if err != nil {
			return err
		}
		end, hasEnd, err := addTimestampQuery(q, "end_time", "end-time", ctx.Arg("end-time"))
		if err != nil {
			return err
		}
		if hasStart && hasEnd && start > end {
			return fmt.Errorf("start-time must be less than or equal to end-time")
		}
	}
	env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/users/%s/%s", login, suffix), q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func resolveUserLogin(ctx *common.RuntimeContext) (string, error) {
	if login := strings.TrimSpace(ctx.Arg("login")); login != "" {
		return login, nil
	}
	if owner := strings.TrimSpace(ctx.Owner); owner != "" {
		return owner, nil
	}
	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("resolve current user: %w", err)
	}
	data, _ := env.Data.(map[string]interface{})
	login, _ := data["login"].(string)
	if login == "" {
		return "", fmt.Errorf("cannot determine current user login; pass --login")
	}
	return login, nil
}

func addTimestampQuery(q url.Values, queryName, flagName, value string) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	n, err := parseNonNegativeInt(flagName, value)
	if err != nil {
		return 0, false, err
	}
	q.Set(queryName, value)
	return n, true, nil
}

func parseNonNegativeInt(name, value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return n, nil
}

func validateYear(value string) error {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1970 || n > 9999 {
		return fmt.Errorf("year must be a four-digit year")
	}
	return nil
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
