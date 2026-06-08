package commit

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List commits in a repository",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Branch, tag, or commit SHA"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return fmt.Errorf("解析仓库信息失败: %w", err)
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if sha := ctx.Arg("sha"); sha != "" {
					q.Set("sha", sha)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/v1/%s/%s/commits", ctx.Owner, ctx.Repo), q)
				if err != nil {
					return fmt.Errorf("获取提交列表失败: %w", err)
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View files changed in a commit",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Commit SHA", Required: true},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return fmt.Errorf("解析仓库信息失败: %w", err)
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/v1/%s/%s/commits/%s/files", ctx.Owner, ctx.Repo, sha), q)
				if err != nil {
					return fmt.Errorf("查看提交详情失败: %w", err)
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "diff",
			Description: "Show diff for a commit",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Commit SHA", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return fmt.Errorf("解析仓库信息失败: %w", err)
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/v1/%s/%s/commits/%s/diff", ctx.Owner, ctx.Repo, sha), nil)
				if err != nil {
					return fmt.Errorf("获取提交 Diff 失败: %w", err)
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "blame",
			Description: "Show blame for a file",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File path", Required: true},
				{Name: "sha", Short: "s", Usage: "Branch, tag, or commit SHA", Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return fmt.Errorf("解析仓库信息失败: %w", err)
				}
				filePath, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("filepath", filePath)
				q.Set("sha", ctx.Arg("sha"))
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/v1/%s/%s/blame", ctx.Owner, ctx.Repo), q)
				if err != nil {
					return fmt.Errorf("获取 Blame 信息失败: %w", err)
				}
				return ctx.Output(env)
			},
		},
	}
}
