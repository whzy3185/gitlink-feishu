package search

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "issues",
			Description: "Search repository issues",
			Flags: []common.Flag{
				{Name: "keyword", Required: true}, {Name: "category"},
				{Name: "assignee"}, {Name: "author"}, {Name: "milestone"}, {Name: "tag"},
				{Name: "sort-by"}, {Name: "sort-dir"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				keyword, err := ctx.RequireArg("keyword")
				if err != nil {
					return err
				}
				query := url.Values{"keyword": []string{keyword}}
				for arg, field := range map[string]string{
					"category": "category", "assignee": "assigner_id", "author": "author_id",
					"milestone": "milestone_id", "tag": "issue_tag_ids", "sort-dir": "sort_direction",
				} {
					if value := ctx.Arg(arg); value != "" {
						query.Set(field, value)
					}
				}
				if value := ctx.Arg("sort-by"); value != "" {
					if !strings.Contains(value, ".") {
						value = "issues." + value
					}
					query.Set("sort_by", value)
				}
				env, err := ctx.CallAPIWithQuery("GET", fmt.Sprintf("/v1/%s/%s/issues", ctx.Owner, ctx.Repo), query)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
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
		},
		{
			Name:        "recommend",
			Description: tr.T("cmd.search.recommend.short"),
			Long:        tr.T("cmd.search.recommend.long"),
			Run: func(ctx *common.RuntimeContext) error {
				env, err := ctx.CallAPI("GET", "/projects/recommend", nil)
				if err != nil {
					return err
				}
				// This endpoint returns a bare JSON array, which the client
				// surfaces as a raw string; normalize it to structured data.
				if s, ok := env.Data.(string); ok {
					var arr interface{}
					if json.Unmarshal([]byte(s), &arr) == nil {
						return ctx.OutputData(arr)
					}
				}
				return ctx.Output(env)
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
