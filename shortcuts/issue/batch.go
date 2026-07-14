package issue

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	closedIssueStatusID = 5
	openIssueStatusID   = 1
)

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

// batch-reopen implementation

type batchReopenResult struct {
	Number string `json:"number" yaml:"number"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchReopenSummary struct {
	Repository string              `json:"repository" yaml:"repository"`
	DryRun     bool                `json:"dry_run" yaml:"dry_run"`
	Total      int                 `json:"total" yaml:"total"`
	Succeeded  int                 `json:"succeeded" yaml:"succeeded"`
	Failed     int                 `json:"failed" yaml:"failed"`
	Results    []batchReopenResult `json:"results" yaml:"results"`
}

func newBatchReopenShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-reopen",
		Description: "Reopen multiple closed issues by issue numbers or a CSV file",
		Flags: []common.Flag{
			{Name: "numbers", Short: "n", Usage: "Comma-separated issue numbers from the web URL, for example: 1,2,3"},
			{Name: "from", Usage: "Read issue numbers from a CSV file. Supports a number/issue_number/project_issues_index column or first column without header"},
			{Name: "dry-run", Usage: "Preview the issues that would be reopened without changing them", Bool: true, Default: "false"},
		},
		Run: runBatchReopen,
	}
}

func runBatchReopen(ctx *common.RuntimeContext) error {
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
	summary := batchReopenSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Total:      len(numbers),
		Results:    make([]batchReopenResult, 0, len(numbers)),
	}

	for _, number := range numbers {
		result := batchReopenResult{Number: number, Action: "reopen"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := reopenIssue(ctx, number); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "reopened"
			summary.Succeeded++
		}
		summary.Results = append(summary.Results, result)
	}

	if err := ctx.OutputData(summary); err != nil {
		return err
	}
	if summary.Failed > 0 {
		return fmt.Errorf("%d of %d issue(s) failed to reopen", summary.Failed, summary.Total)
	}
	return nil
}

func reopenIssue(ctx *common.RuntimeContext, number string) error {
	current, err := fetchExistingIssue(ctx, number)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	body := map[string]interface{}{
		"subject":     current.Subject,
		"description": current.Description,
		"status_id":   openIssueStatusID,
	}
	if _, err := ctx.CallAPI("PATCH", fmt.Sprintf("%s/issues/%s", v1RepoPath(ctx), number), body); err != nil {
		return fmt.Errorf("reopen issue: %w", err)
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

// batch-label implementation

type batchLabelResult struct {
	ID     string `json:"id" yaml:"id"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchLabelSummary struct {
	Repository string             `json:"repository" yaml:"repository"`
	DryRun     bool               `json:"dry_run" yaml:"dry_run"`
	Total      int                `json:"total" yaml:"total"`
	Succeeded  int                `json:"succeeded" yaml:"succeeded"`
	Failed     int                `json:"failed" yaml:"failed"`
	Results    []batchLabelResult `json:"results" yaml:"results"`
}

func newBatchLabelShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-label",
		Description: "Batch add or remove labels from issues by API issue IDs",
		Flags: []common.Flag{
			{Name: "ids", Usage: "Comma-separated API issue IDs, not web URL issue numbers", Required: true},
			{Name: "add", Usage: "Comma-separated tag IDs to add to issues"},
			{Name: "remove", Usage: "Comma-separated tag IDs to remove from issues"},
			{Name: "dry-run", Usage: "Preview changes without updating issues", Bool: true, Default: "false"},
		},
		Run: runBatchLabel,
	}
}

