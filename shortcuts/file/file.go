package file

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all file shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		viewShortcut(),
		searchShortcut(),
		writeShortcut("create", "Create a new file in the repository"),
		writeShortcut("update", "Update an existing file in the repository"),
		deleteShortcut(),
	}
}

func refFlag() common.Flag {
	return common.Flag{Name: "ref", Usage: "Branch, tag, or commit SHA (defaults to the default branch)"}
}

func viewShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "view",
		Description: "View the contents of a file",
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: "File path", Required: true},
			refFlag(),
			{Name: "raw", Usage: "Print only the decoded file content", Bool: true},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			q := url.Values{}
			q.Set("filepath", path)
			if ref := ctx.Arg("ref"); ref != "" {
				q.Set("ref", ref)
			}
			env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
			if err != nil {
				return err
			}
			if ctx.Arg("raw") == "true" {
				return printRawContent(env.Data)
			}
			return ctx.Output(env)
		},
	}
}

func searchShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "search",
		Description: "Search files in the repository by name",
		Flags: []common.Flag{
			{Name: "keyword", Short: "k", Usage: "Search keyword", Required: true},
			refFlag(),
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			keyword, err := ctx.RequireArg("keyword")
			if err != nil {
				return err
			}
			q := url.Values{}
			q.Set("search", keyword)
			if ref := ctx.Arg("ref"); ref != "" {
				q.Set("ref", ref)
			}
			env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/files", q)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func writeShortcut(action, description string) *common.Shortcut {
	return &common.Shortcut{
		Name:        action,
		Description: description,
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: "File path", Required: true},
			{Name: "content", Short: "c", Usage: "File content"},
			{Name: "content-file", Usage: "Read file content from a local file"},
			{Name: "branch", Short: "b", Usage: "Branch to commit to", Required: true},
			{Name: "new-branch", Usage: "Create a new branch from --branch for the commit"},
			{Name: "message", Short: "m", Usage: "Commit message"},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			branch, err := ctx.RequireArg("branch")
			if err != nil {
				return err
			}
			content, err := resolveContent(ctx)
			if err != nil {
				return err
			}
			message := ctx.Arg("message")
			if message == "" {
				message = fmt.Sprintf("%s %s", action, path)
			}
			payload := map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"action_type": action,
						"file_path":   path,
						"content":     content,
						"encoding":    "text",
					},
				},
				"branch":  branch,
				"message": message,
			}
			if nb := ctx.Arg("new-branch"); nb != "" {
				payload["new_branch"] = nb
			}
			env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", payload)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

func deleteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "delete",
		Description: "Delete a file from the repository",
		Flags: []common.Flag{
			{Name: "path", Short: "p", Usage: "File path", Required: true},
			{Name: "branch", Short: "b", Usage: "Branch to commit to", Required: true},
			{Name: "new-branch", Usage: "Create a new branch from --branch for the commit"},
			{Name: "message", Short: "m", Usage: "Commit message"},
		},
		Run: func(ctx *common.RuntimeContext) error {
			if err := ctx.ResolveOwnerRepo(); err != nil {
				return err
			}
			path, err := ctx.RequireArg("path")
			if err != nil {
				return err
			}
			branch, err := ctx.RequireArg("branch")
			if err != nil {
				return err
			}
			message := ctx.Arg("message")
			if message == "" {
				message = fmt.Sprintf("delete %s", path)
			}
			payload := map[string]interface{}{
				"files": []map[string]interface{}{
					{
						"action_type": "delete",
						"file_path":   path,
						"content":     "",
						"encoding":    "text",
					},
				},
				"branch":  branch,
				"message": message,
			}
			if nb := ctx.Arg("new-branch"); nb != "" {
				payload["new_branch"] = nb
			}
			env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", payload)
			if err != nil {
				return err
			}
			return ctx.Output(env)
		},
	}
}

// resolveContent reads file content from --content or --content-file.
func resolveContent(ctx *common.RuntimeContext) (string, error) {
	content := ctx.Arg("content")
	contentFile := ctx.Arg("content-file")
	if content != "" && contentFile != "" {
		return "", fmt.Errorf("use only one of --content or --content-file")
	}
	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", fmt.Errorf("read content file: %w", err)
		}
		return string(data), nil
	}
	if content == "" {
		return "", fmt.Errorf("one of --content or --content-file is required")
	}
	return content, nil
}

// printRawContent extracts and prints the decoded file content from an API
// response (entries object, readme object, or a bare content field).
func printRawContent(data interface{}) error {
	content, encoding, ok := extractContent(data)
	if !ok {
		return fmt.Errorf("no file content in response (is the path a directory?)")
	}
	if encoding == "base64" {
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			fmt.Print(string(decoded))
			return nil
		}
	}
	fmt.Print(content)
	return nil
}

func extractContent(data interface{}) (content, encoding string, ok bool) {
	m, isMap := data.(map[string]interface{})
	if !isMap {
		return "", "", false
	}
	if entries, has := m["entries"]; has {
		if em, isEM := entries.(map[string]interface{}); isEM {
			m = em
		}
	}
	c, has := m["content"].(string)
	if !has {
		return "", "", false
	}
	if t, hasType := m["type"].(string); hasType && t != "file" {
		return "", "", false
	}
	enc, _ := m["encoding"].(string)
	return c, enc, true
}
