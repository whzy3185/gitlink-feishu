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
				{Name: "all", Usage: tr.T("flag.all"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				keyword, _ := ctx.RequireArg("keyword")
				q := url.Values{}
				q.Set("search", keyword)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey("/projects", q, "projects")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("projects", items))
				}
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
				{Name: "all", Usage: tr.T("flag.all"), Bool: true, Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				keyword, _ := ctx.RequireArg("keyword")
				q := url.Values{}
				q.Set("search", keyword)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if ctx.Arg("all") == "true" {
					items, err := ctx.PaginateAllKey("/users/list", q, "users")
					if err != nil {
						return err
					}
					return ctx.Output(common.NewListEnvelope("users", items))
				}
				env, err := ctx.CallAPIWithQuery("GET", "/users/list", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "issues",
			Description: tr.T("cmd.search.issues.short"),
			Flags: []common.Flag{
				{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword"), Required: true},
				{Name: "category", Short: "c", Usage: tr.T("flag.search.issues.category"), Default: "all"},
				{Name: "assignee", Short: "a", Usage: tr.T("flag.search.issues.assignee")},
				{Name: "author", Usage: tr.T("flag.search.issues.author")},
				{Name: "milestone", Short: "m", Usage: tr.T("flag.search.issues.milestone")},
				{Name: "tag", Short: "t", Usage: tr.T("flag.search.issues.tag")},
				{Name: "sort-by", Usage: tr.T("flag.sort_by"), Default: "updated_on"},
				{Name: "sort-dir", Usage: tr.T("flag.sort_direction"), Default: "desc"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
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
				q.Set("keyword", keyword)
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if c := ctx.Arg("category"); c != "" {
					q.Set("category", c)
				}
				if a := ctx.Arg("assignee"); a != "" {
					q.Set("assigner_id", a)
				}
				if a := ctx.Arg("author"); a != "" {
					q.Set("author_id", a)
				}
				if m := ctx.Arg("milestone"); m != "" {
					q.Set("milestone_id", m)
				}
				if t := ctx.Arg("tag"); t != "" {
					q.Set("issue_tag_ids", t)
				}
				if s := ctx.Arg("sort-by"); s != "" {
					q.Set("sort_by", "issues."+s)
				}
				if d := ctx.Arg("sort-dir"); d != "" {
					q.Set("sort_direction", d)
				}
				env, err := ctx.CallAPIWithQuery("GET",
					fmt.Sprintf("/v1/%s/%s/issues", ctx.Owner, ctx.Repo), q)
				if err != nil {
					return err
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