func runBatchLabel(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	ids, err := parseIntIDList(ctx.Arg("ids"), "ids")
	if err != nil {
		return err
	}

	addTags, err := parseOptionalIntIDList(ctx.Arg("add"))
	if err != nil {
		return fmt.Errorf("parse add tags: %w", err)
	}

	removeTags, err := parseOptionalIntIDList(ctx.Arg("remove"))
	if err != nil {
		return fmt.Errorf("parse remove tags: %w", err)
	}

	if len(addTags) == 0 && len(removeTags) == 0 {
		return fmt.Errorf("no label changes provided; use --add and/or --remove with tag IDs")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchLabelSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Total:      len(ids),
		Results:    make([]batchLabelResult, 0, len(ids)),
	}

	for _, id := range ids {
		result := batchLabelResult{ID: strconv.Itoa(id), Action: "update_labels"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := updateIssueLabels(ctx, id, addTags, removeTags); err != nil {
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
		return fmt.Errorf("%d of %d issue(s) failed to update labels", summary.Failed, summary.Total)
	}
	return nil
}

func parseOptionalIntIDList(value string) ([]int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return parseIntIDList(value, "tag-ids")
}

func updateIssueLabels(ctx *common.RuntimeContext, issueID int, addTags, removeTags []int) error {
	// Fetch current issue to get existing tags
	issueData, err := fetchIssueDataByID(ctx, issueID)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	// Get current tag IDs
	currentTagIDs := issueObjectIDs(issueData, "tags", "issue_tags")
	currentTags := make(map[int]bool)
	for _, id := range currentTagIDs {
		if tagID, ok := id.(float64); ok {
			currentTags[int(tagID)] = true
		} else if tagID, ok := id.(int); ok {
			currentTags[tagID] = true
		}
	}

	// Add new tags
	for _, tagID := range addTags {
		currentTags[tagID] = true
	}

	// Remove tags
	for _, tagID := range removeTags {
		delete(currentTags, tagID)
	}

	// Convert back to slice and sort for consistent ordering
	newTagIDs := make([]int, 0, len(currentTags))
	for tagID := range currentTags {
		newTagIDs = append(newTagIDs, tagID)
	}
	sort.Ints(newTagIDs)

	// Convert to []interface{} for JSON
	newTags := make([]interface{}, len(newTagIDs))
	for i, id := range newTagIDs {
		newTags[i] = id
	}

	// Update issue
	body := map[string]interface{}{
		"ids":           []int{issueID},
		"issue_tag_ids": newTags,
	}
	path := fmt.Sprintf("%s/issues/batch_update", v1RepoPath(ctx))
	if _, err := ctx.CallAPI("PATCH", path, body); err != nil {
		return fmt.Errorf("update issue labels: %w", err)
	}
	return nil
}

func fetchIssueDataByID(ctx *common.RuntimeContext, id int) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/issues/%d", v1RepoPath(ctx), id)
	env, err := ctx.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}
	issueData, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse issue data")
	}
	return issueData, nil
}

// batch-assign implementation

