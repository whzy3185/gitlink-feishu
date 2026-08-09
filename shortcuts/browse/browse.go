package browse

import (
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/browser"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "repo",
			Description: "Open the repository home page in a browser",
			Flags:       []common.Flag{noBrowserFlag},
			Run: page(func(r repoRef) string {
				return r.url()
			}),
		},
		{
			Name:        "issue",
			Description: "Open an issue in a browser",
			Flags:       []common.Flag{numberFlag, noBrowserFlag},
			Run: pageWithArg("number", func(r repoRef, n string) string {
				return r.url("issues", n)
			}),
		},
		{
			Name:        "pr",
			Description: "Open a pull request in a browser",
			Flags:       []common.Flag{numberFlag, noBrowserFlag},
			Run: pageWithArg("number", func(r repoRef, n string) string {
				return r.url("pulls", n)
			}),
		},
		{
			Name:        "commit",
			Description: "Open a commit in a browser",
			Flags:       []common.Flag{{Name: "sha", Usage: "Commit SHA", Required: true}, noBrowserFlag},
			Run: pageWithArg("sha", func(r repoRef, sha string) string {
				return r.url("commits", sha)
			}),
		},
		{
			Name:        "branch",
			Description: "Open a branch's code tree, or the branch list when omitted",
			Flags:       []common.Flag{{Name: "name", Usage: "Branch name"}, noBrowserFlag},
			Run: func(ctx *common.RuntimeContext) error {
				r, err := resolve(ctx)
				if err != nil {
					return err
				}
				if name := ctx.Arg("name"); name != "" {
					return emit(ctx, r.url("tree", name))
				}
				return emit(ctx, r.url("branches"))
			},
		},
		{
			Name:        "file",
			Description: "Open a file or directory in a browser",
			Flags: []common.Flag{
				{Name: "path", Usage: "File or directory path", Required: true},
				{Name: "ref", Usage: "Branch, tag, or commit", Default: "master"},
				noBrowserFlag,
			},
			Run: func(ctx *common.RuntimeContext) error {
				r, err := resolve(ctx)
				if err != nil {
					return err
				}
				path, err := ctx.RequireArg("path")
				if err != nil {
					return err
				}
				return emit(ctx, r.url("tree", ctx.Arg("ref"), path))
			},
		},
		{
			Name:        "releases",
			Description: "Open the releases page in a browser",
			Flags:       []common.Flag{noBrowserFlag},
			Run:         page(func(r repoRef) string { return r.url("releases") }),
		},
		{
			Name:        "wiki",
			Description: "Open the wiki in a browser",
			Flags:       []common.Flag{noBrowserFlag},
			Run:         page(func(r repoRef) string { return r.url("wiki") }),
		},
	}
}

var (
	noBrowserFlag = common.Flag{Name: "no-browser", Short: "n", Usage: "Print the URL instead of opening a browser", Bool: true}
	numberFlag    = common.Flag{Name: "number", Short: "N", Usage: "Issue or pull request number", Required: true}
)

// repoRef builds web URLs for a resolved owner/repo against a web host.
type repoRef struct {
	host  string
	owner string
	repo  string
}

// url joins the repo web root with url-escaped path segments. Multi-segment
// inputs (a file path or a branch like "feat/x") keep their slashes.
func (r repoRef) url(segs ...string) string {
	var b strings.Builder
	b.WriteString(r.host)
	for _, seg := range append([]string{r.owner, r.repo}, segs...) {
		for _, part := range strings.Split(strings.Trim(seg, "/"), "/") {
			if part == "" {
				continue
			}
			b.WriteByte('/')
			b.WriteString(url.PathEscape(part))
		}
	}
	return b.String()
}

func resolve(ctx *common.RuntimeContext) (repoRef, error) {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return repoRef{}, err
	}
	return repoRef{host: webRoot(ctx.Client.BaseURL), owner: ctx.Owner, repo: ctx.Repo}, nil
}

// webRoot turns an API base URL into the web host it fronts.
func webRoot(baseURL string) string {
	return strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/api")
}

func emit(ctx *common.RuntimeContext, target string) error {
	if ctx.Arg("no-browser") != "true" {
		// A launch failure (headless host, no browser) is not fatal: the URL is
		// still printed below so the caller can open it themselves.
		_ = browser.Open(target)
	}
	return ctx.OutputData(map[string]interface{}{"url": target})
}

func page(build func(repoRef) string) func(*common.RuntimeContext) error {
	return func(ctx *common.RuntimeContext) error {
		r, err := resolve(ctx)
		if err != nil {
			return err
		}
		return emit(ctx, build(r))
	}
}

func pageWithArg(flag string, build func(repoRef, string) string) func(*common.RuntimeContext) error {
	return func(ctx *common.RuntimeContext) error {
		r, err := resolve(ctx)
		if err != nil {
			return err
		}
		v, err := ctx.RequireArg(flag)
		if err != nil {
			return err
		}
		return emit(ctx, build(r, v))
	}
}
