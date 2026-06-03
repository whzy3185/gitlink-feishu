package search

import (
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "repos",
			Description: tr.T("cmd.search.repos.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				keyword, _ := ctx.RequireArg("keyword")
				q := url.Values{}
				q.Set("search", keyword)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", "/projects", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "users",
			Description: tr.T("cmd.search.users.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword"), Required: true},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				keyword, _ := ctx.RequireArg("keyword")
				q := url.Values{}
				q.Set("search", keyword)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				env, err := ctx.CallAPIWithQuery("GET", "/users/list", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
			{
				Name:        "code",
				Description: "在仓库中搜索代码",
				Flags: []common.Flag{
					{Name: "keyword", Short: "k", Usage: "搜索关键词", Required: true},
					{Name: "owner", Usage: "仓库所有者（可选，限定范围）"},
					{Name: "repo", Usage: "仓库名称（可选，限定范围）"},
					{Name: "language", Usage: "编程语言过滤（如 go, python）"},
					{Name: "page", Short: "p", Usage: "页码", Default: "1"},
					{Name: "limit", Short: "l", Usage: "每页数量", Default: "20"},
				},
				Run: func(ctx *common.RuntimeContext) error {
					keyword, err := ctx.RequireArg("keyword")
					if err != nil {
						return err
					}
					q := url.Values{}
					q.Set("keyword", keyword)
					q.Set("page", ctx.Arg("page"))
					q.Set("limit", ctx.Arg("limit"))
					if lang := ctx.Arg("language"); lang != "" {
						q.Set("language", lang)
					}
					path := "/search/code"
					if owner := ctx.Arg("owner"); owner != "" {
						if repo := ctx.Arg("repo"); repo != "" {
							path = fmt.Sprintf("/%s/%s/search/code", owner, repo)
						}
					}
					env, err := ctx.CallAPIWithQuery("GET", path, q)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				},
			},
			{
				Name:        "issues",
				Description: "搜索 Issue",
				Flags: []common.Flag{
					{Name: "keyword", Short: "k", Usage: "搜索关键词", Required: true},
					{Name: "state", Short: "s", Usage: "状态过滤: open/closed/all", Default: "all"},
					{Name: "label", Usage: "标签过滤"},
					{Name: "author", Usage: "作者过滤"},
					{Name: "page", Short: "p", Usage: "页码", Default: "1"},
					{Name: "limit", Short: "l", Usage: "每页数量", Default: "20"},
				},
				Run: func(ctx *common.RuntimeContext) error {
					keyword, err := ctx.RequireArg("keyword")
					if err != nil {
						return err
					}
					q := url.Values{}
					q.Set("keyword", keyword)
					q.Set("page", ctx.Arg("page"))
					q.Set("limit", ctx.Arg("limit"))
					if state := ctx.Arg("state"); state != "" && state != "all" {
						q.Set("state", state)
					}
					if label := ctx.Arg("label"); label != "" {
						q.Set("label", label)
					}
					if author := ctx.Arg("author"); author != "" {
						q.Set("author", author)
					}
					env, err := ctx.CallAPIWithQuery("GET", "/search/issues", q)
					if err != nil {
						return err
					}
					return ctx.Output(env)
				},
			},
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
