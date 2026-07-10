package wiki

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/output"
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
			Description: "Create a wiki page (optionally in a directory)",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Wiki page name", Required: true},
				{Name: "content", Short: "c", Usage: "Page content (will be base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				{Name: "dir", Short: "d", Usage: "Parent directory to create page in"},
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

				// Step 1: Create the wiki page
				body := map[string]interface{}{
					"owner":          ctx.Owner,
					"repo":           ctx.Repo,
					"projectId":      projectID,
					"pageName":       name,
					"title":          name,
					"message":        ctx.Arg("message"),
					"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
				}
				if err := callWikiAPI(ctx, "POST", "/wiki/open/createWiki", body, nil); err != nil {
					return err
				}

				// Step 2: If --dir specified, add page link under that directory in sidebar
				if dir := ctx.Arg("dir"); dir != "" {
					time.Sleep(1 * time.Second)
					if err := addPageToSidebarDir(ctx, projectID, name, dir); err != nil {
						fmt.Printf("Page created, but failed to add to directory %q in sidebar: %v\n", dir, err)
					} else {
						fmt.Printf("Page %q added to directory %q in sidebar.\n", name, dir)
					}
				}

				return nil
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

				// Step 1: Delete the wiki page content
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
		{
			Name:        "mkdir",
			Description: "Create a wiki directory (use --parent for subdirectory)",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Directory name", Required: true},
				{Name: "parent", Short: "p", Usage: "Parent directory name (creates subdirectory)"},
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
				parent := ctx.Arg("parent")

				if err := createDirectoryInSidebar(ctx, projectID, name, parent); err != nil {
					return err
				}

				if parent != "" {
					fmt.Printf("Subdirectory %q created under %q.\n", name, parent)
				} else {
					fmt.Printf("Directory %q created.\n", name)
				}
				return nil
			},
		},
		{
			Name:        "rmdir",
			Description: "Remove a wiki directory from sidebar",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Directory name", Required: true},
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

				if err := removeDirectoryFromSidebar(ctx, projectID, name); err != nil {
					return err
				}

				fmt.Printf("Directory %q removed from sidebar.\n", name)
				return nil
			},
		},
		{
			Name:        "rename",
			Description: "Rename a wiki page",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Current page name", Required: true},
				{Name: "new-name", Short: "N", Usage: "New page name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				oldName, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				newName, err := ctx.RequireArg("new-name")
				if err != nil {
					return err
				}
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}

				if err := renameWikiPage(ctx, projectID, oldName, newName); err != nil {
					return err
				}

				fmt.Printf("Page renamed from %q to %q.\n", oldName, newName)
				return nil
			},
		},
		{
			Name:        "renamedir",
			Description: "Rename a wiki directory in sidebar",
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: "Current directory name", Required: true},
				{Name: "new-name", Short: "N", Usage: "New directory name", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				oldName, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				newName, err := ctx.RequireArg("new-name")
				if err != nil {
					return err
				}
				projectID, err := fetchProjectID(ctx)
				if err != nil {
					return err
				}

				if err := renameDirectoryInSidebar(ctx, projectID, oldName, newName); err != nil {
					return err
				}

				fmt.Printf("Directory renamed from %q to %q.\n", oldName, newName)
				return nil
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Wiki gateway helpers
// ---------------------------------------------------------------------------

// callWikiAPI temporarily switches the client BaseURL to the wiki gateway.
// In test mode (BaseURL is a local httptest server), the switch is skipped.
func callWikiAPI(ctx *common.RuntimeContext, method, path string, body interface{}, query url.Values) error {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	var env *output.Envelope
	var err error
	if query != nil {
		env, err = ctx.CallAPIRawWithQuery(method, path, query)
	} else {
		env, err = ctx.CallAPIRaw(method, path, body)
	}
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

// callWikiAPISilent is like callWikiAPI but does not print output.
func callWikiAPISilent(ctx *common.RuntimeContext, method, path string, body interface{}, query url.Values) error {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	defer func() { ctx.Client.BaseURL = origBase }()

	_, err := ctx.CallAPIRaw(method, path, body)
	return err
}

const sidebarPageName = "_Sidebar"

// readSidebarContent fetches and decodes the _Sidebar content.
func readSidebarContent(ctx *common.RuntimeContext, projectID int) (string, error) {
	q := url.Values{}
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("projectId", strconv.Itoa(projectID))
	q.Set("pageName", sidebarPageName)

	env, err := ctx.CallAPIRawWithQuery("GET", "/wiki/open/getWiki", q)
	if err != nil {
		return "", fmt.Errorf("failed to read sidebar: %w", err)
	}

	outer, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected sidebar response")
	}

	var inner map[string]interface{}
	switch v := outer["data"].(type) {
	case map[string]interface{}:
		inner = v
	case string:
		if err := json.Unmarshal([]byte(v), &inner); err != nil {
			return "", fmt.Errorf("failed to parse sidebar data: %w", err)
		}
	default:
		return "", fmt.Errorf("sidebar data not found")
	}

	contentB64, ok := inner["content_base64"].(string)
	if !ok {
		return "", fmt.Errorf("sidebar content_base64 not found")
	}
	contentBytes, err := base64.StdEncoding.DecodeString(contentB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode sidebar: %w", err)
	}
	return string(contentBytes), nil
}

// updateSidebarContent writes new content to the _Sidebar page.
func updateSidebarContent(ctx *common.RuntimeContext, projectID int, content, message string) error {
	body := map[string]interface{}{
		"owner":          ctx.Owner,
		"repo":           ctx.Repo,
		"projectId":      projectID,
		"pageName":       sidebarPageName,
		"title":          sidebarPageName,
		"message":        message,
		"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	_, err := ctx.CallAPIRaw("PUT", "/wiki/open/updateWiki", body)
	return err
}

// withWikiGateway temporarily switches BaseURL to the wiki gateway.
func withWikiGateway(ctx *common.RuntimeContext) func() {
	origBase := ctx.Client.BaseURL
	if !strings.HasPrefix(origBase, "http://127.0.0.1") {
		ctx.Client.BaseURL = wikiBaseURL
	}
	return func() { ctx.Client.BaseURL = origBase }
}

// ---------------------------------------------------------------------------
// Sidebar manipulation
// ---------------------------------------------------------------------------

// cleanSidebar fetches the wiki sidebar, removes the deleted page link, and updates it.
func cleanSidebar(ctx *common.RuntimeContext, projectID int, pageName string) {
	defer withWikiGateway(ctx)()

	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return
	}

	target := "[[" + pageName + "]]"
	lines := strings.Split(sidebar, "\n")
	var newLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != target {
			newLines = append(newLines, line)
		}
	}
	newSidebar := strings.Join(newLines, "\n")
	if newSidebar == sidebar {
		return
	}

	updateSidebarContent(ctx, projectID, newSidebar, "Remove deleted page "+pageName+" from sidebar")
}

