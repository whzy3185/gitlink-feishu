package health

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func v1RepoPath(owner, repo string) string {
	return fmt.Sprintf("/v1/%s/%s", owner, repo)
}

func fetchPRListPage(ctx *common.RuntimeContext, state string, page, limit int) ([]interface{}, map[string]interface{}) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if state != "" {
		q.Set("state", state)
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/pulls", q)
	if err != nil {
		fmt.Printf("  CLI error: pr +list state=%s page=%d: %v\n", state, page, err)
		return nil, nil
	}
	if !env.OK {
		fmt.Printf("  API error: pr +list state=%s page=%d\n", state, page)
		return nil, nil
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	issues, _ := data["issues"].([]interface{})
	return issues, data
}

func fetchIssueListPage(ctx *common.RuntimeContext, owner, repo, state string, page, limit int) ([]interface{}, map[string]interface{}) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if state != "" {
		q.Set("state", state)
	}
	env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(owner, repo)+"/issues", q)
	if err != nil {
		fmt.Printf("  CLI error: issue +list state=%s page=%d: %v\n", state, page, err)
		return nil, nil
	}
	if !env.OK {
		fmt.Printf("  API error: issue +list state=%s page=%d\n", state, page)
		return nil, nil
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	issues, _ := data["issues"].([]interface{})
	return issues, data
}

func sleep() {
	time.Sleep(300 * time.Millisecond)
}
