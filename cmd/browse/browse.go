package browse

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/context"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/internal/web"
)

// stdout is the browse command's output target (so tests can redirect).
var stdout io.Writer = os.Stdout

// browsableKinds maps the first path segment of `browse <kind>[/id>` to a URL
// builder. The "default" entry is used as a fallback that appends the raw arg
// to the repo URL, preserving the original passthrough behaviour.
var browsableKinds = []struct {
	kind string
	desc string
}{
	{"issues", "Issue 列表 / 详情 (issues/42)"},
	{"pulls", "PR 列表 / 详情 (pulls/128)"},
	{"wiki", "Wiki 首页 / 页面 (wiki 或 wiki/API指南)"},
	{"actions", "CI/Actions 页面"},
	{"commits", "提交列表 / 详情 (commits/abc123)"},
	{"branches", "分支列表"},
	{"releases", "Release 列表 / 详情 (releases/v2.0)"},
	{"milestones", "里程碑页面"},
	{"labels", "标签管理页"},
	{"settings/hooks", "Webhook 设置页"},
	{"settings/collaboration", "成员管理页"},
	{"projects", "项目看板页"},
}

// NewBrowseCmd creates the browse command for opening GitLink pages in a browser.
func NewBrowseCmd() *cobra.Command {
	var listFlag, noOpen bool
	cmd := &cobra.Command{
		Use:   "browse [resource]",
		Short: "在浏览器中打开 GitLink 页面",
		Long: `打开当前仓库（或指定资源）的 GitLink 页面。

资源格式: issues/42, pulls/42, wiki, wiki/页面名, commits/abc123, ...
不带参数则打开仓库主页。owner/repo 自动从 git remote 推断或用 --owner/--repo 指定。

示例:
  gitlink-cli browse
  gitlink-cli browse issues/42
  gitlink-cli browse pulls/128
  gitlink-cli browse wiki
  gitlink-cli browse --list           # 列出所有可浏览页面
  gitlink-cli browse --no-open        # 只打印 URL，不打开浏览器`,
		Example: `  gitlink-cli browse
  gitlink-cli browse issues/42
  gitlink-cli browse pulls/128
  gitlink-cli browse wiki
  gitlink-cli browse --list
  gitlink-cli browse --no-open`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := context.ResolveOwnerRepo(cmdutil.Owner, cmdutil.Repo)
			if err != nil {
				return fmt.Errorf("无法推断仓库信息: %w", err)
			}

			if listFlag {
				listBrowsables(owner, repo)
				return nil
			}

			rurl := resolveBrowseURL(web.NewBuilder(), owner, repo, args)
			emitBrowse(rurl)
			if !noOpen {
				if err := web.OpenBrowser(rurl.URL); err != nil {
					// 打开失败仅告警，URL 已经打印供手动复制
					fmt.Fprintf(stdout, "（浏览器未自动打开: %v；请手动复制上方 URL）\n", err)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&listFlag, "list", false, "列出当前仓库所有可浏览的页面")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "只打印 URL，不打开浏览器")
	return cmd
}

// emitBrowse prints the URL — friendly single line by default, structured
// envelope when --format is set.
func emitBrowse(r *web.ResourceURL) {
	if cmdutil.Format == "" {
		fmt.Fprintf(stdout, "🔗 %s\n", r.URL)
		return
	}
	_ = output.PrintTo(stdout, output.SuccessEnvelope(r, nil), cmdutil.Format)
}

// listBrowsables prints the catalog of pages `browse` understands.
func listBrowsables(owner, repo string) {
	fmt.Fprintf(stdout, "可浏览的 GitLink 页面 (%s/%s):\n", owner, repo)
	for _, k := range browsableKinds {
		fmt.Fprintf(stdout, "  %-28s %s\n", k.kind, k.desc)
	}
	fmt.Fprintf(stdout, "\n用法: gitlink-cli browse <资源>\n")
}

// resolveBrowseURL maps `browse <arg>` to a web URL. With no arg → repo home.
func resolveBrowseURL(b *web.Builder, owner, repo string, args []string) *web.ResourceURL {
	if len(args) == 0 || args[0] == "" {
		return b.RepoURL(owner, repo)
	}
	arg := strings.TrimPrefix(args[0], "/")
	// Split into kind and (optional) rest after the first "/".
	kind, rest, _ := strings.Cut(arg, "/")
	rest = strings.Trim(rest, "/")

	switch {
	case kind == "issues" || kind == "issue":
		return b.IssueURL(owner, repo, atoiOrZero(rest))
	case kind == "pulls" || kind == "pr" || kind == "pull":
		return b.PRURL(owner, repo, atoiOrZero(rest))
	case kind == "wiki":
		return b.WikiURL(owner, repo, rest)
	case kind == "actions" || kind == "ci":
		return b.CIURL(owner, repo)
	case kind == "commits":
		return b.CommitURL(owner, repo, rest)
	case kind == "branches":
		return b.BranchURL(owner, repo, rest)
	case kind == "releases":
		return b.ReleaseURL(owner, repo, rest)
	case kind == "milestones":
		return b.MilestoneURL(owner, repo)
	case kind == "labels":
		return b.LabelURL(owner, repo)
	case arg == "settings/hooks":
		return b.WebhookURL(owner, repo)
	case arg == "settings/collaboration":
		return b.MemberURL(owner, repo)
	case kind == "settings":
		return b.RepoURL(owner, repo) // settings landing falls back to repo home
	default:
		// Unknown resource: append the raw arg as a path segment so behaviour
		// stays predictable for callers that already know their URL shape.
		return &web.ResourceURL{
			URL:        b.RepoURL(owner, repo).URL + "/" + arg,
			Resource:   "custom",
			Identifier: arg,
		}
	}
}

func atoiOrZero(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
