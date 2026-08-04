package compare

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "view",
			Description: tr.T("cmd.compare.view.short"),
			Flags: []common.Flag{
				{Name: "head", Usage: tr.T("flag.compare.head"), Required: true},
				{Name: "base", Usage: tr.T("flag.compare.base"), Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, err := ctx.RequireArg("head")
				if err != nil {
					return err
				}
				base, err := ctx.RequireArg("base")
				if err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", comparePath(ctx, head, base), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: tr.T("cmd.compare.files.short"),
			Flags: []common.Flag{
				{Name: "head", Usage: tr.T("flag.compare.head"), Required: true},
				{Name: "base", Usage: tr.T("flag.compare.base"), Required: true},
				{Name: "file", Short: "f", Usage: tr.T("flag.compare.file")},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if _, err := parsePositiveIntArg(ctx.Arg("limit"), 20, "limit"); err != nil {
					return err
				}
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, err := ctx.RequireArg("head")
				if err != nil {
					return err
				}
				base, err := ctx.RequireArg("base")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if file := ctx.Arg("file"); file != "" {
					q.Set("filepath", file)
				}
				env, err := ctx.CallAPIWithQuery("GET", "/v1"+comparePath(ctx, head, base)+"/files", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "commits",
			Description: "Summarize commits between two refs",
			Flags: []common.Flag{
				{Name: "head", Required: true}, {Name: "base", Required: true},
				{Name: "author"}, {Name: "keyword"}, {Name: "limit", Default: "20"},
				{Name: "reverse", Bool: true, Default: "false"},
			},
			Run: runCompareCommits,
		},
		{
			Name:        "summary",
			Description: "Aggregate commits and changed files between two refs",
			Flags: []common.Flag{
				{Name: "head", Required: true}, {Name: "base", Required: true},
				{Name: "max-files", Default: "200"}, {Name: "top-files", Default: "10"},
				{Name: "commit-limit", Default: "10"},
			},
			Run: runCompareSummary,
		},
	}
}

func comparePath(ctx *common.RuntimeContext, head, base string) string {
	return fmt.Sprintf("%s/compare/%s...%s", ctx.RepoPath(), encodeRef(head), encodeRef(base))
}

func encodeRef(ref string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}