type batchAssignResult struct {
	ID     string `json:"id" yaml:"id"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchAssignSummary struct {
	Repository string              `json:"repository" yaml:"repository"`
	DryRun     bool                `json:"dry_run" yaml:"dry_run"`
	Total      int                 `json:"total" yaml:"total"`
	Succeeded  int                 `json:"succeeded" yaml:"succeeded"`
	Failed     int                 `json:"failed" yaml:"failed"`
	Results    []batchAssignResult `json:"results" yaml:"results"`
}

func newBatchAssignShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-assign",
		Description: "Batch assign or unassign users from issues by API issue IDs",
		Flags: []common.Flag{
			{Name: "ids", Usage: "Comma-separated API issue IDs, not web URL issue numbers", Required: true},
			{Name: "add", Usage: "Comma-separated user IDs to assign to issues"},
			{Name: "remove", Usage: "Comma-separated user IDs to unassign from issues"},
			{Name: "dry-run", Usage: "Preview changes without updating issues", Bool: true, Default: "false"},
		},
		Run: runBatchAssign,
	}
}

func runBatchAssign(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	ids, err := parseIntIDList(ctx.Arg("ids"), "ids")
	if err != nil {
		return err
	}

	addUsers, err := parseOptionalIntIDList(ctx.Arg("add"))
	if err != nil {
		return fmt.Errorf("parse add users: %w", err)
	}

	removeUsers, err := parseOptionalIntIDList(ctx.Arg("remove"))
	if err != nil {
		return fmt.Errorf("parse remove users: %w", err)
	}

	if len(addUsers) == 0 && len(removeUsers) == 0 {
		return fmt.Errorf("no assignee changes provided; use --add and/or --remove with user IDs")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchAssignSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Total:      len(ids),
		Results:    make([]batchAssignResult, 0, len(ids)),
	}

	for _, id := range ids {
		result := batchAssignResult{ID: strconv.Itoa(id), Action: "update_assignees"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := updateIssueAssignees(ctx, id, addUsers, removeUsers); err != nil {
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
		return fmt.Errorf("%d of %d issue(s) failed to update assignees", summary.Failed, summary.Total)
	}
	return nil
}

func updateIssueAssignees(ctx *common.RuntimeContext, issueID int, addUsers, removeUsers []int) error {
	// Fetch current issue to get existing assignees
	issueData, err := fetchIssueDataByID(ctx, issueID)
	if err != nil {
		return fmt.Errorf("fetch issue: %w", err)
	}

	// Get current assignee IDs
	currentAssigneeIDs := issueObjectIDs(issueData, "assigners")
	currentAssignees := make(map[int]bool)
	for _, id := range currentAssigneeIDs {
		if userID, ok := id.(float64); ok {
			currentAssignees[int(userID)] = true
		} else if userID, ok := id.(int); ok {
			currentAssignees[userID] = true
		}
	}

	// Add new assignees
	for _, userID := range addUsers {
		currentAssignees[userID] = true
	}

	// Remove assignees
	for _, userID := range removeUsers {
		delete(currentAssignees, userID)
	}

	// Convert back to slice and sort for consistent ordering
	newUserIDs := make([]int, 0, len(currentAssignees))
	for userID := range currentAssignees {
		newUserIDs = append(newUserIDs, userID)
	}
	sort.Ints(newUserIDs)

	// Convert to []interface{} for JSON
	newAssignees := make([]interface{}, len(newUserIDs))
	for i, id := range newUserIDs {
		newAssignees[i] = id
	}

	// Update issue
	body := map[string]interface{}{
		"ids":          []int{issueID},
		"assigner_ids": newAssignees,
	}
	path := fmt.Sprintf("%s/issues/batch_update", v1RepoPath(ctx))
	if _, err := ctx.CallAPI("PATCH", path, body); err != nil {
		return fmt.Errorf("update issue assignees: %w", err)
	}
	return nil
}

// batch-comment implementation

type batchCommentResult struct {
	Number string `json:"number" yaml:"number"`
	Action string `json:"action" yaml:"action"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchCommentSummary struct {
	Repository string               `json:"repository" yaml:"repository"`
	DryRun     bool                 `json:"dry_run" yaml:"dry_run"`
	Message    string               `json:"message" yaml:"message"`
	Total      int                  `json:"total" yaml:"total"`
	Succeeded  int                  `json:"succeeded" yaml:"succeeded"`
	Failed     int                  `json:"failed" yaml:"failed"`
	Results    []batchCommentResult `json:"results" yaml:"results"`
}

func newBatchCommentShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-comment",
		Description: "Batch add comments to multiple issues by issue numbers or a CSV file",
		Flags: []common.Flag{
			{Name: "numbers", Short: "n", Usage: "Comma-separated issue numbers from the web URL, for example: 1,2,3"},
			{Name: "from", Usage: "Read issue numbers from a CSV file. Supports a number/issue_number/project_issues_index column or first column without header"},
			{Name: "message", Short: "m", Usage: "Comment message to add to all issues", Required: true},
			{Name: "dry-run", Usage: "Preview the issues that would receive comments without posting them", Bool: true, Default: "false"},
		},
		Run: runBatchComment,
	}
}

