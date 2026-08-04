package issue

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	defaultBatchMaxItems = 100
	defaultBatchDelayMs  = 0
	closeIssueStatusID   = closedIssueStatusID
)

type BatchOptions struct {
	DryRun   bool
	Confirm  bool
	MaxItems int
	DelayMs  int
}

type BatchResult struct {
	Number string `json:"number"`
	Action string `json:"action"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type BatchSummary struct {
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
	DryRun    bool          `json:"dry_run"`
	Truncated bool          `json:"truncated"`
	Results   []BatchResult `json:"results"`
}

// RunBatch applies a bounded operation and requires an explicit confirmation
// for mutations. Dry-run mode never invokes the operation callback.
func RunBatch(ctx *common.RuntimeContext, numbers []string, action string, opts BatchOptions, fn func(*common.RuntimeContext, string) error) (BatchSummary, error) {
	if !opts.DryRun && !opts.Confirm {
		return BatchSummary{}, fmt.Errorf("batch mutation requires --confirm")
	}
	limit := opts.MaxItems
	if limit <= 0 {
		limit = defaultBatchMaxItems
	}
	truncated := len(numbers) > limit
	if truncated {
		numbers = numbers[:limit]
	}
	summary := BatchSummary{Total: len(numbers), DryRun: opts.DryRun, Truncated: truncated, Results: make([]BatchResult, 0, len(numbers))}
	for i, number := range numbers {
		result := BatchResult{Number: number, Action: action}
		if opts.DryRun {
			result.Status = "dry_run"
			summary.Succeeded++
		} else if err := fn(ctx, number); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "success"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
		if opts.DelayMs > 0 && i+1 < len(numbers) {
			time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)
		}
	}
	if summary.Failed > 0 {
		return summary, fmt.Errorf("%d of %d batch items failed", summary.Failed, summary.Total)
	}
	if truncated {
		return summary, fmt.Errorf("batch truncated to %d items", limit)
	}
	return summary, nil
}

func parseBatchOptions(ctx *common.RuntimeContext) BatchOptions {
	maxItems := parsePositiveInt(ctx.Arg("max"), defaultBatchMaxItems)
	delayMs := parseNonNegativeInt(ctx.Arg("delay"), defaultBatchDelayMs)
	return BatchOptions{
		DryRun:   parseBool(ctx.Arg("dry-run")),
		Confirm:  parseBool(ctx.Arg("confirm")),
		MaxItems: maxItems,
		DelayMs:  delayMs,
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseNonNegativeInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func patchIssue(ctx *common.RuntimeContext, number string, extraFields map[string]interface{}, action string) error {
	current, err := fetchExistingIssue(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue before %s: %w", action, err)
	}
	payload := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
	}
	for key, value := range extraFields {
		payload[key] = value
	}
	if _, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), payload); err != nil {
		return fmt.Errorf("%s issue: %w", action, err)
	}
	return nil
}

func mergeLabelIDs(existing, additions []int) []int {
	if existing == nil && additions == nil {
		return nil
	}
	result := append([]int{}, existing...)
	seen := make(map[int]bool, len(result))
	for _, id := range result {
		seen[id] = true
	}
	for _, id := range additions {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func removeLabelIDs(existing, removals []int) []int {
	removed := make(map[int]bool, len(removals))
	for _, id := range removals {
		removed[id] = true
	}
	result := make([]int, 0, len(existing))
	for _, id := range existing {
		if !removed[id] {
			result = append(result, id)
		}
	}
	return result
}

func ReadCSV(path string) ([]string, [][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, nil
	}
	return records[0], records[1:], nil
}

func FindColumn(headers []string, names ...string) int {
	for index, header := range headers {
		for _, name := range names {
			if strings.EqualFold(strings.TrimSpace(header), name) {
				return index
			}
		}
	}
	return -1
}

func ResolveIssueNumbers(_ *common.RuntimeContext, inline, csvPath, _ string) ([]string, error) {
	return collectIssueNumbers(inline, csvPath)
}
