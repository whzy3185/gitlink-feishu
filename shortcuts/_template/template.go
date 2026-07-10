// Package template is the starter scaffold for a new shortcut group.
//
// 复制本目录并改名后使用（见同目录 README.md 的 5 步流程）。
// 目录以 `_` 开头，Go 工具链自动忽略，不参与构建。
package template

import (
	"errors"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns the command group's subcommands.
// 三个样例覆盖读 / 写 / 删三种典型形态与本仓库全部惯例。
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List gadgets", // 实际命令请改用 i18n 键（见 dev-guide 第 4 节）
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: "Filter by keyword"},
				{Name: "page", Usage: "Page number", Default: "1"},
				{Name: "limit", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				// 惯例 1：先解析 owner/repo（flag 优先，其次 git remote 自动探测）
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if v := ctx.Arg("keyword"); v != "" {
					q.Set("keyword", v)
				}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				// 惯例 2：所有网络调用走 common 助手（认证/错误处理/重试统一）
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/gadgets", q)
				if err != nil {
					return err
				}
				// 惯例 3：输出统一走信封（自动兼容 --format 与 --jq）
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a gadget",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Gadget name", Required: true},
				{Name: "description", Short: "d", Usage: "Gadget description"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				body := map[string]interface{}{"name": name}
				if v := ctx.Arg("description"); v != "" {
					body["description"] = v
				}
				env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/gadgets", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a gadget",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Gadget ID", Required: true},
				{Name: "yes", Short: "y", Usage: "Confirm deletion", Bool: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				// 惯例 4：破坏性操作必须有 --yes 确认保护
				if ctx.Arg("yes") != "true" {
					return errors.New("destructive operation: re-run with --yes to confirm")
				}
				env, err := ctx.CallAPI("DELETE", "/v1"+ctx.RepoPath()+"/gadgets/"+id, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