func runBatchComment(ctx *common.RuntimeContext) error {
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

	message, err := ctx.RequireArg("message")
	if err != nil {
		return err
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchCommentSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Message:    message,
		Total:      len(numbers),
		Results:    make([]batchCommentResult, 0, len(numbers)),
	}

	for _, number := range numbers {
		result := batchCommentResult{Number: number, Action: "add_comment"}
		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := addIssueComment(ctx, number, message); err != nil {
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
		return fmt.Errorf("%d of %d issue(s) failed to add comment", summary.Failed, summary.Total)
	}
	return nil
}

func addIssueComment(ctx *common.RuntimeContext, number, message string) error {
	payload := map[string]interface{}{
		"notes": message,
	}
	path := fmt.Sprintf("%s/issues/%s/journals", v1RepoPath(ctx), number)
	if _, err := ctx.CallAPI("POST", path, payload); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

// ============================================================================
// Batch Export
// ============================================================================

type batchExportSummary struct {
	Repository string `json:"repository" yaml:"repository"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output" yaml:"output"`
	Total      int    `json:"total" yaml:"total"`
}

func newBatchExportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-export",
		Description: "Export issues to CSV or JSON file",
		Flags: []common.Flag{
			{Name: "format", Short: "f", Usage: "Export format: csv or json", Default: "csv"},
			{Name: "output", Short: "o", Usage: "Output file path (default: issues.csv or issues.json)"},
			{Name: "state", Short: "s", Usage: "Filter by state: open, closed, or all", Default: "open"},
			{Name: "keyword", Short: "k", Usage: "Filter by keyword"},
			{Name: "author-id", Usage: "Filter by author ID"},
			{Name: "assignee-id", Usage: "Filter by assignee ID"},
			{Name: "milestone-id", Usage: "Filter by milestone ID"},
			{Name: "status-id", Usage: "Filter by status ID"},
			{Name: "tag-ids", Usage: "Filter by comma-separated tag IDs"},
			{Name: "limit", Short: "l", Usage: "Maximum number of issues to export (0 for all)", Default: "0"},
		},
		Run: runBatchExport,
	}
}

func runBatchExport(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	// Build query parameters
	q := url.Values{}
	if s := ctx.Arg("state"); s != "" {
		q.Set("category", normalizeIssueListState(s))
	}
	if keyword := ctx.Arg("keyword"); keyword != "" {
		q.Set("keyword", keyword)
	}
	if authorID := ctx.Arg("author-id"); authorID != "" {
		q.Set("author_id", authorID)
	}
	if assigneeID := ctx.Arg("assignee-id"); assigneeID != "" {
		q.Set("assigner_id", assigneeID)
	}
	if milestoneID := ctx.Arg("milestone-id"); milestoneID != "" {
		q.Set("milestone_id", milestoneID)
	}
	if statusID := ctx.Arg("status-id"); statusID != "" {
		q.Set("status_id", statusID)
	}
	if tagIDs := ctx.Arg("tag-ids"); tagIDs != "" {
		q.Set("issue_tag_ids", tagIDs)
	}

	// Fetch all issues with pagination
	limitStr := ctx.Arg("limit")
	maxLimit := 0
	if limitStr != "" && limitStr != "0" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			maxLimit = l
		}
	}

	allIssues := make([]map[string]interface{}, 0)
	page := 1
	pageSize := 100

	for {
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(pageSize))

		env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issues", q)
		if err != nil {
			return fmt.Errorf("fetch issues: %w", err)
		}

		data, ok := env.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}

		issues, ok := data["issues"].([]interface{})
		if !ok {
			break
		}

		for _, item := range issues {
			issue, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// Normalize issue data
			if num, ok := issue["project_issues_index"]; ok {
				issue["number"] = num
			}
			if id, ok := issue["id"]; ok {
				issue["database_id"] = id
				delete(issue, "id")
			}
			allIssues = append(allIssues, issue)
		}

		if len(issues) < pageSize {
			break
		}
		if maxLimit > 0 && len(allIssues) >= maxLimit {
			allIssues = allIssues[:maxLimit]
			break
		}
		page++
	}

	// Determine output format and file
	format := strings.ToLower(ctx.Arg("format"))
	if format != "csv" && format != "json" {
		format = "csv"
	}

	outputPath := ctx.Arg("output")
	if outputPath == "" {
		if format == "csv" {
			outputPath = "issues.csv"
		} else {
			outputPath = "issues.json"
		}
	}

	// Export to file
	var err error
	if format == "csv" {
		err = exportIssuesToCSV(allIssues, outputPath)
	} else {
		err = exportIssuesToJSON(allIssues, outputPath)
	}
	if err != nil {
		return fmt.Errorf("export issues: %w", err)
	}

	summary := batchExportSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		Format:     format,
		Output:     outputPath,
		Total:      len(allIssues),
	}

	return ctx.OutputData(summary)
}

