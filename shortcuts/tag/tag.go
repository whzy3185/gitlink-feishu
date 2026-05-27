package tag

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.tag.list.short"),
			Flags: []common.Flag{
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: runList,
		},
		{
			Name:        "names",
			Description: tr.T("cmd.tag.names.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: tr.T("flag.tag.keyword")},
			},
			Run: runNames,
		},
		{
			Name:        "view",
			Description: tr.T("cmd.tag.view.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.tag.name"), Required: true},
			},
			Run: runView,
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.tag.delete.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.tag.name"), Required: true},
			},
			Run: runDelete,
		},
	}
}

func runList(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	query, err := paginationQuery(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPIWithQuery("GET", tagPath(ctx), query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runNames(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	query := url.Values{}
	query.Set("only_name", "true")
	if keyword := strings.TrimSpace(ctx.Arg("keyword")); keyword != "" {
		query.Set("name", keyword)
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/tags", query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runView(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := requireTagName(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", tagItemPath(ctx, name), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	name, err := requireTagName(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("DELETE", tagItemPath(ctx, name), nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func tagPath(ctx *common.RuntimeContext) string {
	return fmt.Sprintf("/v1/%s/%s/tags", ctx.Owner, ctx.Repo)
}

func tagItemPath(ctx *common.RuntimeContext, name string) string {
	return fmt.Sprintf("%s/%s", tagPath(ctx), url.PathEscape(name))
}

func paginationQuery(ctx *common.RuntimeContext) (url.Values, error) {
	page, err := positiveIntArg(defaultString(ctx.Arg("page"), "1"), "page")
	if err != nil {
		return nil, err
	}
	limit, err := positiveIntArg(defaultString(ctx.Arg("limit"), "20"), "limit")
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(limit))
	return query, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func positiveIntArg(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", name, value)
	}
	return parsed, nil
}

func requireTagName(ctx *common.RuntimeContext) (string, error) {
	name, err := ctx.RequireArg("name")
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("required flag --name is empty")
	}
	return name, nil
}
