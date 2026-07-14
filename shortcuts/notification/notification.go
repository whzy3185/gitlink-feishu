package notification

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "列出通知",
			Flags: []common.Flag{
				{Name: "all", Usage: "显示所有通知（含已读）", Bool: true, Default: "false"},
				{Name: "participating", Usage: "仅显示参与的通知", Bool: true, Default: "false"},
				{Name: "page", Short: "p", Usage: "页码", Default: "1"},
				{Name: "limit", Short: "l", Usage: "每页数量", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					q.Set("all", "true")
				}
				if ctx.Arg("participating") == "true" {
					q.Set("participating", "true")
				}
				env, err := ctx.CallAPIWithQuery("GET", "/notifications", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "read",
			Description: "标记单条通知为已读",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "通知 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PUT", fmt.Sprintf("/notifications/%s", id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "read-all",
			Description: "标记所有通知为已读",
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("PUT", "/notifications", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "watch",
			Description: "关注或取消关注仓库的通知",
			Flags: []common.Flag{
				{Name: "owner", Short: "o", Usage: "仓库所有者", Required: true},
				{Name: "repo", Short: "r", Usage: "仓库名称", Required: true},
				{Name: "unwatch", Usage: "取消关注（默认为关注）", Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				owner, err := ctx.RequireArg("owner")
				if err != nil {
					return err
				}
				repo, err := ctx.RequireArg("repo")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/watchers/%s/%s.json", owner, repo)
				method := "POST"
				if ctx.Arg("unwatch") == "true" {
					method = "DELETE"
				}
				env, err := ctx.CallAPI(method, path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