// addPageToSidebarDir adds a [[pageName]] link under the specified directory in the sidebar.
func addPageToSidebarDir(ctx *common.RuntimeContext, projectID int, pageName, dirName string) error {
	defer withWikiGateway(ctx)()

	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return err
	}

	lines := strings.Split(sidebar, "\n")
	dirLineIdx := findDirectoryLine(lines, dirName)
	if dirLineIdx == -1 {
		return fmt.Errorf("directory %q not found in sidebar", dirName)
	}

	// Find the insert position: after the last child of this directory
	insertIdx := findDirectoryEnd(lines, dirLineIdx)
	dirIndent := countIndent(lines[dirLineIdx])
	newLine := strings.Repeat("\t", dirIndent+1) + "[[" + pageName + "]]"

	// Insert the new page link
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:insertIdx]...)
	result = append(result, newLine)
	result = append(result, lines[insertIdx:]...)

	newSidebar := strings.Join(result, "\n")
	return updateSidebarContent(ctx, projectID, newSidebar, "Add page "+pageName+" to directory "+dirName)
}

// createDirectoryInSidebar creates a new directory entry in the sidebar.
// If parent is empty, creates a top-level directory; otherwise creates a subdirectory.
func createDirectoryInSidebar(ctx *common.RuntimeContext, projectID int, name, parent string) error {
	defer withWikiGateway(ctx)()

	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return err
	}

	lines := strings.Split(sidebar, "\n")

	if parent == "" {
		// Top-level directory: append at the end
		newLine := "- " + name
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			sidebar += "\n" + newLine
		} else {
			sidebar += newLine
		}
	} else {
		// Subdirectory: find parent and insert under it
		parentIdx := findDirectoryLine(lines, parent)
		if parentIdx == -1 {
			return fmt.Errorf("parent directory %q not found in sidebar", parent)
		}
		insertIdx := findDirectoryEnd(lines, parentIdx)
		parentIndent := countIndent(lines[parentIdx])
		newLine := strings.Repeat("\t", parentIndent+1) + "- " + name

		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:insertIdx]...)
		result = append(result, newLine)
		result = append(result, lines[insertIdx:]...)
		sidebar = strings.Join(result, "\n")
	}

	return updateSidebarContent(ctx, projectID, sidebar, "Create directory "+name)
}

