package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "pages",
			Description: "List repository wiki pages",
			Flags:       wikiProjectFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q, err := wikiProjectQuery(ctx)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/wikiPages", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "Show a wiki page",
			Flags: append(wikiProjectFlags(),
				common.Flag{Name: "page", Short: "p", Usage: "Wiki page name", Required: true},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q, err := wikiProjectQuery(ctx)
				if err != nil {
					return err
				}
				page, err := ctx.RequireArg("page")
				if err != nil {
					return err
				}
				q.Set("pageName", page)
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/getWiki", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a wiki page",
			Flags:       wikiWriteFlags(true),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				payload, err := wikiWritePayload(ctx, true)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", "/wiki/createWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update a wiki page",
			Flags:       wikiWriteFlags(false),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				payload, err := wikiWritePayload(ctx, false)
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("PUT", "/wiki/updateWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a wiki page",
			Flags: append(wikiProjectFlags(),
				common.Flag{Name: "page", Short: "p", Usage: "Wiki page name", Required: true},
				common.Flag{Name: "dry-run", Usage: "Preview the delete request without changing wiki state", Bool: true, Default: "false"},
			),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				payload, err := wikiBasePayload(ctx)
				if err != nil {
					return err
				}
				page, err := ctx.RequireArg("page")
				if err != nil {
					return err
				}
				payload["pageName"] = page
				if ctx.Arg("dry-run") == "true" {
					return ctx.OutputData(map[string]interface{}{
						"dry_run": true,
						"action":  "delete_wiki_page",
						"method":  "DELETE",
						"path":    "/wiki/deleteWiki",
						"payload": payload,
					})
				}
				env, err := ctx.CallAPI("DELETE", "/wiki/deleteWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func wikiProjectFlags() []common.Flag {
	return []common.Flag{
		{Name: "project-id", Usage: "GitLink project ID", Required: true},
	}
}

func wikiWriteFlags(contentRequired bool) []common.Flag {
	return append(wikiProjectFlags(),
		common.Flag{Name: "page", Short: "p", Usage: "Wiki page name", Required: true},
		common.Flag{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
		common.Flag{Name: "message", Short: "m", Usage: "Commit message"},
		common.Flag{Name: "content", Usage: "Wiki content; encoded to base64 before sending"},
		common.Flag{Name: "content-base64", Usage: "Pre-encoded wiki content"},
	)
}

func wikiProjectQuery(ctx *common.RuntimeContext) (url.Values, error) {
	projectID, err := wikiProjectID(ctx)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("projectId", strconv.Itoa(projectID))
	return q, nil
}

func wikiWritePayload(ctx *common.RuntimeContext, contentRequired bool) (map[string]interface{}, error) {
	payload, err := wikiBasePayload(ctx)
	if err != nil {
		return nil, err
	}
	page, err := ctx.RequireArg("page")
	if err != nil {
		return nil, err
	}
	title, err := ctx.RequireArg("title")
	if err != nil {
		return nil, err
	}
	content, ok, err := wikiContent(ctx, contentRequired)
	if err != nil {
		return nil, err
	}
	payload["pageName"] = page
	payload["title"] = title
	if message := ctx.Arg("message"); message != "" {
		payload["message"] = message
	}
	if ok {
		payload["content_base64"] = content
	}
	return payload, nil
}

func wikiBasePayload(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	projectID, err := wikiProjectID(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"owner":     ctx.Owner,
		"repo":      ctx.Repo,
		"projectId": projectID,
	}, nil
}

func wikiProjectID(ctx *common.RuntimeContext) (int, error) {
	value, err := ctx.RequireArg("project-id")
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("--project-id must be a positive integer")
	}
	return id, nil
}

func wikiContent(ctx *common.RuntimeContext, required bool) (string, bool, error) {
	content := ctx.Arg("content")
	encoded := ctx.Arg("content-base64")
	if content != "" && encoded != "" {
		return "", false, fmt.Errorf("--content cannot be used with --content-base64")
	}
	if content != "" {
		return base64.StdEncoding.EncodeToString([]byte(content)), true, nil
	}
	if encoded != "" {
		return encoded, true, nil
	}
	if required {
		return "", false, fmt.Errorf("required flag --content is missing (or use --content-base64)")
	}
	return "", false, nil
}
