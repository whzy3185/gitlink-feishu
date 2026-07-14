package watch

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "watch",
			Description: "Watch a repository",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("target_type", "project")
				q.Set("id", fmt.Sprintf("%d", projectID))
				env, err := ctx.CallAPIWithQuery("POST", "/watchers/follow", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "unwatch",
			Description: "Unwatch a repository",
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("target_type", "project")
				q.Set("id", fmt.Sprintf("%d", projectID))
				env, err := ctx.CallAPIWithQuery("DELETE", "/watchers/unfollow", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "watchers",
			Description: "List watchers of a repository",
			Flags: []common.Flag{
				{Name: "owner", Short: "o", Usage: "Repository owner", Required: true},
				{Name: "repo", Short: "r", Usage: "Repository name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				owner, _ := ctx.RequireArg("owner")
				repo, _ := ctx.RequireArg("repo")
				ctx.Owner = owner
				ctx.Repo = repo
				env, err := ctx.CallAPI("GET", fmt.Sprintf("/%s/%s/watchers", owner, repo), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func resolveProjectID(ctx *common.RuntimeContext) (int64, error) {
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get project info: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected project info response")
	}
	for _, key := range []string{"id", "project_id", "repo_id"} {
		if id, ok := data[key].(float64); ok {
			return int64(id), nil
		}
	}
	return 0, fmt.Errorf("cannot find project id in response")
}
