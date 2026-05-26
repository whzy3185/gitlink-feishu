package health

import (
	"fmt"
	"net/url"
	"os"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// v1RepoPath constructs the v1 API path for a repository.
// Issue operations use the v1 API (/v1/{owner}/{repo}) to match shortcuts/issue conventions,
// while PR operations use the v2 API via ctx.RepoPath() (/{owner}/{repo}/pulls) to match shortcuts/pr conventions.

func v1RepoPath(owner, repo string) string {
	return fmt.Sprintf("/v1/%s/%s", owner, repo)
}

func fetchPRListPage(ctx *common.RuntimeContext, state string, page, limit int) ([]interface{}, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if state != "" {
		q.Set("state", state)
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/pulls", q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  CLI error: pr +list state=%s page=%d: %v\n", state, page, err)
		return nil, err
	}
	if !env.OK {
		err := fmt.Errorf("API error: pr +list state=%s page=%d", state, page)
		fmt.Fprintf(os.Stderr, "  %v\n", err)
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response type for pr +list")
	}
	issues, _ := data["issues"].([]interface{})
	return issues, nil
}

// fetchPRDetail retrieves full PR detail to extract merged_at timestamp.
// The list API doesn't return merged_at; only the detail endpoint does.
func fetchPRDetail(ctx *common.RuntimeContext, prNumber int) (string, error) {
	path := fmt.Sprintf("%s/pulls/%d", ctx.RepoPath(), prNumber)
	env, err := ctx.CallAPI("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("PR detail fetch #%d: %w", prNumber, err)
	}
	if !env.OK {
		return "", fmt.Errorf("PR detail API error #%d", prNumber)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", nil
	}

	// Try multiple possible locations for merged_at in the detail response
	for _, candidate := range []string{
		"merged_at",
		"mergedAt",
	} {
		// data.pull_request.merged_at
		if pr, _ := data["pull_request"].(map[string]interface{}); pr != nil {
			if v, _ := pr[candidate].(string); v != "" {
				return v, nil
			}
		}
		// data.issue.merged_at
		if issue, _ := data["issue"].(map[string]interface{}); issue != nil {
			if v, _ := issue[candidate].(string); v != "" {
				return v, nil
			}
		}
		// data.merged_at (flat)
		if v, _ := data[candidate].(string); v != "" {
			return v, nil
		}
	}
	return "", nil
}

func fetchIssueListPage(ctx *common.RuntimeContext, owner, repo, state string, page, limit int) ([]interface{}, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	if state != "" {
		q.Set("state", state)
	}
	env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(owner, repo)+"/issues", q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  CLI error: issue +list state=%s page=%d: %v\n", state, page, err)
		return nil, err
	}
	if !env.OK {
		err := fmt.Errorf("API error: issue +list state=%s page=%d", state, page)
		fmt.Fprintf(os.Stderr, "  %v\n", err)
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response type for issue +list")
	}
	issues, _ := data["issues"].([]interface{})
	return issues, nil
}
