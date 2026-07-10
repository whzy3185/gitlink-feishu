package issue

import (
	"fmt"
	"os"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newBatchAssignShortcut(tr *i18n.Translator) *common.Shortcut {
	flags := []common.Flag{
		{Name: "numbers", Short: "n", Usage: tr.T("flag.issue.batch_assign.numbers")},
		{Name: "from", Usage: tr.T("flag.issue.batch_assign.csv")},
		{Name: "search", Usage: tr.T("flag.issue.batch.search")},
		{Name: "state", Usage: tr.T("flag.issue.batch.state")},
		{Name: "assignee", Short: "a", Usage: tr.T("flag.issue.batch_assign.assignee")},
	}
	flags = append(flags, batchRuntimeFlags(tr)...)
	return &common.Shortcut{
		Name:        "batch-assign",
		Description: tr.T("cmd.issue.batch_assign.short"),
		Flags:       flags,
		Run:         runBatchAssign,
	}
}

func runBatchAssign(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	csvPath := ctx.Arg("from")
	opts := parseBatchOptions(ctx)

	if csvPath != "" {
		headers, rows, err := ReadCSV(csvPath)
		if err != nil {
			return err
		}
		numberCol := FindColumn(headers, "number", "issue_number", "project_issues_index")
		if numberCol == -1 {
			return fmt.Errorf("CSV 缺少编号列（number/issue_number/project_issues_index）")
		}
		assigneeCol := FindColumn(headers, "assignee", "assignee_id", "assigned_to_id")
		if assigneeCol == -1 {
			return fmt.Errorf("CSV 缺少经办人列（assignee/assignee_id/assigned_to_id）")
		}

		// 暂存原始经办人字符串，resolve 推迟到逐条 callback 内执行，
		// 这样 --dry-run 不会触发任何 GET /users/search。
		assigneeMap := make(map[string]string, len(rows))
		numbers := make([]string, 0, len(rows))
		numberSeen := make(map[string]bool)
		for _, row := range rows {
			if numberCol >= len(row) || assigneeCol >= len(row) {
				continue
			}
			num := strings.TrimSpace(row[numberCol])
			arg := strings.TrimSpace(row[assigneeCol])
			if num == "" || arg == "" {
				continue
			}
			if _, exists := assigneeMap[num]; exists {
				fmt.Fprintf(os.Stderr, "警告：issue #%s 在 CSV 中出现多次，仅使用最后一次的经办人\n", num)
			}
			assigneeMap[num] = arg
			if !numberSeen[num] {
				numbers = append(numbers, num)
				numberSeen[num] = true
			}
		}
		if len(assigneeMap) == 0 {
			return fmt.Errorf("no valid entries in CSV")
		}

		assignFn := func(c *common.RuntimeContext, number string) error {
			arg := assigneeMap[number]
			aid, err := ResolveUserID(c, arg)
			if err != nil {
				return fmt.Errorf("assignee %q: %w", arg, err)
			}
			return assignIssue(c, number, aid)
		}
		_, err = RunBatch(ctx, numbers, "assign", opts, assignFn)
		return err
	}

	numbers, err := ResolveIssueNumbers(ctx, ctx.Arg("numbers"), "", ctx.Arg("search"))
	if err != nil {
		return err
	}

	assigneeArg := ctx.Arg("assignee")
	if assigneeArg == "" {
		return fmt.Errorf("--assignee is required in uniform mode")
	}

	// 把 ResolveUserID 推迟到逐条 callback，使 --dry-run 不会调用 GET /users/search；
	// 解析失败改为按条记录在 BatchResult.Error 中。
	assigneeFn := func(c *common.RuntimeContext, number string) error {
		aid, err := ResolveUserID(c, assigneeArg)
		if err != nil {
			return fmt.Errorf("assignee %q: %w", assigneeArg, err)
		}
		return assignIssue(c, number, aid)
	}
	_, err = RunBatch(ctx, numbers, "assign", opts, assigneeFn)
	return err
}

func assignIssue(ctx *common.RuntimeContext, number string, assigneeID int) error {
	return patchIssue(ctx, number, map[string]interface{}{"assigner_ids": []int{assigneeID}}, "assign")
}

