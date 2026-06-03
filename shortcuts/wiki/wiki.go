package wiki

import (
	"fmt"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns wiki management shortcuts for GitLink.
//
// The wiki domain provides commands for listing, viewing, creating,
// updating, and deleting wiki pages within a repository.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "pages",
			Description: "列出 Wiki 页面",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", "/api/wiki/wikiPages", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "get",
			Description: "获取 Wiki 页面内容",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Wiki 页面 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("/api/wiki/getWiki?id=%s", id)
				env, err := ctx.CallAPI("GET", path, nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "创建 Wiki 页面",
			Flags: []common.Flag{
				{Name: "title", Short: "t", Usage: "页面标题", Required: true},
				{Name: "content", Short: "c", Usage: "页面内容（Markdown）", Required: true},
				{Name: "project", Usage: "项目 ID"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title, err := ctx.RequireArg("title")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"title":   title,
					"content": content,
				}
				if project := ctx.Arg("project"); project != "" {
					body["project_id"] = project
				}
				env, err := ctx.CallAPI("POST", "/api/wiki/createWiki", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "更新 Wiki 页面",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Wiki 页面 ID", Required: true},
				{Name: "title", Short: "t", Usage: "新标题"},
				{Name: "content", Short: "c", Usage: "新内容（Markdown）"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{"id": id}
				if t := ctx.Arg("title"); t != "" {
					body["title"] = t
				}
				if c := ctx.Arg("content"); c != "" {
					body["content"] = c
				}
				env, err := ctx.CallAPI("PUT", "/api/wiki/updateWiki", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "删除 Wiki 页面",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Wiki 页面 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				body := map[string]interface{}{"id": id}
				env, err := ctx.CallAPI("POST", "/api/wiki/deleteWiki", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
