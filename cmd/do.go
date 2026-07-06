package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// NLCommand 自然语言命令路由
// 用法: gitlink-cli do "列出issue" 或 gitlink-cli do wiki
// 关键词匹配 → 推荐命令 + 显示参数 + 简短/完全示例
// 输入模块名（如 wiki）→ 列出该模块全部命令

func newDoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `do "自然语言描述或模块名"`,
		Short: "Natural language command helper (e.g. do \"list issues\" or do wiki)",
		Long: `用自然语言描述你想做的事，自动匹配命令并显示参数。
也可以直接输入模块名查看该模块全部命令。

示例:
  gitlink-cli do 列出issue        # 匹配到 issue +list
  gitlink-cli do wiki             # 列出 wiki 全部命令
  gitlink-cli do 创建标签         # 匹配到 label +create`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := strings.ToLower(strings.Join(args, " "))
			input = strings.TrimSpace(input)
			exePath, _ := os.Executable()

			// 1. 先检查是否是模块名（如 wiki/issue/pr/label...）
			modules := []string{"wiki", "issue", "pr", "label", "notification", "snippet",
				"repo", "release", "branch", "member", "milestone", "webhook",
				"ci", "search", "org", "user", "compare", "workflow"}
			for _, mod := range modules {
				if input == mod {
					fmt.Printf("📦 %s 模块全部命令：\n", mod)
					fmt.Println(strings.Repeat("=", 50))
					helpCmd := exec.Command(exePath, mod, "--help")
					helpCmd.Stdout = os.Stdout
					helpCmd.Stderr = os.Stderr
					helpCmd.Run()
					fmt.Println(strings.Repeat("=", 50))
					fmt.Println("\n💡 选择一个命令运行，例如：")
					fmt.Printf("  gitlink-cli %s +list\n", mod)
					fmt.Printf("  gitlink-cli %s +list --owner ylly --repo gitlink-cli --format json  （完全版）\n", mod)
					return
				}
			}

			// 2. 关键词匹配
			matched := matchNL(input)
			if matched == "" {
				fmt.Println("❌ 未识别。试试这些：")
				fmt.Println("  输入模块名（wiki/issue/pr/label/notification/snippet/repo...）")
				fmt.Println("  或描述操作（列出issue/创建标签/登录/搜索仓库...）")
				fmt.Println("\n示例:")
				fmt.Println("  gitlink-cli do wiki           # 查看 wiki 全部命令")
				fmt.Println("  gitlink-cli do 列出issue       # 匹配 issue +list")
				os.Exit(1)
			}

			parts := strings.Fields(matched)
			fmt.Printf("✅ 匹配到命令: %s\n\n", matched)

			// 显示该命令的 --help
			if len(parts) >= 2 {
				group := parts[0]
				sub := parts[1]
				fmt.Println("📋 命令参数说明：")
				fmt.Println(strings.Repeat("-", 50))
				helpCmd := exec.Command(exePath, group, sub, "--help")
				helpCmd.Stdout = os.Stdout
				helpCmd.Stderr = os.Stderr
				helpCmd.Run()
				fmt.Println(strings.Repeat("-", 50))

				// 显示简短版 + 完全版示例
				fmt.Println("\n💡 命令示例：")
				fmt.Printf("  简短版（在自己的仓库目录里）：\n")
				fmt.Printf("    gitlink-cli %s\n", matched)
				fmt.Printf("  完全版（任何目录都能用）：\n")
				fmt.Printf("    gitlink-cli %s --owner ylly --repo gitlink-cli --format json\n", matched)
			} else {
				// auth login / auth status 等无子命令的
				fmt.Printf("\n💡 运行：gitlink-cli %s\n", matched)
			}
		},
	}
}

