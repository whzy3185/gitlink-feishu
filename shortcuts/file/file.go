package file

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List repository files",
			Flags: []common.Flag{
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA"},
				{Name: "search", Short: "s", Usage: "Search keyword"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if ref := ctx.Arg("ref"); ref != "" {
					q.Set("ref", ref)
				}
				if search := ctx.Arg("search"); search != "" {
					q.Set("search", search)
				}
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/files", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tree",
			Description: "List file tree for a branch or commit",
			Flags: []common.Flag{
				{Name: "sha", Short: "s", Usage: "Branch, tag, or commit SHA", Default: "master"},
				{Name: "recursive", Usage: "Recursively list all files", Bool: true, Default: "false"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				sha := ctx.Arg("sha")
				if sha == "" {
					sha = "master"
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("recursive") == "true" {
					q.Set("recursive", "true")
				}
				env, err := ctx.CallAPIWithQuery("GET",
					fmt.Sprintf("/v1/%s/%s/git/trees/%s", ctx.Owner, ctx.Repo, sha), q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "get",
			Description: "Get file or directory contents",
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: "File or directory path", Required: true},
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA", Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				filePath, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("filepath", filePath)
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
				{Name: "content", Short: "c", Usage: "File content (plain text, auto Base64 encoded)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message", Required: true},
				{Name: "branch", Short: "b", Usage: "Target branch", Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				filePath, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				content, err := ctx.RequireArg("content")
				if err != nil {
					return err
				}
				message, err := ctx.RequireArg("message")
				if err != nil {
					return err
				}
				branch := ctx.Arg("branch")
				if branch == "" {
					branch = "master"
				}
				body := map[string]interface{}{
					"filepath": filePath,
					"content":  base64.StdEncoding.EncodeToString([]byte(content)),
					"message":  message,
					"branch":   branch,
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/create_file", body)
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
				{Name: "sha", Short: "s", Usage: "File blob SHA (from file +list)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message", Required: true},
				{Name: "branch", Short: "b", Usage: "Target branch", Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				filePath, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				sha, err := ctx.RequireArg("sha")
				if err != nil {
					return err
				}
				message, err := ctx.RequireArg("message")
				if err != nil {
					return err
				}
				branch := ctx.Arg("branch")
				if branch == "" {
					branch = "master"
				}
				body := map[string]interface{}{
					"filepath": filePath,
					"sha":      sha,
					"message":  message,
					"branch":   branch,
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
