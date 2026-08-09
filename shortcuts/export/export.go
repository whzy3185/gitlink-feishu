package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns data export shortcuts for GitLink.
//
// The export domain provides commands for exporting repository data
// (issues, pull requests, contributors) to CSV or JSON files for
// offline analysis and reporting.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "issues",
			Description: "导出仓库 Issue 列表为 CSV 或 JSON 文件",
			Flags: []common.Flag{
				{Name: "format", Short: "f", Usage: "输出格式: csv / json", Default: "csv"},
				{Name: "output", Short: "o", Usage: "输出文件路径（默认: issues.csv）", Default: "issues.csv"},
				{Name: "state", Short: "s", Usage: "状态过滤: open / closed / all", Default: "all"},
				{Name: "page", Short: "p", Usage: "起始页", Default: "1"},
				{Name: "limit", Short: "l", Usage: "每页数量", Default: "50"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("state", ctx.Arg("state"))
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				apiPath := fmt.Sprintf("/v1/%s/%s/issues", ctx.Owner, ctx.Repo)
				items, err := ctx.PaginateAll(apiPath, q)
				if err != nil {
					return err
				}
				return writeExport(ctx.Arg("format"), ctx.Arg("output"), items, issueCSVHeader, issueCSVRow)
			},
		},
		{
			Name:        "prs",
			Description: "导出仓库 PR 列表为 CSV 或 JSON 文件",
			Flags: []common.Flag{
				{Name: "format", Short: "f", Usage: "输出格式: csv / json", Default: "csv"},
				{Name: "output", Short: "o", Usage: "输出文件路径", Default: "prs.csv"},
				{Name: "state", Short: "s", Usage: "状态过滤", Default: "all"},
				{Name: "page", Short: "p", Usage: "起始页", Default: "1"},
				{Name: "limit", Short: "l", Usage: "每页数量", Default: "50"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				q.Set("state", ctx.Arg("state"))
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				apiPath := fmt.Sprintf("/v1/%s/%s/pulls", ctx.Owner, ctx.Repo)
				items, err := ctx.PaginateAll(apiPath, q)
				if err != nil {
					return err
				}
				return writeExport(ctx.Arg("format"), ctx.Arg("output"), items, prCSVHeader, prCSVRow)
			},
		},
		{
			Name:        "contributors",
			Description: "导出贡献者统计为 CSV 或 JSON 文件",
			Flags: []common.Flag{
				{Name: "format", Short: "f", Usage: "输出格式: csv / json", Default: "csv"},
				{Name: "output", Short: "o", Usage: "输出文件路径", Default: "contributors.csv"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				apiPath := fmt.Sprintf("/%s/%s/contributors", ctx.Owner, ctx.Repo)
				items, err := ctx.PaginateAll(apiPath, url.Values{})
				if err != nil {
					return err
				}
				return writeExport(ctx.Arg("format"), ctx.Arg("output"), items, contributorCSVHeader, contributorCSVRow)
			},
		},
	}
}

// --- CSV headers ---

var issueCSVHeader = []string{"id", "title", "state", "created_at"}
var prCSVHeader = []string{"id", "title", "state", "created_at"}
var contributorCSVHeader = []string{"id", "login", "contributions"}

// --- CSV row extractors ---

func issueCSVRow(m map[string]interface{}) []string {
	return []string{
		fmt.Sprint(m["id"]),
		fmt.Sprint(m["subject"]),
		fmt.Sprint(m["status"]),
		fmt.Sprint(m["created_at"]),
	}
}

func prCSVRow(m map[string]interface{}) []string {
	return []string{
		fmt.Sprint(m["id"]),
		fmt.Sprint(m["title"]),
		fmt.Sprint(m["status"]),
		fmt.Sprint(m["created_at"]),
	}
}

func contributorCSVRow(m map[string]interface{}) []string {
	return []string{
		fmt.Sprint(m["id"]),
		fmt.Sprint(m["login"]),
		fmt.Sprint(m["contributions"]),
	}
}

// --- Export writers ---

type csvRowFunc func(map[string]interface{}) []string

func writeExport(format, path string, items []json.RawMessage, header []string, rowFn csvRowFunc) error {
	switch format {
	case "csv":
		return writeCSV(path, items, header, rowFn)
	case "json":
		return writeJSON(path, items)
	default:
		return fmt.Errorf("不支持的格式: %s（可选: csv, json）", format)
	}
}

func writeCSV(path string, items []json.RawMessage, header []string, rowFn csvRowFunc) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(header); err != nil {
		return err
	}
	for _, item := range items {
		var m map[string]interface{}
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		if err := w.Write(rowFn(m)); err != nil {
			return err
		}
	}
	fmt.Printf("已导出 %d 条记录到 %s\n", len(items), path)
	return nil
}

func writeJSON(path string, items []json.RawMessage) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	fmt.Printf("已导出 %d 条记录到 %s\n", len(items), path)
	return nil
}