func exportIssuesToCSV(issues []map[string]interface{}, path string) error {
	if len(issues) == 0 {
		return fmt.Errorf("no issues to export")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Define CSV columns
	headers := []string{
		"number", "subject", "description", "status", "priority",
		"author", "assigners", "tags", "milestone", "branch_name",
		"start_date", "due_date", "created_at", "updated_at",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write headers: %w", err)
	}

	for _, issue := range issues {
		record := make([]string, len(headers))
		record[0] = getStringField(issue, "number")
		record[1] = getStringField(issue, "subject")
		record[2] = getStringField(issue, "description")
		record[3] = getNestedStringField(issue, "status", "name")
		record[4] = getNestedStringField(issue, "priority", "name")
		record[5] = getNestedStringField(issue, "author", "login")
		record[6] = getNestedArrayField(issue, "assigners", "login")
		record[7] = getNestedArrayField(issue, "tags", "name")
		record[8] = getNestedStringField(issue, "milestone", "name")
		record[9] = getStringField(issue, "branch_name")
		record[10] = getStringField(issue, "start_date")
		record[11] = getStringField(issue, "due_date")
		record[12] = getStringField(issue, "created_at")
		record[13] = getStringField(issue, "updated_at")

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	return nil
}

func exportIssuesToJSON(issues []map[string]interface{}, path string) error {
	data, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return strconv.FormatFloat(val, 'f', -1, 64)
		case int:
			return strconv.Itoa(val)
		}
	}
	return ""
}

func getNestedStringField(m map[string]interface{}, keys ...string) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			return getStringField(current, key)
		}
		if v, ok := current[key]; ok {
			if nested, ok := v.(map[string]interface{}); ok {
				current = nested
			} else {
				break
			}
		} else {
			break
		}
	}
	return ""
}

func getNestedArrayField(m map[string]interface{}, arrayKey, fieldKey string) string {
	if v, ok := m[arrayKey]; ok {
		if arr, ok := v.([]interface{}); ok {
			values := make([]string, 0, len(arr))
			for _, item := range arr {
				if obj, ok := item.(map[string]interface{}); ok {
					if field, ok := obj[fieldKey]; ok {
						if s, ok := field.(string); ok {
							values = append(values, s)
						}
					}
				}
			}
			return strings.Join(values, ",")
		}
	}
	return ""
}

// ============================================================================
// Batch Import
// ============================================================================

type batchImportResult struct {
	Row    int    `json:"row" yaml:"row"`
	Title  string `json:"title" yaml:"title"`
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

type batchImportSummary struct {
	Repository string              `json:"repository" yaml:"repository"`
	DryRun     bool                `json:"dry_run" yaml:"dry_run"`
	Input      string              `json:"input" yaml:"input"`
	Total      int                 `json:"total" yaml:"total"`
	Succeeded  int                 `json:"succeeded" yaml:"succeeded"`
	Failed     int                 `json:"failed" yaml:"failed"`
	Results    []batchImportResult `json:"results" yaml:"results"`
}

func newBatchImportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-import",
		Description: "Create issues from a CSV file",
		Flags: []common.Flag{
			{Name: "from", Short: "f", Usage: "CSV file path to import issues from", Required: true},
			{Name: "dry-run", Usage: "Preview the issues that would be created without creating them", Bool: true, Default: "false"},
		},
		Run: runBatchImport,
	}
}

func runBatchImport(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}

	csvPath := ctx.Arg("from")
	if csvPath == "" {
		return fmt.Errorf("--from is required")
	}

	issues, err := readIssuesFromCSV(csvPath)
	if err != nil {
		return fmt.Errorf("read CSV: %w", err)
	}
	if len(issues) == 0 {
		return fmt.Errorf("no issues found in CSV file")
	}

	dryRun := parseBool(ctx.Arg("dry-run"))
	summary := batchImportSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		DryRun:     dryRun,
		Input:      csvPath,
		Total:      len(issues),
		Results:    make([]batchImportResult, 0, len(issues)),
	}

	for i, issue := range issues {
		result := batchImportResult{
			Row:   i + 2, // +2 because row 1 is header
			Title: issue.Title,
		}

		if dryRun {
			result.Status = "planned"
			summary.Succeeded++
			summary.Results = append(summary.Results, result)
			continue
		}

		if err := createIssueFromImport(ctx, issue); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "created"
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
	return nil
}

