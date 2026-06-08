package wiki

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const wikiBaseURL = "https://gateway.gitlink.org.cn/api"

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
				q.Set("projectId", fmt.Sprintf("%d", projectID))
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
				q.Set("projectId", fmt.Sprintf("%d", projectID))
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
			Description: "Delete a wiki page and remove it from sidebar",
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

				// Step 1: Delete the wiki page
				body := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": projectID,
					"pageName":  name,
				}
				if err := callWikiAPISilent(ctx, "DELETE", "/wiki/open/deleteWiki", body, nil); err != nil {
					return err
				}

				// Step 2: Wait for GitLink async sidebar rebuild, then clean up
				time.Sleep(2 * time.Second)
				cleanSidebar(ctx, projectID, name)

				fmt.Printf("Wiki page %q deleted successfully.\n", name)
				return nil
			},
		},
	}
}

// callWikiAPI sends a request to the wiki gateway.
// It switches the client BaseURL to the wiki gateway for the duration of the call,
// but skips the switch during tests (local httptest server).
func callWikiAPI(ctx *common.RuntimeContext, method, path string, body interface{}, query url.Values) error {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") && !strings.HasPrefix(origBase, "http://localhost") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	env, err := ctx.Client.DoRaw(method, path, body, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// fetchProjectID resolves the numeric project ID from the repo info API.
func fetchProjectID(ctx *common.RuntimeContext) (int64, error) {
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get project info: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected project info response")
	}
	for _, key := range []string{"project_id", "repo_id", "id"} {
		if id, ok := data[key].(float64); ok {
			return int64(id), nil
		}
	}
	return 0, fmt.Errorf("project id not found in response")
}

// callWikiAPISilent is like callWikiAPI but does not print output.
func callWikiAPISilent(ctx *common.RuntimeContext, method, path string, body interface{}, query url.Values) error {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") && !strings.HasPrefix(origBase, "http://localhost") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	_, err := ctx.Client.DoRaw(method, path, body, query)
	return err
}

const sidebarPageName = "_Sidebar" // GitLink uses capital S for the sidebar page

// cleanSidebar fetches the wiki sidebar, removes the deleted page link, and updates it.
func cleanSidebar(ctx *common.RuntimeContext, projectID int64, pageName string) {
	// Fetch sidebar
	q := url.Values{}
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("projectId", fmt.Sprintf("%d", projectID))
	q.Set("pageName", sidebarPageName)

	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") && !strings.HasPrefix(origBase, "http://localhost") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	env, err := ctx.Client.DoRaw("GET", "/wiki/open/getWiki", nil, q)
	if err != nil {
		return // sidebar might not exist, silently skip
	}

	// Extract content_base64 from response.
	// DoRaw auto-parses JSON, so env.Data is a map with "data" as either
	// a nested dict (already parsed) or a JSON string (needs parsing).
	outer, ok := env.Data.(map[string]interface{})
	if !ok {
		return
	}

	var inner map[string]interface{}
	switch v := outer["data"].(type) {
	case map[string]interface{}:
		inner = v
	case string:
		if err := json.Unmarshal([]byte(v), &inner); err != nil {
			return
		}
	default:
		return
	}

	contentB64, ok := inner["content_base64"].(string)
	if !ok {
		return
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return
	}
	sidebar := string(contentBytes)

	// Remove the line containing [[pageName]]
	target := "[[" + pageName + "]]"
	lines := strings.Split(sidebar, "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != target {
			newLines = append(newLines, line)
		}
	}
	newSidebar := strings.Join(newLines, "\n")

	// No change needed
	if newSidebar == sidebar {
		return
	}

	// Update sidebar
	body := map[string]interface{}{
		"owner":          ctx.Owner,
		"repo":           ctx.Repo,
		"projectId":      projectID,
		"pageName":       sidebarPageName,
		"title":          sidebarPageName,
		"message":        "Remove deleted page " + pageName + " from sidebar",
		"content_base64": base64.StdEncoding.EncodeToString([]byte(newSidebar)),
	}
	ctx.Client.DoRaw("PUT", "/wiki/open/updateWiki", body, nil)
}
