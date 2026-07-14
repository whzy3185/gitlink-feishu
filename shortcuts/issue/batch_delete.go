package issue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newBatchDeleteShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-delete",
		Description: tr.T("cmd.issue.batch_delete.short"),
		Flags: []common.Flag{
			{Name: "ids", Short: "i", Usage: tr.T("flag.issue.batch_delete.ids"), Required: true},
			{Name: "dry-run", Usage: tr.T("flag.issue.batch_delete.dry_run"), Bool: true, Default: "false"},
			{Name: "confirm", Usage: tr.T("flag.issue.batch_delete.confirm"), Bool: true, Default: "false"},
		},
		Run: runBatchDelete,
	}
}

func runBatchDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	idsValue, err := ctx.RequireArg("ids")
	if err != nil {
		return err
	}
	ids, err := parseCommaInts(idsValue)
	if err != nil {
		return err
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	if dryRun {
		return ctx.OutputData(map[string]interface{}{
			"action":  "batch-delete",
			"dry_run": true,
			"ids":     ids,
			"message": "使用 --confirm 执行实际删除",
		})
	}

	if !parseBool(ctx.Arg("confirm")) {
		return ctx.OutputData(map[string]interface{}{
			"action":  "batch-delete",
			"dry_run": true,
			"ids":     ids,
			"message": "批量删除是危险操作，请添加 --confirm 标志确认删除",
		})
	}

	_, err = ctx.CallAPI("DELETE", v1RepoPath(ctx)+"/issues/batch_destroy", map[string]interface{}{
		"ids": ids,
	})
	if err != nil {
		return err
	}

	return ctx.OutputData(map[string]interface{}{
		"message": fmt.Sprintf("成功删除 %d 个 issue", len(ids)),
		"ids":     ids,
	})
}

// parseCommaInts 把逗号分隔的字符串解析为唯一整数切片。
func parseCommaInts(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	ids := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("无效的 ID: %q", p)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("请提供至少一个 ID")
	}
	return ids, nil
}
