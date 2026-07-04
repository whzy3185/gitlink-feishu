package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NLCommand 自然语言命令路由
// 用法: gitlink-cli do "列出我的issue"
// AI/关键词匹配 → 推荐或执行对应命令

func newDoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `do "自然语言描述"`,
		Short: "Natural language command (e.g. do \"list my issues\")",
		Long:  `用自然语言描述你想做的事，自动匹配对应命令。例如: gitlink-cli do "列出issue"`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := strings.ToLower(strings.Join(args, " "))
			matched := matchNL(input)
			if matched == "" {
				fmt.Println("未识别的命令。试试这些关键词：")
				fmt.Println("  issue/pr/label/wiki/release/repo/auth/snippet/notification")
				fmt.Println("  list/create/view/delete/merge/close")
				fmt.Println("示例: gitlink-cli do \"列出issue\"")
				os.Exit(1)
			}
			fmt.Printf("💡 匹配到命令: %s\n", matched)
			fmt.Printf("执行: gitlink-cli %s\n", matched)
			// 提示用户直接执行
			fmt.Println("\n请运行上述命令（或加上参数）：")
			fmt.Printf("  gitlink-cli %s\n", matched)
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
		{[]string{"issue", "疑修", "问题", "list", "列", "查看"}, "issue +list"},
		{[]string{"issue", "create", "新建", "创建", "提"}, "issue +create"},
		{[]string{"pr", "pull", "合并请求", "merge", "list"}, "pr +list"},
		{[]string{"label", "tag", "标签", "list"}, "label +list"},
		{[]string{"label", "tag", "标签", "create", "新建", "创建"}, "label +create"},
		{[]string{"wiki", "文档", "知识库", "list"}, "wiki +list"},
		{[]string{"wiki", "文档", "create", "新建", "创建"}, "wiki +create"},
		{[]string{"release", "发布", "版本", "list"}, "release +list"},
		{[]string{"release", "发布", "create", "新建"}, "release +create"},
		{[]string{"repo", "仓库", "info", "信息"}, "repo +info"},
		{[]string{"repo", "仓库", "create", "新建", "创建"}, "repo +create"},
		{[]string{"auth", "login", "登录", "认证"}, "auth login"},
		{[]string{"auth", "status", "状态"}, "auth status"},
		{[]string{"snippet", "片段", "代码", "list"}, "snippet +list"},
		{[]string{"snippet", "片段", "代码", "create", "新建"}, "snippet +create"},
		{[]string{"notification", "通知", "消息", "list"}, "notification +list"},
		{[]string{"member", "成员", "list"}, "member +list"},
		{[]string{"branch", "分支", "list"}, "branch +list"},
		{[]string{"ci", "构建", "流水线", "list"}, "ci +list"},
		{[]string{"search", "搜索", "查找"}, "search +repos"},
		{[]string{"milestone", "里程碑", "list"}, "milestone +list"},
		{[]string{"webhook", "钩子", "list"}, "webhook +list"},
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
		// 至少匹配2个关键词（避免误匹配）
		if score >= 2 && score > bestScore {
			bestScore = score
			bestMatch = r.cmd
		}
	}

	// 降级：只匹配1个但有动作词
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
