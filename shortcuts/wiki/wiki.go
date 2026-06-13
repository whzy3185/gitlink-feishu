// Package wiki implements shortcuts for managing a repository's wiki pages
// (list, view, create, update, delete) and exporting the wiki. These wrap
// GitLink's /api/wiki and /api/wikiExport endpoints, which require the numeric
// GitLink project ID in addition to owner/repo.
package wiki

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns wiki management shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)

	projectIDFlag := common.Flag{Name: "project-id", Usage: tr.T("flag.wiki.project_id")}
	contentFlags := []common.Flag{
		{Name: "content", Short: "c", Usage: tr.T("flag.wiki.content")},
		{Name: "content-file", Usage: tr.T("flag.wiki.content_file")},
		{Name: "content-base64", Usage: tr.T("flag.wiki.content_base64")},
	}
	dryRunFlag := common.Flag{Name: "dry-run", Usage: tr.T("flag.wiki.dry_run"), Bool: true, Default: "false"}

	writeFlags := func() []common.Flag {
		flags := []common.Flag{
			projectIDFlag,
			{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
			{Name: "title", Short: "t", Usage: tr.T("flag.wiki.title")},
			{Name: "message", Short: "m", Usage: tr.T("flag.wiki.message")},
		}
		flags = append(flags, contentFlags...)
		flags = append(flags, dryRunFlag)
		return flags
	}

	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.wiki.list.short"),
			Long:        tr.T("cmd.wiki.list.long"),
			Flags:       []common.Flag{projectIDFlag},
			Run: func(ctx *common.RuntimeContext) error {
				projectID, err := prepare(ctx)
				if err != nil {
					return err
				}
				q := baseQuery(ctx, projectID)
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/wikiPages", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "view",
			Description: tr.T("cmd.wiki.view.short"),
			Long:        tr.T("cmd.wiki.view.long"),
			Flags: []common.Flag{
				projectIDFlag,
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				projectID, err := prepare(ctx)
				if err != nil {
					return err
				}
				page, err := ctx.RequireArg("page")
				if err != nil {
					return err
				}
				q := baseQuery(ctx, projectID)
				q.Set("pageName", page)
				env, err := ctx.CallAPIWithQuery("GET", "/wiki/open/getWiki", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.wiki.create.short"),
			Long:        tr.T("cmd.wiki.create.long"),
			Flags:       writeFlags(),
			Run:         runWrite("POST", "/wiki/open/createWiki", true),
		},
		{
			Name:        "update",
			Description: tr.T("cmd.wiki.update.short"),
			Long:        tr.T("cmd.wiki.update.long"),
			Flags:       writeFlags(),
			Run:         runWrite("PUT", "/wiki/open/updateWiki", false),
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.wiki.delete.short"),
			Long:        tr.T("cmd.wiki.delete.long"),
			Flags: []common.Flag{
				projectIDFlag,
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
				dryRunFlag,
			},
			Run: func(ctx *common.RuntimeContext) error {
				projectID, err := prepare(ctx)
				if err != nil {
					return err
				}
				page, err := ctx.RequireArg("page")
				if err != nil {
					return err
				}
				body := map[string]interface{}{
					"owner":     ctx.Owner,
					"repo":      ctx.Repo,
					"projectId": projectID,
					"pageName":  page,
				}
				if ctx.Arg("dry-run") == "true" {
					return dryRun(ctx, "DELETE", "/wiki/open/deleteWiki", body)
				}
				env, err := ctx.CallAPI("DELETE", "/wiki/open/deleteWiki", body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "export",
			Description: tr.T("cmd.wiki.export.short"),
			Long:        tr.T("cmd.wiki.export.long"),
			Flags: []common.Flag{
				projectIDFlag,
				{Name: "type", Short: "t", Usage: tr.T("flag.wiki.export_type"), Default: "markdown"},
				{Name: "project-name", Usage: tr.T("flag.wiki.project_name")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				projectID, err := prepare(ctx)
				if err != nil {
					return err
				}
				exportType := ctx.Arg("type")
				if exportType == "" {
					exportType = "markdown"
				}
				if !validExportType(exportType) {
					return fmt.Errorf("%s", ctx.Tr.T("error.wiki.export_type"))
				}
				projectName := ctx.Arg("project-name")
				if projectName == "" {
					projectName = ctx.Repo
				}
				q := url.Values{}
				q.Set("owner", ctx.Owner)
				q.Set("repoName", ctx.Repo)
				q.Set("projectId", strconv.FormatInt(projectID, 10))
				q.Set("projectName", projectName)
				q.Set("type", exportType)
				env, err := ctx.CallAPIWithQuery("GET", "/wikiExport/wikiExport-wrapper", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

// runWrite builds the create/update handlers, which share the same request body.
// contentRequired distinguishes create (content mandatory) from update (optional).
func runWrite(method, path string, contentRequired bool) func(ctx *common.RuntimeContext) error {
	return func(ctx *common.RuntimeContext) error {
		projectID, err := prepare(ctx)
		if err != nil {
			return err
		}
		page, err := ctx.RequireArg("page")
		if err != nil {
			return err
		}
		title := ctx.Arg("title")
		if title == "" {
			title = page
		}
		content, hasContent, err := wikiContent(ctx)
		if err != nil {
			return err
		}
		if contentRequired && !hasContent {
			return fmt.Errorf("%s", ctx.Tr.T("error.wiki.content_required"))
		}
		body := map[string]interface{}{
			"owner":     ctx.Owner,
			"repo":      ctx.Repo,
			"projectId": projectID,
			"pageName":  page,
			"title":     title,
			"message":   ctx.Arg("message"),
		}
		if hasContent {
			body["content_base64"] = content
		}
		if ctx.Arg("dry-run") == "true" {
			return dryRun(ctx, method, path, body)
		}
		env, err := ctx.CallAPI(method, path, body)
		if err != nil {
			return err
		}
		return ctx.Output(env)
	}
}

// prepare resolves owner/repo and the numeric project ID for wiki requests,
// and switches the API base URL to the gateway endpoint that hosts wiki APIs.
func prepare(ctx *common.RuntimeContext) (int64, error) {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return 0, err
	}
	// Resolve project ID first (requires www base URL for /owner/repo endpoint).
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	// Switch to gateway for wiki API calls (/wiki/open/* only available there).
	switchToGateway(ctx)
	return projectID, nil
}

// switchToGateway replaces the www subdomain with gateway in the API base URL.
// Wiki endpoints (/wiki/open/*) are only available on gateway.gitlink.org.cn.
func switchToGateway(ctx *common.RuntimeContext) {
	ctx.Client.BaseURL = strings.Replace(ctx.Client.BaseURL, "www.gitlink.org.cn", "gateway.gitlink.org.cn", 1)
}

// baseQuery returns the owner/repo/projectId query shared by read endpoints.
func baseQuery(ctx *common.RuntimeContext, projectID int64) url.Values {
	q := url.Values{}
	q.Set("owner", ctx.Owner)
	q.Set("repo", ctx.Repo)
	q.Set("projectId", strconv.FormatInt(projectID, 10))
	return q
}

// wikiContent returns the base64-encoded wiki content from --content-base64,
// --content, or --content-file (in that order of precedence).
func wikiContent(ctx *common.RuntimeContext) (string, bool, error) {
	if raw := ctx.Arg("content-base64"); raw != "" {
		return raw, true, nil
	}
	if text := ctx.Arg("content"); text != "" {
		return base64.StdEncoding.EncodeToString([]byte(text)), true, nil
	}
	if file := ctx.Arg("content-file"); file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", false, fmt.Errorf("read --content-file: %w", err)
		}
		return base64.StdEncoding.EncodeToString(data), true, nil
	}
	return "", false, nil
}

func dryRun(ctx *common.RuntimeContext, method, path string, body map[string]interface{}) error {
	preview := map[string]interface{}{
		"dry_run": true,
		"method":  method,
		"path":    path,
		"body":    body,
	}
	return ctx.OutputData(preview)
}

func validExportType(t string) bool {
	switch t {
	case "pdf", "markdown", "html":
		return true
	}
	return false
}

// resolveProjectID returns the numeric GitLink project ID from --project-id,
// falling back to the repository's id resolved via owner/repo.
func resolveProjectID(ctx *common.RuntimeContext) (int64, error) {
	if raw := strings.TrimSpace(ctx.Arg("project-id")); raw != "" {
		return normalizeProjectID(raw)
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"id", "project_id"} {
		if id, ok := projectIDValue(data[key]); ok {
			return id, nil
		}
	}
	return 0, fmt.Errorf("resolve project id: repository response did not include id")
}

func normalizeProjectID(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --project-id %q: use a positive numeric project ID", value)
	}
	return parsed, nil
}

func projectIDValue(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return int64(v), true
		}
	case int:
		if v > 0 {
			return int64(v), true
		}
	case int64:
		if v > 0 {
			return v, true
		}
	case string:
		if id, err := normalizeProjectID(v); err == nil {
			return id, true
		}
	}
	return 0, false
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
