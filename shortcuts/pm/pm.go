package pm

import (
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns project management shortcuts for GitLink.
//
// The pm domain provides commands for viewing dashboards, sprints,
// weekly issues, tags, pipelines, and action runs associated with
// a project.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "dashboards",
			Description: "查看项目仪表盘数据",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/dashboards", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "sprints",
			Description: "查看 Sprint 任务列表",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/sprint_issues", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "weekly",
			Description: "查看周报任务",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/weekly_issues", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tags",
			Description: "查看项目 Issue 标签",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/issue_tags", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "pipelines",
			Description: "查看项目 CI/CD 流水线列表",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/pipelines", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "runs",
			Description: "查看项目 Action 运行记录",
			Flags: []common.Flag{
				{Name: "project", Usage: "项目 ID", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				project, err := ctx.RequireArg("project")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("project_id", project)
				env, err := ctx.CallAPIWithQuery("GET", "/pm/action_runs", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
