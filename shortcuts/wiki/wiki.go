package wiki

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	wikiListPath   = "/api/wiki/wikiPages"
	wikiViewPath   = "/api/wiki/getWiki"
	wikiCreatePath = "/api/wiki/createWiki"
	wikiUpdatePath = "/api/wiki/updateWiki"
	wikiDeletePath = "/api/wiki/deleteWiki"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.wiki.list.short"),
			Flags: []common.Flag{
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id")},
			},
			Run: runListWikiPages,
		},
		{
			Name:        "view",
			Description: tr.T("cmd.wiki.view.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id")},
			},
			Run: runViewWikiPage,
		},
		{
			Name:        "create",
			Description: tr.T("cmd.wiki.create.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
				{Name: "title", Short: "t", Usage: tr.T("flag.wiki.title")},
				{Name: "content", Short: "c", Usage: tr.T("flag.wiki.content")},
				{Name: "file", Short: "f", Usage: tr.T("flag.wiki.file")},
				{Name: "message", Short: "m", Usage: tr.T("flag.wiki.message")},
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id")},
			},
			Run: runCreateWikiPage,
		},
		{
			Name:        "update",
			Description: tr.T("cmd.wiki.update.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
				{Name: "title", Short: "t", Usage: tr.T("flag.wiki.title")},
				{Name: "content", Short: "c", Usage: tr.T("flag.wiki.content")},
				{Name: "file", Short: "f", Usage: tr.T("flag.wiki.file")},
				{Name: "message", Short: "m", Usage: tr.T("flag.wiki.message")},
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id")},
			},
			Run: runUpdateWikiPage,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.wiki.delete.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.wiki.page"), Required: true},
				{Name: "project-id", Usage: tr.T("flag.wiki.project_id")},
			},
			Run: runDeleteWikiPage,
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func runListWikiPages(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	query, err := buildWikiQuery(ctx, "")
	if err != nil {
		return err
	}
	env, err := callWikiAPIWithQuery(ctx, "GET", wikiListPath, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runViewWikiPage(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	page, err := ctx.RequireArg("page")
	if err != nil {
		return err
	}
	query, err := buildWikiQuery(ctx, page)
	if err != nil {
		return err
	}
	env, err := callWikiAPIWithQuery(ctx, "GET", wikiViewPath, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCreateWikiPage(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	payload, err := buildWikiWritePayload(ctx, true)
	if err != nil {
		return err
	}
	env, err := callWikiAPI(ctx, "POST", wikiCreatePath, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runUpdateWikiPage(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	payload, err := buildWikiWritePayload(ctx, false)
	if err != nil {
		return err
	}
	env, err := callWikiAPI(ctx, "PUT", wikiUpdatePath, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDeleteWikiPage(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	page, err := ctx.RequireArg("page")
	if err != nil {
		return err
	}
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"owner":     ctx.Owner,
		"repo":      ctx.Repo,
		"projectId": projectID,
		"pageName":  page,
	}
	env, err := callWikiAPI(ctx, "DELETE", wikiDeletePath, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func buildWikiQuery(ctx *common.RuntimeContext, page string) (url.Values, error) {
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("owner", ctx.Owner)
	query.Set("repo", ctx.Repo)
	query.Set("projectId", strconv.Itoa(projectID))
	if page != "" {
		query.Set("pageName", page)
	}
	return query, nil
}

func callWikiAPI(ctx *common.RuntimeContext, method, path string, body interface{}) (*output.Envelope, error) {
	env, err := ctx.CallAPI(method, path, body)
	if err != nil {
		return nil, wrapWikiAPIError(path, err)
	}
	return env, nil
}

func callWikiAPIWithQuery(ctx *common.RuntimeContext, method, path string, query url.Values) (*output.Envelope, error) {
	env, err := ctx.CallAPIWithQuery(method, path, query)
	if err != nil {
		return nil, wrapWikiAPIError(path, err)
	}
	return env, nil
}

func wrapWikiAPIError(path string, err error) error {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return fmt.Errorf("GitLink Wiki API %s returned 404; confirm the repository has Wiki enabled and the Wiki OpenAPI is available: %w", path, err)
	}
	return err
}

func buildWikiWritePayload(ctx *common.RuntimeContext, requireContent bool) (map[string]interface{}, error) {
	page, err := ctx.RequireArg("page")
	if err != nil {
		return nil, err
	}
	projectID, err := resolveProjectID(ctx)
	if err != nil {
		return nil, err
	}
	title := ctx.Arg("title")
	if title == "" {
		title = page
	}

	payload := map[string]interface{}{
		"owner":     ctx.Owner,
		"repo":      ctx.Repo,
		"projectId": projectID,
		"pageName":  page,
		"title":     title,
	}
	if message := ctx.Arg("message"); message != "" {
		payload["message"] = message
	}
	contentBase64, ok, err := readWikiContent(ctx)
	if err != nil {
		return nil, err
	}
	if ok {
		payload["content_base64"] = contentBase64
	} else if requireContent {
		return nil, fmt.Errorf("one of --content or --file is required")
	}
	return payload, nil
}

func readWikiContent(ctx *common.RuntimeContext) (string, bool, error) {
	content := ctx.Arg("content")
	filePath := ctx.Arg("file")
	if content != "" && filePath != "" {
		return "", false, fmt.Errorf("use only one of --content or --file")
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", false, fmt.Errorf("read wiki file: %w", err)
		}
		return base64.StdEncoding.EncodeToString(data), true, nil
	}
	if content != "" {
		return base64.StdEncoding.EncodeToString([]byte(content)), true, nil
	}
	return "", false, nil
}

func resolveProjectID(ctx *common.RuntimeContext) (int, error) {
	if value := strings.TrimSpace(ctx.Arg("project-id")); value != "" {
		projectID, err := strconv.Atoi(value)
		if err != nil || projectID <= 0 {
			return 0, fmt.Errorf("invalid --project-id %q", value)
		}
		return projectID, nil
	}

	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return 0, fmt.Errorf("resolve project ID: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected repository response format")
	}
	for _, key := range []string{"project_id", "id"} {
		if value, ok := data[key].(float64); ok && value > 0 {
			return int(value), nil
		}
	}
	return 0, fmt.Errorf("repository response missing project_id")
}
