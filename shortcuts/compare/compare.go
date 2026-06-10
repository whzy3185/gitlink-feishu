package compare

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
			Name:        "view",
			Description: "Compare two branches, tags, or commits",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, base, err := resolveCompareRefs(ctx)
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
			Description: "List changed files between two refs",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
				{Name: "file", Short: "f", Usage: "Filter by file path"},
				{Name: "page", Short: "p", Usage: "Page number", Default: "1"},
				{Name: "limit", Short: "l", Usage: "Items per page", Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				head, base, err := resolveCompareRefs(ctx)
				if err != nil {
					return err
				}
				page, err := parsePositiveIntArg(ctx.Arg("page"), 1, "page")
				if err != nil {
					return err
				}
				limit, err := parsePositiveIntArg(ctx.Arg("limit"), 20, "limit")
				if err != nil {
					return err
				}
				q := url.Values{}
				q.Set("page", strconv.Itoa(page))
				q.Set("limit", strconv.Itoa(limit))
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
			Description: "List commits between two refs with optional filters",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
				{Name: "author", Usage: "Filter by commit author or committer"},
				{Name: "keyword", Short: "k", Usage: "Filter by commit message keyword"},
				{Name: "limit", Short: "l", Usage: "Maximum commits to return", Default: "20"},
				{Name: "reverse", Usage: "Return commits in reverse order", Bool: true, Default: "false"},
			},
			Run: runCompareCommits,
		},
		{
			Name:        "summary",
			Description: "Summarize commits and changed files between two refs",
			Flags: []common.Flag{
				{Name: "head", Usage: "Source branch, tag, or commit", Required: true},
				{Name: "base", Usage: "Target branch, tag, or commit", Required: true},
				{Name: "max-files", Usage: "Maximum changed files to analyze", Default: "200"},
				{Name: "top-files", Usage: "Maximum top changed files to include", Default: "10"},
				{Name: "commit-limit", Usage: "Maximum commits to include in the summary sample", Default: "10"},
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

func resolveCompareRefs(ctx *common.RuntimeContext) (string, string, error) {
	head, err := ctx.RequireArg("head")
	if err != nil {
		return "", "", err
	}
	base, err := ctx.RequireArg("base")
	if err != nil {
		return "", "", err
	}
	return head, base, nil
}