type importIssue struct {
	Title       string
	Description string
	PriorityID  int
	TagIDs      []int
	AssignerIDs []int
	MilestoneID int
	BranchName  string
	StartDate   string
	DueDate     string
}

func readIssuesFromCSV(path string) ([]importIssue, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	// Parse header to find column indices
	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	issues := make([]importIssue, 0, len(records)-1)
	for i, record := range records[1:] {
		if len(record) == 0 {
			continue
		}

		issue := importIssue{
			PriorityID: 2, // default: normal
		}

		// Title (required)
		if idx, ok := colIndex["title"]; ok && idx < len(record) {
			issue.Title = strings.TrimSpace(record[idx])
		} else if idx, ok := colIndex["subject"]; ok && idx < len(record) {
			issue.Title = strings.TrimSpace(record[idx])
		}
		if issue.Title == "" {
			return nil, fmt.Errorf("row %d: title is required", i+2)
		}

		// Description
		if idx, ok := colIndex["description"]; ok && idx < len(record) {
			issue.Description = strings.TrimSpace(record[idx])
		} else if idx, ok := colIndex["body"]; ok && idx < len(record) {
			issue.Description = strings.TrimSpace(record[idx])
		}

		// Priority ID
		if idx, ok := colIndex["priority_id"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				if id, err := strconv.Atoi(val); err == nil {
					issue.PriorityID = id
				}
			}
		}

		// Tag IDs
		if idx, ok := colIndex["tag_ids"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				ids, err := parseIntList(val)
				if err != nil {
					return nil, fmt.Errorf("row %d: invalid tag_ids: %w", i+2, err)
				}
				issue.TagIDs = ids
			}
		}

		// Assigner IDs
		if idx, ok := colIndex["assigner_ids"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				ids, err := parseIntList(val)
				if err != nil {
					return nil, fmt.Errorf("row %d: invalid assigner_ids: %w", i+2, err)
				}
				issue.AssignerIDs = ids
			}
		}

		// Milestone ID
		if idx, ok := colIndex["milestone_id"]; ok && idx < len(record) {
			if val := strings.TrimSpace(record[idx]); val != "" {
				if id, err := strconv.Atoi(val); err == nil {
					issue.MilestoneID = id
				}
			}
		}

		// Branch name
		if idx, ok := colIndex["branch_name"]; ok && idx < len(record) {
			issue.BranchName = strings.TrimSpace(record[idx])
		}

		// Start date
		if idx, ok := colIndex["start_date"]; ok && idx < len(record) {
			issue.StartDate = strings.TrimSpace(record[idx])
		}

		// Due date
		if idx, ok := colIndex["due_date"]; ok && idx < len(record) {
			issue.DueDate = strings.TrimSpace(record[idx])
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

func parseIntList(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func createIssueFromImport(ctx *common.RuntimeContext, issue importIssue) error {
	body := map[string]interface{}{
		"subject":     issue.Title,
		"status_id":   1, // open
		"priority_id": issue.PriorityID,
		"done_ratio":  0,
	}

	if issue.Description != "" {
		body["description"] = issue.Description
	}
	if len(issue.TagIDs) > 0 {
		body["issue_tag_ids"] = issue.TagIDs
	}
	if len(issue.AssignerIDs) > 0 {
		body["assigner_ids"] = issue.AssignerIDs
	}
	if issue.MilestoneID > 0 {
		body["fixed_version_id"] = issue.MilestoneID
	}
	if issue.BranchName != "" {
		body["branch_name"] = issue.BranchName
	}
	if issue.StartDate != "" {
		body["start_date"] = issue.StartDate
	}
	if issue.DueDate != "" {
		body["due_date"] = issue.DueDate
	}

	if _, err := ctx.CallAPI("POST", v1RepoPath(ctx)+"/issues", body); err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	return nil
}
