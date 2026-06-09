package issue

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const closedIssueStatusID = 5

type batchCloseResult struct {
	Number string `json:"number" yaml:"number"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchCloseSummary struct {
	Repository string             `json:"repository" yaml:"repository"`
	DryRun     bool               `json:"dry_run" yaml:"dry_run"`
	Total      int                `json:"total" yaml:"total"`
	Succeeded  int                `json:"succeeded" yaml:"succeeded"`
	Failed     int                `json:"failed" yaml:"failed"`
	Results    []batchCloseResult `json:"results" yaml:"results"`
}

func newBatchCloseShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-close",
		Description: "Close multiple issues by issue numbers or a CSV file",
		Flags: []common.Flag{
			{Name: "numbers", Short: "n", Usage: "Comma-separated issue numbers from the web URL, for example: 1,2,3"},
			{Name: "from", Usage: "Read issue numbers from a CSV file. Supports a number/issue_number/project_issues_index column or first column without header"},
			{Name: "dry-run", Usage: "Preview the issues that would be closed without changing them", Bool: true, Default: "false"},
		},
		Run: runBatchClose,
	}
}

func runBatchClose(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	numbers, err := collectIssueNumbers(ctx.Arg("numbers"), ctx.Arg("from"))
	if err != nil {
		return err
	}
	if len(numbers) == 0 {
		return fmt.Errorf("no issue numbers provided; use --numbers 1,2,3 or --from issues.csv")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchCloseSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Total:      len(numbers),
		Results:    make([]batchCloseResult, 0, len(numbers)),
	}

	for _, number := range numbers {
		result := batchCloseResult{Number: number, Action: "close"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := closeIssue(ctx, number); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "closed"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d issue(s) failed to close", summary.Failed, summary.Total)
	}
	return nil
}

func closeIssue(ctx *common.RuntimeContext, number string) error {
	current, err := fetchExistingIssue(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	body := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
		"status_id":   closedIssueStatusID,
	}
	if _, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body); err != nil {
		return fmt.Errorf("close issue: %w", err)
	}
	return nil
}

func collectIssueNumbers(numbersValue, csvPath string) ([]string, error) {
	numbers, err := parseIssueNumbers(numbersValue)
	if err != nil {
		return nil, err
	}
	if csvPath == "" {
		return numbers, nil
	}

	csvNumbers, err := readIssueNumbersFromCSV(csvPath)
	if err != nil {
		return nil, err
	}
	return mergeIssueNumbers(numbers, csvNumbers), nil
}

func parseIssueNumbers(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return normalizeIssueNumbers(strings.Split(value, ","))
}

func readIssueNumbersFromCSV(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read issue numbers from CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse issue numbers from CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	numberColumn := -1
	startRow := 0
	for i, cell := range records[0] {
		switch strings.ToLower(strings.TrimSpace(cell)) {
		case "number", "issue_number", "project_issues_index":
			numberColumn = i
			startRow = 1
		}
	}
	if numberColumn == -1 {
		numberColumn = 0
	}

	values := make([]string, 0, len(records)-startRow)
	for _, record := range records[startRow:] {
		if numberColumn >= len(record) {
			continue
		}
		values = append(values, record[numberColumn])
	}
	return normalizeIssueNumbers(values)
}

func normalizeIssueNumbers(values []string) ([]string, error) {
	numbers := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		number := strings.TrimSpace(value)
		if number == "" {
			continue
		}
		if _, err := strconv.ParseInt(number, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid issue number %q: issue numbers must be integers", number)
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		numbers = append(numbers, number)
	}
	return numbers, nil
}

func mergeIssueNumbers(values ...[]string) []string {
	merged := []string{}
	seen := map[string]bool{}
	for _, numbers := range values {
		for _, number := range numbers {
			if seen[number] {
				continue
			}
			seen[number] = true
			merged = append(merged, number)
		}
	}
	return merged
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

type batchMaintenanceDryRun struct {
	Repository string                 `json:"repository" yaml:"repository"`
	DryRun     bool                   `json:"dry_run" yaml:"dry_run"`
	Action     string                 `json:"action" yaml:"action"`
	Method     string                 `json:"method" yaml:"method"`
	Path       string                 `json:"path" yaml:"path"`
	Body       map[string]interface{} `json:"body" yaml:"body"`
}

func newBatchUpdateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-update",
		Description: "Batch update issue metadata by API issue IDs",
		Flags: []common.Flag{
			{Name: "ids", Usage: "Comma-separated API issue IDs, not web URL issue numbers", Required: true},
			{Name: "status-id", Usage: "Issue status ID"},
			{Name: "priority-id", Usage: "Issue priority ID"},
			{Name: "milestone-id", Usage: "Issue milestone ID"},
			{Name: "tag-ids", Usage: "Comma-separated issue tag IDs"},
			{Name: "assigner-ids", Usage: "Comma-separated assignee user IDs"},
			{Name: "dry-run", Usage: "Preview request without updating issues", Bool: true, Default: "false"},
		},
		Run: runBatchUpdate,
	}
}

func newBatchDeleteShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-delete",
		Description: "Batch delete issues by API issue IDs",
		Flags: []common.Flag{
			{Name: "ids", Usage: "Comma-separated API issue IDs, not web URL issue numbers", Required: true},
			{Name: "dry-run", Usage: "Preview request without deleting issues", Bool: true, Default: "false"},
			{Name: "yes", Usage: "Confirm real batch deletion", Bool: true, Default: "false"},
		},
		Run: runBatchDelete,
	}
}

func runBatchUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	body, err := buildBatchUpdateBody(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s/issues/batch_update", v1RepoPath(ctx))
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(batchMaintenanceDryRun{
			Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			DryRun:     true,
			Action:     "batch_update_issues",
			Method:     "PATCH",
			Path:       path,
			Body:       body,
		})
	}
	env, err := ctx.CallAPI("PATCH", path, body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runBatchDelete(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	ids, err := parseIntIDList(ctx.Arg("ids"), "ids")
	if err != nil {
		return err
	}
	body := map[string]interface{}{"ids": ids}
	path := fmt.Sprintf("%s/issues/batch_destroy", v1RepoPath(ctx))
	dryRun := parseBool(ctx.Arg("dry-run"))
	if dryRun {
		return ctx.OutputData(batchMaintenanceDryRun{
			Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
			DryRun:     true,
			Action:     "batch_delete_issues",
			Method:     "DELETE",
			Path:       path,
			Body:       body,
		})
	}
	if !parseBool(ctx.Arg("yes")) {
		return fmt.Errorf("batch-delete is destructive; run with --dry-run first, then pass --yes to confirm")
	}
	env, err := ctx.CallAPI("DELETE", path, body)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func buildBatchUpdateBody(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	ids, err := parseIntIDList(ctx.Arg("ids"), "ids")
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{"ids": ids}
	changed := false
	if value := ctx.Arg("status-id"); value != "" {
		id, err := parseSingleIntID(value, "status-id")
		if err != nil {
			return nil, err
		}
		body["status_id"] = id
		changed = true
	}
	if value := ctx.Arg("priority-id"); value != "" {
		id, err := parseSingleIntID(value, "priority-id")
		if err != nil {
			return nil, err
		}
		body["priority_id"] = id
		changed = true
	}
	if value := ctx.Arg("milestone-id"); value != "" {
		id, err := parseSingleIntID(value, "milestone-id")
		if err != nil {
			return nil, err
		}
		body["milestone_id"] = id
		changed = true
	}
	if value := ctx.Arg("tag-ids"); value != "" {
		ids, err := parseIntIDList(value, "tag-ids")
		if err != nil {
			return nil, err
		}
		body["issue_tag_ids"] = ids
		changed = true
	}
	if value := ctx.Arg("assigner-ids"); value != "" {
		ids, err := parseIntIDList(value, "assigner-ids")
		if err != nil {
			return nil, err
		}
		body["assigner_ids"] = ids
		changed = true
	}
	if !changed {
		return nil, fmt.Errorf("no update fields provided; set at least one of --status-id, --priority-id, --milestone-id, --tag-ids, --assigner-ids")
	}
	return body, nil
}

func parseSingleIntID(value, field string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s cannot be empty", field)
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s %q: must be a positive integer", field, value)
	}
	return id, nil
}

func parseIntIDList(value, field string) ([]int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("%s cannot be empty", field)
	}
	parts := strings.Split(value, ",")
	ids := make([]int, 0, len(parts))
	seen := map[int]bool{}
	for _, part := range parts {
		id, err := parseSingleIntID(part, field)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}
