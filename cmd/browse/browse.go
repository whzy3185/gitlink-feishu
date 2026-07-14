package browse

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/internal/context"
)

// NewBrowseCmd creates the browse command for opening GitLink pages in a browser.
func NewBrowseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "browse [resource]",
		Short: "在浏览器中打开 GitLink 页面",
		Long: `打开当前仓库（或指定资源）的 GitLink 页面。

如果不带参数，打开当前仓库主页。
资源格式: issues/42, pulls/42, wiki

浏览器打开命令:
  - macOS: open
  - Windows: start
  - Linux: xdg-open`,
		Example: `  gitlink-cli browse
  gitlink-cli browse issues/42
  gitlink-cli browse pulls/128
  gitlink-cli browse wiki`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, err := context.ResolveOwnerRepo("", "")
			if err != nil {
				return fmt.Errorf("无法推断仓库信息: %w", err)
			}
			url := fmt.Sprintf("https://gitlink.org.cn/%s/%s", owner, repo)
			if len(args) > 0 {
				url += "/" + args[0]
			}
			fmt.Printf("正在打开: %s\n", url)
			return openBrowser(url)
		},
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
