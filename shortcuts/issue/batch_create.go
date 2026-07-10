package issue

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newBatchCreateShortcut(tr *i18n.Translator) *common.Shortcut {
	flags := []common.Flag{
		{Name: "from", Short: "f", Usage: tr.T("flag.issue.batch_create.csv"), Required: true},
		{Name: "print-schema", Usage: tr.T("flag.issue.batch_create.print_schema"), Bool: true, Default: "false"},
	}
	flags = append(flags, batchRuntimeFlags(tr)...)
	return &common.Shortcut{
		Name:        "batch-create",
		Description: tr.T("cmd.issue.batch_create.short"),
		Flags:       flags,
		Run:         runBatchCreate,
	}
}

func runBatchCreate(ctx *common.RuntimeContext) error {
	if parseBool(ctx.Arg("print-schema")) {
		fmt.Println("title,body,assignee,milestone,label,priority")
		return nil
	}
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	opts := parseBatchOptions(ctx)
	if !opts.DryRun && !opts.Confirm && os.Getenv("GITLINK_CONFIRM_BATCH") != "true" {
		return fmt.Errorf("请添加 --confirm 确认执行，或使用 --dry-run 预览。也可设置 GITLINK_CONFIRM_BATCH=true 环境变量跳过此检查")
	}

	headers, rows, err := ReadCSV(ctx.Arg("from"))
	if err != nil {
		return err
	}

	titleCol := FindColumn(headers, "title", "subject")
	if titleCol == -1 {
		return fmt.Errorf("CSV 缺少标题列（title/subject）")
	}
	bodyCol := FindColumn(headers, "body", "description")
	assigneeCol := FindColumn(headers, "assignee", "assignee_id")
	milestoneCol := FindColumn(headers, "milestone", "fixed_version_id", "milestone_id")
	labelCol := FindColumn(headers, "label", "labels")
	priorityCol := FindColumn(headers, "priority", "priority_id")

	truncated := false
	if opts.MaxItems > 0 && len(rows) > opts.MaxItems {
		fmt.Fprintf(os.Stderr, "警告：CSV 有 %d 行，已按 --max=%d 截断\n", len(rows), opts.MaxItems)
		rows = rows[:opts.MaxItems]
		truncated = true
	}

	summary := &BatchSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     opts.DryRun,
		Total:      len(rows),
		Truncated:  truncated,
		Results:    make([]BatchResult, 0, len(rows)),
	}

	for i, row := range rows {
		result := BatchResult{ID: fmt.Sprintf("row-%d", i+1), Action: "create"}
		if opts.DryRun {
			result.Status = "dry_run"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}
		if opts.DelayMs > 0 && i > 0 {
			time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
		}

		if err := createIssueFromRow(ctx, row, titleCol, bodyCol, assigneeCol, milestoneCol, labelCol, priorityCol); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "success"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d issue(s) failed to create", summary.Failed, summary.Total)
	}
	if truncated {
		return fmt.Errorf("结果已截断，仅处理了 %d 个 Issue", summary.Total)
	}
	return nil
}

func createIssueFromRow(ctx *common.RuntimeContext, row []string, titleCol, bodyCol, assigneeCol, milestoneCol, labelCol, priorityCol int) error {
	title := getCell(row, titleCol)
	if title == "" {
		return fmt.Errorf("empty title")
	}

	body := map[string]interface{}{"subject": title, "status_id": 1, "priority_id": 2, "done_ratio": 0}
	if desc := getCell(row, bodyCol); desc != "" {
		body["description"] = desc
	}
	if assignee := getCell(row, assigneeCol); assignee != "" {
		id, err := ResolveUserID(ctx, assignee)
		if err != nil {
			return fmt.Errorf("assignee %q: %w", assignee, err)
		}
		body["assigner_ids"] = []int{id}
	}
	if milestone := getCell(row, milestoneCol); milestone != "" {
		if id, err := strconv.Atoi(milestone); err == nil {
			body["milestone_id"] = id
		} else {
			id, err := ResolveMilestoneID(ctx, milestone)
			if err != nil {
				return fmt.Errorf("milestone %q: %w", milestone, err)
			}
			body["milestone_id"] = id
		}
	}
	if labels := getCell(row, labelCol); labels != "" {
		labelIDs, err := resolveLabelArgs(ctx, labels, "")
		if err != nil {
			return fmt.Errorf("label %q: %w", labels, err)
		}
		body["issue_tag_ids"] = labelIDs
	}
	if pri := getCell(row, priorityCol); pri != "" {
		pid, err := strconv.Atoi(pri)
		if err != nil {
			return fmt.Errorf("priority %q: must be a numeric priority_id", pri)
		}
		body["priority_id"] = pid
	}
	_, err := ctx.CallAPI("POST", v1RepoPath(ctx)+"/issues", body)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	return nil
}

func getCell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}
