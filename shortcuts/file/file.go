package file

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns all shortcuts for repository file operations.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "browse",
			Description: "Browse repository directory tree or file details",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File or directory path", Required: true},
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA", Default: "master"},
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
				q.Set("ref", ctx.Arg("ref"))
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "get",
			Description: "Get file content (auto-decodes base64)",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File path", Required: true},
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA", Default: "master"},
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
				q.Set("ref", ctx.Arg("ref"))
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a new file in the repository",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File path", Required: true},
				{Name: "content", Short: "c", Usage: "File content (will be base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				{Name: "branch", Short: "b", Usage: "Target branch", Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				message := ctx.Arg("message")
				if message == "" {
					message = fmt.Sprintf("Add %s", path)
				}
				body := map[string]interface{}{
					"filepath":        path,
					"base64_filepath": base64.StdEncoding.EncodeToString([]byte(path)),
					"content":         base64.StdEncoding.EncodeToString([]byte(content)),
					"message":         message,
					"branch":          ctx.Arg("branch"),
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/create_file", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update an existing file in the repository",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File path", Required: true},
				{Name: "content", Short: "c", Usage: "New file content (will be base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				{Name: "branch", Short: "b", Usage: "Target branch", Default: "master"},
				{Name: "sha", Usage: "File SHA (required, fetch automatically if not provided)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				sha := ctx.Arg("sha")
				if sha == "" {
					fetchedSHA, err := fetchFileSHA(ctx, path)
					if err != nil {
						return fmt.Errorf("请使用 --sha 手动指定（获取文件 SHA 失败: %w）", err)
					}
					sha = fetchedSHA
				}
				message := ctx.Arg("message")
				if message == "" {
					message = fmt.Sprintf("Update %s", path)
				}
				body := map[string]interface{}{
					"filepath":        path,
					"base64_filepath": base64.StdEncoding.EncodeToString([]byte(path)),
					"content":         base64.StdEncoding.EncodeToString([]byte(content)),
					"sha":             sha,
					"message":         message,
					"branch":          ctx.Arg("branch"),
				}
				env, err := ctx.CallAPI("PUT", ctx.RepoPath()+"/update_file", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a file from the repository",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File path", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				{Name: "branch", Short: "b", Usage: "Target branch", Default: "master"},
				{Name: "sha", Usage: "File SHA (required, fetch automatically if not provided)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				sha := ctx.Arg("sha")
				if sha == "" {
					fetchedSHA, err := fetchFileSHA(ctx, path)
					if err != nil {
						return fmt.Errorf("请使用 --sha 手动指定（获取文件 SHA 失败: %w）", err)
					}
					sha = fetchedSHA
				}
				message := ctx.Arg("message")
				if message == "" {
					message = fmt.Sprintf("Delete %s", path)
				}
				body := map[string]interface{}{
					"filepath":        path,
					"base64_filepath": base64.StdEncoding.EncodeToString([]byte(path)),
					"sha":             sha,
					"message":         message,
					"branch":          ctx.Arg("branch"),
				}
				env, err := ctx.CallAPI("DELETE", ctx.RepoPath()+"/delete_file", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func fetchFileSHA(ctx *common.RuntimeContext, path string) (string, error) {
	q := url.Values{}
	q.Set("filepath", path)
	q.Set("ref", ctx.Arg("branch"))
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
	if err != nil {
		return "", err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}
	sha, _ := data["sha"].(string)
	if sha == "" {
		return "", fmt.Errorf("SHA not found in response")
	}
	return sha, nil
}
