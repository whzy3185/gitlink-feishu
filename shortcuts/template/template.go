package template

import (
	"fmt"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// v1RepoPath returns the v1 API path prefix: /v1/{owner}/{repo}
func v1RepoPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s", ctx.Owner, ctx.Repo)
}

// templatePath returns the API path for template collection.
func templatePath(ctx *common.RuntimeContext) string {
	return v1RepoPath(ctx) + "/project_templates"
}

// templateItemPath returns the API path for a single template.
func templateItemPath(ctx *common.RuntimeContext, id string) string {
	return templatePath(ctx) + "/" + id
}

// Shortcuts returns project template management shortcuts.
//
// Project templates are reusable content templates (e.g., issue templates)
// that can be applied when creating new resources.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "列出项目模板",
			Flags:       []common.Flag{},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", templatePath(ctx), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "get",
			Description: "根据 ID 获取项目模板",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "模板 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", templateItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "创建项目模板",
			Flags: []common.Flag{
				{Name: "type", Short: "t", Usage: "模板类型（如：ProjectTemplates::Issue、ProjectTemplates::PullRequest）", Required: true},
				{Name: "name", Short: "n", Usage: "模板名称", Required: true},
				{Name: "content", Short: "c", Usage: "模板内容（支持 Markdown）", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				templateType, err := ctx.RequireArg("type")
				if err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				payload := map[string]interface{}{
					"type":    templateType,
					"name":    name,
					"content": content,
				}
				env, err := ctx.CallAPI("POST", templatePath(ctx), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "更新项目模板",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "模板 ID", Required: true},
				{Name: "type", Short: "t", Usage: "模板类型（如：ProjectTemplates::Issue、ProjectTemplates::PullRequest）"},
				{Name: "name", Short: "n", Usage: "模板名称"},
				{Name: "content", Short: "c", Usage: "模板内容（支持 Markdown）"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				// At least one field must be provided for update
				if ctx.Arg("type") == "" && ctx.Arg("name") == "" && ctx.Arg("content") == "" {
					return fmt.Errorf("至少需要提供 --type、--name 或 --content 中的一个")
				}
				payload := map[string]interface{}{}
				if t := ctx.Arg("type"); t != "" {
					payload["type"] = t
				}
				if n := ctx.Arg("name"); n != "" {
					payload["name"] = n
				}
				if c := ctx.Arg("content"); c != "" {
					payload["content"] = c
				}
				env, err := ctx.CallAPI("PUT", templateItemPath(ctx, id), payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "删除项目模板",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "模板 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", templateItemPath(ctx, id), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
