package issue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func newBatchLabelShortcut(tr *i18n.Translator) *common.Shortcut {
	flags := []common.Flag{
		{Name: "numbers", Short: "n", Usage: tr.T("flag.issue.batch_label.numbers")},
		{Name: "from", Usage: tr.T("flag.issue.batch_label.csv")},
		{Name: "search", Usage: tr.T("flag.issue.batch.search")},
		{Name: "state", Usage: tr.T("flag.issue.batch.state")},
		{Name: "action", Short: "a", Usage: tr.T("flag.issue.batch_label.action"), Required: true},
		{Name: "labels", Short: "l", Usage: tr.T("flag.issue.batch_label.labels")},
		{Name: "label-ids", Usage: tr.T("flag.issue.batch_label.label_ids")},
	}
	flags = append(flags, batchRuntimeFlags(tr)...)
	return &common.Shortcut{
		Name:        "batch-label",
		Description: tr.T("cmd.issue.batch_label.short"),
		Flags:       flags,
		Run:         runBatchLabel,
	}
}

func runBatchLabel(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	action := strings.ToLower(strings.TrimSpace(ctx.Arg("action")))
	switch action {
	case "add", "remove", "set":
	default:
		return fmt.Errorf("invalid --action %q: must be add, remove, or set", action)
	}

	labelNames := ctx.Arg("labels")
	labelIDsStr := ctx.Arg("label-ids")
	if labelNames == "" && labelIDsStr == "" {
		return fmt.Errorf("either --labels or --label-ids is required")
	}
	if labelNames != "" && labelIDsStr != "" {
		return fmt.Errorf("--labels and --label-ids are mutually exclusive")
	}

	numbers, err := ResolveIssueNumbers(ctx, ctx.Arg("numbers"), ctx.Arg("from"), ctx.Arg("search"))
	if err != nil {
		return err
	}

	opts := parseBatchOptions(ctx)

	// 把 label 名称解析推迟到逐条 callback，使 --dry-run 不会触发
	// 用于预热 label 缓存的 API 调用（如 GET /labels）。
	labelFn := func(c *common.RuntimeContext, number string) error {
		labelIDs, err := resolveLabelArgs(c, labelNames, labelIDsStr)
		if err != nil {
			return err
		}
		return manageIssueLabels(c, number, action, labelIDs)
	}
	_, err = RunBatch(ctx, numbers, "label-"+action, opts, labelFn)
	return err
}

func manageIssueLabels(ctx *common.RuntimeContext, number string, action string, newIDs []int) error {
	current, err := fetchIssueData(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	existingIDs := current.LabelIDs
	if existingIDs == nil {
		existingIDs = []int{}
	}

	var finalIDs []int
	switch action {
	case "add":
		finalIDs = mergeLabelIDs(existingIDs, newIDs)
	case "remove":
		finalIDs = removeLabelIDs(existingIDs, newIDs)
	case "set":
		finalIDs = newIDs
	}

	body := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
		"issue_tag_ids": finalIDs,
	}
	_, err = ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body)
	if err != nil {
		return fmt.Errorf("update labels: %w", err)
	}
	return nil
}

func mergeLabelIDs(existing, new []int) []int {
	has := map[int]bool{}
	for _, id := range existing {
		has[id] = true
	}
	for _, id := range new {
		if !has[id] {
			existing = append(existing, id)
			has[id] = true
		}
	}
	return existing
}

func removeLabelIDs(existing, toRemove []int) []int {
	remove := map[int]bool{}
	for _, id := range toRemove {
		remove[id] = true
	}
	result := make([]int, 0, len(existing))
	for _, id := range existing {
		if !remove[id] {
			result = append(result, id)
		}
	}
	return result
}

func resolveLabelArgs(ctx *common.RuntimeContext, names, idsStr string) ([]int, error) {
	if idsStr != "" {
		parts := strings.Split(idsStr, ",")
		ids := make([]int, 0, len(parts))
		for _, p := range parts {
			id, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return nil, fmt.Errorf("invalid label ID %q: %w", p, err)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	parts := strings.Split(names, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		id, err := ResolveLabelID(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("label %q: %w", name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
