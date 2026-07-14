package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// switchToGateway overrides the client base URL with the gateway URL from config.
func switchToGateway(ctx *common.RuntimeContext) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = config.DefaultGatewayURL
	}
	ctx.Client.BaseURL = cfg.GatewayURL
	return nil
}

// gatewayFlag returns the common --gateway flag definition.
func gatewayFlag() common.Flag {
	return common.Flag{Name: "gateway", Short: "g", Usage: "Use gateway API endpoint", Bool: true}
}

// projectIDFlag returns the common --project-id flag definition.
func projectIDFlag() common.Flag {
	return common.Flag{Name: "project-id", Usage: "GitLink project ID (auto-resolved from the repository when omitted)"}
}

// resolveProjectID returns the explicit --project-id value, or resolves it
// from the repository detail endpoint on the main API. It must be called
// before switching the client to the gateway base URL.
func resolveProjectID(ctx *common.RuntimeContext) (string, error) {
	if raw := strings.TrimSpace(ctx.Arg("project-id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return "", fmt.Errorf("invalid --project-id %q: use a positive numeric project ID", raw)
		}
		return strconv.FormatInt(parsed, 10), nil
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return "", fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"project_id", "id"} {
		switch v := data[key].(type) {
		case float64:
			if v > 0 {
				return strconv.FormatInt(int64(v), 10), nil
			}
		case string:
			if s := strings.TrimSpace(v); s != "" {
				return s, nil
			}
		}
	}
	return "", fmt.Errorf("resolve project id: repository response did not include project_id; pass --project-id explicitly")
}

// Shortcuts returns all wiki shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: "List wiki pages",
			Flags: []common.Flag{
				projectIDFlag(),
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", projectID)
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/wikiPages", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: "View a wiki page by page name",
			Flags: []common.Flag{
				projectIDFlag(),
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repo", ctx.Repo)
				q.Set("projectId", projectID)
				q.Set("pageName", ctx.Arg("page-name"))
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/getWiki", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: "Create a new wiki page",
			Flags: []common.Flag{
				projectIDFlag(),
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
				{Name: "content", Short: "c", Usage: "Wiki page content (markdown)", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message"},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				content := ctx.Arg("content")
				payload := map[string]interface{}{
					"owner":          ctx.Owner,
					"repo":           ctx.Repo,
					"projectId":      projectID,
					"pageName":       ctx.Arg("page-name"),
					"title":          ctx.Arg("title"),
					"content_base64": base64.StdEncoding.EncodeToString([]byte(content)),
					"message":        ctx.Arg("message"),
				}
				env, err := ctx.CallAPI("POST", "/wiki/open/createWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "update",
			Description: "Update an existing wiki page",
			Flags: []common.Flag{
				projectIDFlag(),
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				{Name: "title", Short: "t", Usage: "Wiki page title", Required: true},
				{Name: "content", Short: "c", Usage: "Wiki page content (markdown)"},
				{Name: "message", Short: "m", Usage: "Commit message"},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				title := ctx.Arg("title")
				if title == "" {
					return fmt.Errorf("--title is required")
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				content := ctx.Arg("content")
				payload := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": projectID,
					"pageName":  ctx.Arg("page-name"),
					"title":     title,
					"message":   ctx.Arg("message"),
				}
				if content != "" {
					payload["content_base64"] = base64.StdEncoding.EncodeToString([]byte(content))
				}
				env, err := ctx.CallAPI("PUT", "/wiki/open/updateWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: "Delete a wiki page",
			Flags: []common.Flag{
				projectIDFlag(),
				{Name: "page-name", Short: "n", Usage: "Wiki page name (slug)", Required: true},
				gatewayFlag(),
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx)
				if err != nil {
					return err
				}
				if ctx.Arg("gateway") == "true" {
					if err := switchToGateway(ctx); err != nil {
						return err
					}
				}
				payload := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": projectID,
					"pageName":  ctx.Arg("page-name"),
				}
				env, err := ctx.CallAPI("DELETE", "/wiki/open/deleteWiki", payload)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}
