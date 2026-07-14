package issue

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const closedIssueStatusID = 5

type batchIssueResult struct {
	Number string `json:"number" yaml:"number"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchIssueSummary struct {
	Repository string                 `json:"repository" yaml:"repository"`
	Action     string                 `json:"action" yaml:"action"`
	DryRun     bool                   `json:"dry_run" yaml:"dry_run"`
	Preview    map[string]interface{} `json:"preview,omitempty" yaml:"preview,omitempty"`
	Total      int                    `json:"total" yaml:"total"`
	Succeeded  int                    `json:"succeeded" yaml:"succeeded"`
	Failed     int                    `json:"failed" yaml:"failed"`
	Results    []batchIssueResult     `json:"results" yaml:"results"`
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

func newBatchCommentShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-comment",
		Description: "Add the same comment to multiple issues by issue numbers or a CSV file",
		Flags: append(batchIssueTargetFlags(),
			common.Flag{Name: "body", Short: "b", Usage: "Comment body"},
			common.Flag{Name: "body-file", Usage: "Read comment body from a file"},
		),
		Run: runBatchComment,
	}
}

func newBatchUpdateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-update",
		Description: "Apply the same metadata updates to multiple issues by issue numbers or a CSV file",
		Flags: append(batchIssueTargetFlags(),
			common.Flag{Name: "title", Short: "t", Usage: "New issue title"},
			common.Flag{Name: "body", Short: "b", Usage: "New issue description"},
			common.Flag{Name: "body-file", Usage: "Read the new issue description from a file"},
			common.Flag{Name: "state", Short: "s", Usage: "New issue state"},
			common.Flag{Name: "priority-id", Usage: "New priority ID"},
			common.Flag{Name: "tag-ids", Usage: "Comma-separated issue tag IDs"},
			common.Flag{Name: "assigner-ids", Usage: "Comma-separated issue assigner IDs"},
			common.Flag{Name: "branch", Usage: "Linked branch name"},
			common.Flag{Name: "start-date", Usage: "Start date (YYYY-MM-DD)"},
			common.Flag{Name: "due-date", Usage: "Due date (YYYY-MM-DD)"},
		),
		Run: runBatchUpdate,
	}
}

func batchIssueTargetFlags() []common.Flag {
	return []common.Flag{
		{Name: "numbers", Short: "n", Usage: "Comma-separated issue numbers from the web URL, for example: 1,2,3"},
		{Name: "from", Usage: "Read issue numbers from a CSV file. Supports a number/issue_number/project_issues_index column or first column without header"},
		{Name: "dry-run", Usage: "Preview the issues that would be changed without changing them", Bool: true, Default: "false"},
	}
}

func runBatchClose(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	numbers, err := collectBatchIssueTargets(ctx)
	if err != nil {
		return err
	}
	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := newBatchIssueSummary(ctx, "close", dryRun, len(numbers), nil)

	for _, number := range numbers {
		result := batchIssueResult{Number: number, Action: "close"}
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

func runBatchComment(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	numbers, err := collectBatchIssueTargets(ctx)
	if err != nil {
		return err
	}
	body, err := readIssueTextArg(ctx, "body", "body-file", true)
	if err != nil {
		return err
	}
	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := newBatchIssueSummary(ctx, "comment", dryRun, len(numbers), map[string]interface{}{
		"body_preview": previewText(body),
		"body_length":  utf8.RuneCountInString(body),
	})
	if file := strings.TrimSpace(ctx.Arg("body-file")); file != "" {
		summary.Preview["body_file"] = file
	}

	for _, number := range numbers {
		result := batchIssueResult{Number: number, Action: "comment"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if _, err := commentOnIssue(ctx, number, body); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "commented"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d issue(s) failed to comment", summary.Failed, summary.Total)
	}
	return nil
}

func runBatchUpdate(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	numbers, err := collectBatchIssueTargets(ctx)
	if err != nil {
		return err
	}
	preview, err := buildIssueUpdatePreview(ctx)
	if err != nil {
		return err
	}
	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := newBatchIssueSummary(ctx, "update", dryRun, len(numbers), preview)

	for _, number := range numbers {
		result := batchIssueResult{Number: number, Action: "update"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := updateIssue(ctx, number); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "updated"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d issue(s) failed to update", summary.Failed, summary.Total)
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

func updateIssue(ctx *common.RuntimeContext, number string) error {
	current, err := fetchExistingIssue(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}
	body, err := buildIssueUpdateBody(ctx, current)
	if err != nil {
		return err
	}
	if _, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body); err != nil {
		return fmt.Errorf("update issue: %w", err)
	}
	return nil
}

func commentOnIssue(ctx *common.RuntimeContext, number, body string) (*output.Envelope, error) {
	payload := map[string]interface{}{
		"notes": body,
	}
	return ctx.CallAPI("POST", fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), number), payload)
}

func newBatchIssueSummary(ctx *common.RuntimeContext, action string, dryRun bool, total int, preview map[string]interface{}) batchIssueSummary {
	return batchIssueSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		Action:     action,
		DryRun:     dryRun,
		Preview:    preview,
		Total:      total,
		Results:    make([]batchIssueResult, 0, total),
	}
}

func collectBatchIssueTargets(ctx *common.RuntimeContext) ([]string, error) {
	numbers, err := collectIssueNumbers(ctx.Arg("numbers"), ctx.Arg("from"))
	if err != nil {
		return nil, err
	}
	if len(numbers) == 0 {
		return nil, fmt.Errorf("no issue numbers provided; use --numbers 1,2,3 or --from issues.csv")
	}
	return numbers, nil
}

func buildIssueUpdatePreview(ctx *common.RuntimeContext) (map[string]interface{}, error) {
	description, err := readIssueTextArg(ctx, "body", "body-file", false)
	if err != nil {
		return nil, err
	}
	preview := map[string]interface{}{}
	if title := ctx.Arg("title"); title != "" {
		preview["subject"] = title
	}
	if description != "" {
		preview["description_preview"] = previewText(description)
		preview["description_length"] = utf8.RuneCountInString(description)
	}
	if state := ctx.Arg("state"); state != "" {
		statusID, err := normalizeIssueStatus(state)
		if err != nil {
			return nil, err
		}
		preview["status_id"] = statusID
	}
	if err := applyIssueMetadataArgs(ctx, preview); err != nil {
		return nil, err
	}
	if file := strings.TrimSpace(ctx.Arg("body-file")); file != "" {
		preview["body_file"] = file
	}
	if len(preview) == 0 {
		return nil, fmt.Errorf("at least one update field is required")
	}
	return preview, nil
}

func readIssueTextArg(ctx *common.RuntimeContext, inlineArg, fileArg string, required bool) (string, error) {
	inline := ctx.Arg(inlineArg)
	file := strings.TrimSpace(ctx.Arg(fileArg))
	if inline != "" && file != "" {
		return "", fmt.Errorf("--%s cannot be used with --%s", inlineArg, fileArg)
	}
	if inline != "" {
		return inline, nil
	}
	if file != "" {
		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fileArg, err)
		}
		return string(content), nil
	}
	if required {
		return "", fmt.Errorf("required flag --%s is missing", inlineArg)
	}
	return "", nil
}

func previewText(text string) string {
	const maxRunes = 120
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "..."
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
