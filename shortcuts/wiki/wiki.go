package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const wikiBaseURL = "https://gateway.gitlink.org.cn/api"

// Shortcuts returns wiki page management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List wiki pages",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", strconv.Itoa(projectID))
				return callWikiAPI(ctx, "GET", "/wiki/open/wikiPages", nil, q)
			},
		},
		{
			Name:        "view",
			Description: "View a wiki page",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Wiki page name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", strconv.Itoa(projectID))
				q.Set("pageName", name)
				return callWikiAPI(ctx, "GET", "/wiki/open/getWiki", nil, q)
			},
		},
		{
			Name:        "create",
			Description: "Create a wiki page",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Wiki page name", Required: true},
				{Name: "content", Short: "c", Usage: "Page content (will be base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
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
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"owner":          ctx.Owner,
					"repo":           ctx.Repo,
					"projectId":      projectID,
					"pageName":       name,
					"title":          name,
					"message":        ctx.Arg("message"),
					"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
				}
				return callWikiAPI(ctx, "POST", "/wiki/open/createWiki", body, nil)
			},
		},
		{
			Name:        "update",
			Description: "Update a wiki page",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Wiki page name", Required: true},
				{Name: "content", Short: "c", Usage: "New page content (will be base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
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
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"owner":          ctx.Owner,
					"repo":           ctx.Repo,
					"projectId":      projectID,
					"pageName":       name,
					"title":          name,
					"message":        ctx.Arg("message"),
					"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
				}
				return callWikiAPI(ctx, "PUT", "/wiki/open/updateWiki", body, nil)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a wiki page",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Wiki page name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": projectID,
					"pageName":  name,
				}
				return callWikiAPI(ctx, "DELETE", "/wiki/open/deleteWiki", body, nil)
			},
		},
	}
}

// callWikiAPI temporarily switches the client BaseURL to the wiki gateway.
// In test mode (BaseURL is a local httptest server), the switch is skipped.
func callWikiAPI(ctx *common.RuntimeContext, method, path string, body interface{}, query url.Values) error {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	if query != nil {
		env, err := ctx.CallAPIRawWithQuery(method, path, query)
		if err != nil {
			return err
		}
		return ctx.Output(env)
	}
	env, err := ctx.CallAPIRaw(method, path, body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func fetchProjectID(ctx *common.RuntimeContext) (int, error) {
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("获取项目信息失败: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("无法解析项目信息")
	}
	if idFloat, ok := data["project_id"].(float64); ok {
		return int(idFloat), nil
	}
	if idFloat, ok := data["repo_id"].(float64); ok {
		return int(idFloat), nil
	}
	if idFloat, ok := data["id"].(float64); ok {
		return int(idFloat), nil
	}
	return 0, fmt.Errorf("项目 ID 未找到，请确认仓库是否存在")
}
