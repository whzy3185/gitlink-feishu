package status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/internal/auth"
	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/context"
)

// NewStatusCmd creates the status command that displays login state and context.
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "显示当前登录状态和上下文信息",
		Long: `显示 gitlink-cli 的当前状态，包括：
  - 认证状态（是否已登录、Token 来源）
  - API 地址
  - 当前目录
  - 自动推断的仓库信息`,
		Example: `  gitlink-cli status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			token, _ := auth.LoadToken()
			if token == "" {
				token = os.Getenv("GITLINK_TOKEN")
			}
			cwd, _ := os.Getwd()

			fmt.Println("GitLink CLI 状态")
			fmt.Println("───────────────")

			// 认证状态
			if token != "" {
				fmt.Println("  认证状态:    已登录")
				fmt.Printf("  Token 来源:  %s\n", tokenSource(token))
			} else {
				fmt.Println("  认证状态:    未登录（运行 gitlink-cli auth login）")
			}

			// API 地址
			fmt.Printf("  API 地址:    %s\n", cfg.BaseURL)

			// 当前目录
			fmt.Printf("  当前目录:    %s\n", cwd)

			// 推断的仓库
			owner, repo, err := context.ResolveOwnerRepo("", "")
			if err == nil {
				fmt.Printf("  推断仓库:    %s/%s\n", owner, repo)
			} else {
				fmt.Println("  推断仓库:    （不在 Git 仓库中）")
			}

			return nil
		},
	}
}

func tokenSource(token string) string {
	if token == os.Getenv("GITLINK_TOKEN") {
		return "环境变量 GITLINK_TOKEN"
	}
	return "keyring / 配置文件"
}