// matchNL 关键词匹配自然语言 → 命令
func matchNL(input string) string {
	type rule struct {
		keywords []string
		cmd      string
	}
	rules := []rule{
		// Issue
		{[]string{"issue", "疑修", "问题", "list", "列", "查看"}, "issue +list"},
		{[]string{"issue", "create", "新建", "创建", "提"}, "issue +create"},
		{[]string{"issue", "view", "详情", "看"}, "issue +view"},
		{[]string{"issue", "close", "关闭"}, "issue +close"},
		{[]string{"issue", "update", "更新", "修改"}, "issue +update"},
		{[]string{"issue", "comment", "评论", "回复"}, "issue +comment"},
		{[]string{"issue", "batch", "批量", "关闭"}, "issue +batch-close"},
		{[]string{"issue", "assigners", "负责人", "分配", "候选"}, "issue +assigners"},
		{[]string{"issue", "authors", "作者"}, "issue +authors"},
		// PR
		{[]string{"pr", "pull", "合并请求", "merge", "list"}, "pr +list"},
		{[]string{"pr", "create", "新建", "提交"}, "pr +create"},
		{[]string{"pr", "view", "详情", "看"}, "pr +view"},
		{[]string{"pr", "merge", "合并"}, "pr +merge"},
		{[]string{"pr", "diff", "差异", "变更"}, "pr +diff"},
		{[]string{"pr", "files", "文件", "变更文件"}, "pr +files"},
		{[]string{"pr", "review", "审查", "review"}, "pr +review"},
		{[]string{"pr", "close", "关闭"}, "pr +close"},
		{[]string{"pr", "reopen", "重开", "重新打开"}, "pr +reopen"},
		// Label
		{[]string{"label", "tag", "标签", "list"}, "label +list"},
		{[]string{"label", "tag", "标签", "create", "新建", "创建"}, "label +create"},
		{[]string{"label", "tag", "标签", "update", "更新", "修改"}, "label +update"},
		{[]string{"label", "tag", "标签", "delete", "删除"}, "label +delete"},
		{[]string{"label", "tag", "标签", "batch", "批量"}, "label +batch-create"},
		// Wiki
		{[]string{"wiki", "文档", "知识库", "list"}, "wiki +list"},
		{[]string{"wiki", "文档", "create", "新建", "创建"}, "wiki +create"},
		{[]string{"wiki", "文档", "view", "查看"}, "wiki +view"},
		{[]string{"wiki", "文档", "update", "更新"}, "wiki +update"},
		{[]string{"wiki", "文档", "delete", "删除"}, "wiki +delete"},
		// Release
		{[]string{"release", "发布", "版本", "list"}, "release +list"},
		{[]string{"release", "发布", "create", "新建"}, "release +create"},
		{[]string{"release", "发布", "view", "查看"}, "release +view"},
		// Repo
		{[]string{"repo", "仓库", "info", "信息"}, "repo +info"},
		{[]string{"repo", "仓库", "create", "新建", "创建"}, "repo +create"},
		{[]string{"repo", "仓库", "readme"}, "repo +readme"},
		{[]string{"repo", "仓库", "fork", "复刻"}, "repo +fork"},
		{[]string{"repo", "仓库", "list", "列"}, "repo +list"},
		// Auth
		{[]string{"auth", "login", "登录", "认证"}, "auth login"},
		{[]string{"auth", "status", "状态"}, "auth status"},
		// Snippet
		{[]string{"snippet", "片段", "代码片段", "list"}, "snippet +list"},
		{[]string{"snippet", "片段", "代码片段", "create", "新建", "保存"}, "snippet +create"},
		{[]string{"snippet", "片段", "代码片段", "view", "查看"}, "snippet +view"},
		{[]string{"snippet", "片段", "代码片段", "delete", "删除"}, "snippet +delete"},
		// Notification
		{[]string{"notification", "通知", "消息", "list"}, "notification +list"},
		{[]string{"notification", "通知", "read", "已读"}, "notification +read"},
		{[]string{"notification", "通知", "delete", "删除"}, "notification +delete"},
		// Member
		{[]string{"member", "成员", "list"}, "member +list"},
		{[]string{"member", "成员", "add", "添加"}, "member +add"},
		{[]string{"member", "成员", "invite", "邀请"}, "member +invite-link"},
		// Branch
		{[]string{"branch", "分支", "list"}, "branch +list"},
		{[]string{"branch", "分支", "create", "新建"}, "branch +create"},
		// CI
		{[]string{"ci", "构建", "流水线", "list"}, "ci +list"},
		{[]string{"ci", "构建", "log", "日志"}, "ci +logs"},
		// Search
		{[]string{"search", "搜索", "查找", "repos"}, "search +repos"},
		{[]string{"search", "搜索", "查找", "user", "用户"}, "search +users"},
		// Milestone
		{[]string{"milestone", "里程碑", "list"}, "milestone +list"},
		{[]string{"milestone", "里程碑", "create", "新建"}, "milestone +create"},
		// Webhook
		{[]string{"webhook", "钩子", "list"}, "webhook +list"},
		{[]string{"webhook", "钩子", "create", "新建"}, "webhook +create"},
	}

	bestMatch := ""
	bestScore := 0
	for _, r := range rules {
		score := 0
		for _, kw := range r.keywords {
			if strings.Contains(input, kw) {
				score++
			}
		}
		if score >= 2 && score > bestScore {
			bestScore = score
			bestMatch = r.cmd
		}
	}

	// 降级：只匹配1个关键词
	if bestMatch == "" {
		for _, r := range rules {
			for _, kw := range r.keywords {
				if strings.Contains(input, kw) {
					bestMatch = r.cmd
					break
				}
			}
			if bestMatch != "" {
				break
			}
		}
	}

	return bestMatch
}