// removeDirectoryFromSidebar removes a directory entry (and its children) from the sidebar.
func removeDirectoryFromSidebar(ctx *common.RuntimeContext, projectID int, dirName string) error {
	defer withWikiGateway(ctx)()

	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return err
	}

	lines := strings.Split(sidebar, "\n")
	dirLineIdx := findDirectoryLine(lines, dirName)
	if dirLineIdx == -1 {
		return fmt.Errorf("directory %q not found in sidebar", dirName)
	}

	// Remove the directory line and all its children (lines with greater indent)
	dirIndent := countIndent(lines[dirLineIdx])
	endIdx := dirLineIdx + 1
	for endIdx < len(lines) {
		if strings.TrimSpace(lines[endIdx]) == "" {
			break
		}
		if countIndent(lines[endIdx]) <= dirIndent {
			break
		}
		endIdx++
	}

	result := make([]string, 0, len(lines)-(endIdx-dirLineIdx))
	result = append(result, lines[:dirLineIdx]...)
	result = append(result, lines[endIdx:]...)
	newSidebar := strings.Join(result, "\n")

	return updateSidebarContent(ctx, projectID, newSidebar, "Remove directory "+dirName)
}

// renameWikiPage renames a page: get content → create new → delete old → update sidebar.
func renameWikiPage(ctx *common.RuntimeContext, projectID int, oldName, newName string) error {
	defer withWikiGateway(ctx)()

	// Step 1: Get old page content
	q := url.Values{}
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("projectId", strconv.Itoa(projectID))
	q.Set("pageName", oldName)

	env, err := ctx.CallAPIRawWithQuery("GET", "/wiki/open/getWiki", q)
	if err != nil {
		return fmt.Errorf("failed to get page %q: %w", oldName, err)
	}

	var contentB64, message string
	outer, ok := env.Data.(map[string]interface{})
	if ok {
		var inner map[string]interface{}
		switch v := outer["data"].(type) {
		case map[string]interface{}:
			inner = v
		case string:
			json.Unmarshal([]byte(v), &inner)
		}
		if inner != nil {
			if c, ok := inner["content_base64"].(string); ok {
				contentB64 = c
			}
			if m, ok := inner["message"].(string); ok {
				message = m
			}
		}
	}
	if contentB64 == "" {
		return fmt.Errorf("could not read content of page %q", oldName)
	}

	// Step 2: Create new page with old content
	createBody := map[string]interface{}{
		"owner":          ctx.Owner,
		"repo":           ctx.Repo,
		"projectId":      projectID,
		"pageName":       newName,
		"title":          newName,
		"message":        "Rename from " + oldName,
		"content_base64": contentB64,
	}
	if _, err := ctx.CallAPIRaw("POST", "/wiki/open/createWiki", createBody); err != nil {
		return fmt.Errorf("failed to create page %q: %w", newName, err)
	}

	// Step 3: Delete old page
	deleteBody := map[string]interface{}{
		"owner":     ctx.Owner,
		"repo":      ctx.Repo,
		"projectId": projectID,
		"pageName":  oldName,
	}
	ctx.CallAPIRaw("DELETE", "/wiki/open/deleteWiki", deleteBody)

	// Step 4: Update sidebar: [[oldName]] → [[newName]]
	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return nil // page renamed, sidebar update is best-effort
	}
	newSidebar := strings.ReplaceAll(sidebar, "[["+oldName+"]]", "[["+newName+"]]")
	if newSidebar != sidebar {
		updateSidebarContent(ctx, projectID, newSidebar, "Rename page "+oldName+" to "+newName)
	}

	_ = message
	return nil
}

// renameDirectoryInSidebar renames a directory entry in the sidebar.
func renameDirectoryInSidebar(ctx *common.RuntimeContext, projectID int, oldName, newName string) error {
	defer withWikiGateway(ctx)()

	sidebar, err := readSidebarContent(ctx, projectID)
	if err != nil {
		return err
	}

	lines := strings.Split(sidebar, "\n")
	dirLineIdx := findDirectoryLine(lines, oldName)
	if dirLineIdx == -1 {
		return fmt.Errorf("directory %q not found in sidebar", oldName)
	}

	// Replace the directory name on that line
	oldEntry := "- " + oldName
	newEntry := "- " + newName
	lines[dirLineIdx] = strings.Replace(lines[dirLineIdx], oldEntry, newEntry, 1)

	newSidebar := strings.Join(lines, "\n")
	return updateSidebarContent(ctx, projectID, newSidebar, "Rename directory "+oldName+" to "+newName)
}

// findDirectoryLine returns the line index of "- dirName" in the sidebar lines.
func findDirectoryLine(lines []string, dirName string) int {
	target := "- " + dirName
	for i, line := range lines {
		if strings.TrimSpace(line) == target {
			return i
		}
	}
	return -1
}

// findDirectoryEnd returns the line index after the last child of the directory at dirLineIdx.
func findDirectoryEnd(lines []string, dirLineIdx int) int {
	dirIndent := countIndent(lines[dirLineIdx])
	for i := dirLineIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if countIndent(lines[i]) <= dirIndent {
			return i
		}
	}
	return len(lines)
}

// countIndent returns the number of leading tabs in a line.
func countIndent(line string) int {
	n := 0
	for _, ch := range line {
		if ch == '\t' {
			n++
		} else {
			break
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Project ID resolution
// ---------------------------------------------------------------------------

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
