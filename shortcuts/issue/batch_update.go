package issue

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var updateFieldMapping = map[string]string{
	"title":     "subject",
	"body":      "description",
	"state":     "status_id",
	"assignee":  "assigner_ids",
	"milestone": "milestone_id",
	"label":     "issue_tag_ids",
	"priority":  "priority_id",
}

func newBatchUpdateShortcut(tr *i18n.Translator) *common.Shortcut {
	flags := []common.Flag{
		// --ids 统一模式参数
		{Name: "ids", Short: "i", Usage: tr.T("flag.issue.batch_update.ids"), Required: false},
		{Name: "status", Short: "s", Usage: tr.T("flag.issue.batch_update.status")},
		{Name: "priority", Short: "p", Usage: tr.T("flag.issue.batch_update.priority")},
		{Name: "milestone", Short: "m", Usage: tr.T("flag.issue.batch_update.milestone")},
		{Name: "labels", Short: "l", Usage: tr.T("flag.issue.batch_update.tags")},
		{Name: "assignees", Short: "a", Usage: tr.T("flag.issue.batch_update.assignees")},
		// CSV 模式参数
		{Name: "from", Usage: tr.T("flag.issue.batch_update.csv")},
	}
	flags = append(flags, batchRuntimeFlags(tr)...)
	return &common.Shortcut{
		Name:        "batch-update",
		Description: tr.T("cmd.issue.batch_update.short"),
		Flags:       flags,
		Run:         runBatchUpdate,
	}
}

func runBatchUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	if csvPath := ctx.Arg("from"); csvPath != "" {
		return runBatchUpdateCSV(ctx, csvPath)
	}
	return runBatchUpdateIDs(ctx)
}

func runBatchUpdateCSV(ctx *common.RuntimeContext, csvPath string) error {
	headers, rows, err := ReadCSV(csvPath)
	if err != nil {
		return err
	}

	numberCol := FindColumn(headers, "number", "issue_number", "project_issues_index")
	if numberCol == -1 {
		return fmt.Errorf("CSV 缺少编号列（number/issue_number/project_issues_index）")
	}

	numbers := make([]string, 0, len(rows))
	rowByNumber := make(map[string][]string)
	for _, row := range rows {
		if numberCol < len(row) {
			n := strings.TrimSpace(row[numberCol])
			if n != "" {
				if _, exists := rowByNumber[n]; !exists {
					numbers = append(numbers, n)
				} else {
					fmt.Fprintf(os.Stderr, "警告：issue #%s 在 CSV 中出现多次，仅使用最后一次的数据\n", n)
				}
				rowByNumber[n] = row
			}
		}
	}

	opts := parseBatchOptions(ctx)

	updateFn := func(c *common.RuntimeContext, number string) error {
		row, ok := rowByNumber[number]
		if !ok {
			return fmt.Errorf("no CSV data for issue #%s", number)
		}
		return applyIssueUpdates(c, number, row, headers)
	}
	_, err = RunBatch(ctx, numbers, "update", opts, updateFn)
	return err
}

func runBatchUpdateIDs(ctx *common.RuntimeContext) error {
	idsValue, err := ctx.RequireArg("ids")
	if err != nil {
		return err
	}
	ids, err := parseCommaInts(idsValue)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"ids": ids,
	}

	if s := ctx.Arg("status"); s != "" {
		statusID, err := normalizeIssueStatus(s)
		if err != nil {
			return err
		}
		body["status_id"] = statusID
	}
	if p := ctx.Arg("priority"); p != "" {
		pid, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("无效的优先级 ID: %s", p)
		}
		body["priority_id"] = pid
	}
	if m := ctx.Arg("milestone"); m != "" {
		mid, err := strconv.Atoi(m)
		if err != nil {
			return fmt.Errorf("无效的里程碑 ID: %s", m)
		}
		body["milestone_id"] = mid
	}
	if l := ctx.Arg("labels"); l != "" {
		labelIDs, err := parseCommaInts(l)
		if err != nil {
			return fmt.Errorf("无效的标签 ID: %w", err)
		}
		body["issue_tag_ids"] = labelIDs
	}
	if a := ctx.Arg("assignees"); a != "" {
		assigneeIDs, err := parseCommaInts(a)
		if err != nil {
			return fmt.Errorf("无效的负责人 ID: %w", err)
		}
		body["assigner_ids"] = assigneeIDs
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	if dryRun {
		return ctx.OutputData(map[string]interface{}{
			"action":  "batch-update",
			"dry_run": true,
			"ids":     ids,
			"changes": body,
		})
	}

	env, err := ctx.CallAPI("PATCH", v1RepoPath(ctx)+"/issues/batch_update", body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func applyIssueUpdates(ctx *common.RuntimeContext, number string, row []string, headers []string) error {
	current, err := fetchIssueData(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	body := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
	}

	for i, colName := range headers {
		colName = strings.ToLower(strings.TrimSpace(colName))
		apiField, ok := updateFieldMapping[colName]
		if !ok || i >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[i])
		if val == "" {
			continue
		}

		switch apiField {
		case "subject":
			body["subject"] = val
		case "description":
			body["description"] = val
		case "status_id":
			sid, err := normalizeIssueStatus(val)
			if err != nil {
				return fmt.Errorf("issue #%s state %q: %w", number, val, err)
			}
			body["status_id"] = sid
		case "assigner_ids":
			id, err := ResolveUserID(ctx, val)
			if err != nil {
				return fmt.Errorf("issue #%s assignee %q: %w", number, val, err)
			}
			body["assigner_ids"] = []int{id}
		case "milestone_id":
			if id, err := strconv.Atoi(val); err == nil {
				body["milestone_id"] = id
			} else {
				id, err := ResolveMilestoneID(ctx, val)
				if err != nil {
					return fmt.Errorf("issue #%s milestone %q: %w", number, val, err)
				}
				body["milestone_id"] = id
			}
		case "issue_tag_ids":
			labelIDs, err := resolveLabelArgs(ctx, val, "")
			if err != nil {
				return fmt.Errorf("issue #%s label %q: %w", number, val, err)
			}
			body["issue_tag_ids"] = labelIDs
		case "priority_id":
			pid, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("issue #%s priority %q: must be a numeric priority_id", number, val)
			}
			body["priority_id"] = pid
		}
	}
	_, err = ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}
	return nil
}
