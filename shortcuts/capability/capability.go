package capability

import (
	"fmt"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/capability"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// SharedRegistry is the global capability registry used across the CLI.
// It is initialized in register.go and used by help annotations.
var SharedRegistry = capability.NewRegistry()

// resultRow is one row of the capability table, also the JSON shape produced
// when --format json is passed.
type resultRow struct {
	Domain     string `json:"domain"`
	Status     string `json:"status"`
	StatusText string `json:"status_text"`
	Message    string `json:"message,omitempty"`
}

// buildResultRows turns the probe results into the ordered, structured form
// used by both the human-readable table and the structured envelope.
func buildResultRows(results map[string]*capability.DomainStatus) []resultRow {
	domains := []string{
		"label", "notification", "pm", "wiki", "pipeline",
		"webhook", "member", "milestone", "export", "search", "workflow",
	}
	rows := make([]resultRow, 0, len(domains))
	for _, d := range domains {
		ds, ok := results[d]
		if !ok || ds == nil {
			rows = append(rows, resultRow{Domain: d, Status: "unknown", StatusText: "skipped", Message: "缺少 owner/repo 上下文，未探测"})
			continue
		}
		detail := ds.Message
		if detail == "" && ds.Status == capability.StatusAvailable {
			detail = "API 正常响应"
		}
		rows = append(rows, resultRow{
			Domain:     d,
			Status:     statusString(ds.Status),
			StatusText: statusText(ds.Status),
			Message:    detail,
		})
	}
	return rows
}

func statusString(s capability.Status) string {
	switch s {
	case capability.StatusAvailable:
		return "available"
	case capability.StatusUnavailable:
		return "unavailable"
	case capability.StatusError:
		return "error"
	default:
		return "unknown"
	}
}

func statusText(s capability.Status) string {
	switch s {
	case capability.StatusAvailable:
		return "可用 ✓"
	case capability.StatusUnavailable:
		return "不可用 ✗"
	case capability.StatusError:
		return "错误 ✗"
	default:
		return "未知 ?"
	}
}

func statusEmoji(s string) string {
	switch s {
	case "available":
		return "✓"
	case "unavailable", "error":
		return "✗"
	default:
		return "?"
	}
}

// ensure output stays referenced for future structured extensions.
var _ = output.SuccessEnvelope

// Shortcuts returns the capability management shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "check",
			Description: "探测后端 API 能力，检查各模块是否可用",
			Long: `向 GitLink 后端发送探测请求，检查各命令模块依赖的 API 是否就绪。

探测结果会缓存 24 小时。之后运行 capability +list 查看缓存结果。

需要 owner/repo 上下文的模块（如 label、webhook、member 等）会自动从
git remote 推断，或通过 --owner/--repo 指定。`,
			Flags: []common.Flag{},
			Run: func(ctx *common.RuntimeContext) error {
				// Resolve owner/repo for repo-dependent probes
				owner, repo := ctx.Owner, ctx.Repo
				if owner == "" || repo == "" {
					_ = ctx.ResolveOwnerRepo()
					owner, repo = ctx.Owner, ctx.Repo
				}

				results := SharedRegistry.Refresh(ctx.Client, owner, repo)

				// When the user explicitly asks for a structured format, route
				// through the output envelope (so capability +check plays nice
				// with --format json/table/yaml and AI Agents). Empty format =
				// the human-readable table (the historical default).
				if cmdutil.Format != "" {
					return ctx.OutputData(buildResultRows(results))
				}

				// Print results table
				fmt.Println("API 后端能力探测结果:")
				fmt.Println()
				fmt.Printf("  %-4s %-15s %-12s %s\n", "", "模块", "状态", "说明")
				fmt.Println("  " + "---- --------------- ------------ ------------------------------")
				for _, row := range buildResultRows(results) {
					icon := statusEmoji(row.Status)
					fmt.Printf("  %-4s %-15s %-12s %s\n", icon, row.Domain, row.StatusText, row.Message)
				}
				fmt.Println()
				fmt.Println("提示: 不可用的模块会在 --help 中标记 ⚠，调用时会显示中文错误指引。")
				fmt.Println("缓存位置: ~/.config/gitlink-cli/capabilities.json（24 小时有效）")
				return nil
			},
		},
		{
			Name:        "list",
			Description: "查看已缓存的 API 能力探测结果",
			Flags:       []common.Flag{},
			Run: func(ctx *common.RuntimeContext) error {
				fmt.Print(SharedRegistry.Summary())
				if SharedRegistry.IsStale() {
					fmt.Println("\n⚠ 缓存已过期（超过 24 小时），运行 capability +check 刷新。")
				}
				return nil
			},
		},
	}
}
