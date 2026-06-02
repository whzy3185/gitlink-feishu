package user

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "me",
			Description: "Show current authenticated user",
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
			Description: "Show user profile",
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
			},
			Run: func(ctx *common.RuntimeContext) error {
				login, err := ctx.RequireArg("login")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/users/%s/project_trends", login), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
