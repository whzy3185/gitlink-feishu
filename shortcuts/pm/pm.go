package pm

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "boards",
			Description: "List kanban boards",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/dashboards")
			},
		},
		{
			Name:        "sprints",
			Description: "List sprint issues",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/sprint_issues")
			},
		},
		{
			Name:        "weekly",
			Description: "List weekly reports",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/weekly_issues")
			},
		},
		{
			Name:        "tags",
			Description: "List PM issue tags",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/issue_tags")
			},
		},
		{
			Name:        "pipelines",
			Description: "List PM pipelines",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/pipelines")
			},
		},
		{
			Name:        "actions",
			Description: "List action run records",
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				return listPM(ctx, "/pm/action_runs")
			},
		},
	}
}

func listPM(ctx *common.RuntimeContext, endpoint string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := fetchProjectID(ctx)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("project_id", strconv.Itoa(projectID))
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("page", ctx.Arg("page"))
	q.Set("limit", ctx.Arg("limit"))
	env, err := ctx.CallAPIRawWithQuery("GET", endpoint, q)
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
	if idFloat, ok := data["repo_id"].(float64); ok {
		return int(idFloat), nil
	}
	if idFloat, ok := data["project_id"].(float64); ok {
		return int(idFloat), nil
	}
	if idFloat, ok := data["id"].(float64); ok {
		return int(idFloat), nil
	}
	return 0, fmt.Errorf("项目 ID 未找到，请确认仓库是否存在")
}
