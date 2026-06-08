package issue

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const defaultIssueExportFields = "number,title,state,status,priority,author,assignees,tags,updated_at,url"

type issueExportRecord map[string]string

type issueExportSummary struct {
	Repository string `json:"repository" yaml:"repository"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output,omitempty" yaml:"output,omitempty"`
	Total      int    `json:"total" yaml:"total"`
	Pages      int    `json:"pages" yaml:"pages"`
	Fields     string `json:"fields" yaml:"fields"`
}

func newExportShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "export",
		Description: tr.T("cmd.issue.export.short"),
		Long:        tr.T("cmd.issue.export.long"),
		Flags: []common.Flag{
			{Name: "state", Short: "s", Usage: tr.T("flag.issue.state"), Default: "open"},
			{Name: "keyword", Short: "k", Usage: tr.T("flag.search.keyword")},
			{Name: "participant", Usage: tr.T("flag.issue.participant")},
			{Name: "author-id", Usage: tr.T("flag.issue.author_id")},
			{Name: "assignee-id", Usage: tr.T("flag.issue.assignee_id")},
			{Name: "milestone-id", Usage: tr.T("flag.issue.milestone")},
			{Name: "status-id", Usage: tr.T("flag.issue.status_id")},
			{Name: "tag-ids", Usage: tr.T("flag.issue.tag_ids")},
			{Name: "sort-by", Usage: tr.T("flag.sort_by")},
			{Name: "sort-direction", Usage: tr.T("flag.sort_direction")},
			{Name: "limit", Short: "l", Usage: tr.T("flag.issue.export_limit"), Default: "50"},
			{Name: "max", Usage: tr.T("flag.issue.export_max"), Default: "0"},
			{Name: "fields", Usage: tr.T("flag.issue.export_fields"), Default: defaultIssueExportFields},
			{Name: "export-format", Usage: tr.T("flag.issue.export_format"), Default: "csv"},
			{Name: "output", Short: "o", Usage: tr.T("flag.issue.export_output")},
		},
		Run: runIssueExport,
	}
}

func runIssueExport(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	limit, err := boundedPositiveInt(ctx.Arg("limit"), 50, 100, "limit")
	if err != nil {
		return err
	}
	maxItems, err := nonNegativeInt(ctx.Arg("max"), "max")
	if err != nil {
		return err
	}
	fields, err := parseIssueExportFields(ctx.Arg("fields"))
	if err != nil {
		return err
	}
	format, err := normalizeIssueExportFormat(ctx.Arg("export-format"))
	if err != nil {
		return err
	}

	issues, pages, err := fetchIssuesForExport(ctx, buildIssueExportQuery(ctx, limit), limit, maxItems)
	if err != nil {
		return err
	}
	records := make([]issueExportRecord, 0, len(issues))
	for _, issue := range issues {
		records = append(records, normalizeIssueExportRecord(ctx, issue))
	}

	content, err := renderIssueExport(records, fields, format)
	if err != nil {
		return err
	}
	outputPath := strings.TrimSpace(ctx.Arg("output"))
	if outputPath == "" {
		_, err = os.Stdout.Write(content)
		return err
	}
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	return ctx.OutputData(issueExportSummary{
		Repository: fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		Format:     format,
		Output:     outputPath,
		Total:      len(records),
		Pages:      pages,
		Fields:     strings.Join(fields, ","),
	})
}

func buildIssueExportQuery(ctx *common.RuntimeContext, limit int) url.Values {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if s := ctx.Arg("state"); s != "" {
		q.Set("category", normalizeIssueListState(s))
	}
	if keyword := ctx.Arg("keyword"); keyword != "" {
		q.Set("keyword", keyword)
	}
	if participant := ctx.Arg("participant"); participant != "" {
		q.Set("participant_category", participant)
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
	if sortBy := ctx.Arg("sort-by"); sortBy != "" {
		q.Set("sort_by", sortBy)
	}
	if sortDirection := ctx.Arg("sort-direction"); sortDirection != "" {
		q.Set("sort_direction", sortDirection)
	}
	return q
}

func fetchIssuesForExport(ctx *common.RuntimeContext, query url.Values, limit, maxItems int) ([]map[string]interface{}, int, error) {
	var result []map[string]interface{}
	page := 1
	pagesFetched := 0
	for {
		query.Set("page", strconv.Itoa(page))
		env, err := ctx.CallAPIWithQuery("GET", v1RepoPath(ctx)+"/issues", query)
		if err != nil {
			return nil, pagesFetched, err
		}
		items := extractIssueList(env)
		if len(items) == 0 {
			break
		}
		pagesFetched++
		for _, item := range items {
			if maxItems > 0 && len(result) >= maxItems {
				return result, pagesFetched, nil
			}
			result = append(result, item)
		}
		if len(items) < limit {
			break
		}
		if total := totalCount(env); total > 0 && len(result) >= total {
			break
		}
		page++
	}
	return result, pagesFetched, nil
}

func extractIssueList(env *output.Envelope) []map[string]interface{} {
	if env == nil || env.Data == nil {
		return nil
	}
	var raw []interface{}
	switch data := env.Data.(type) {
	case []interface{}:
		raw = data
	case map[string]interface{}:
		for _, key := range []string{"issues", "data"} {
			if values, ok := data[key].([]interface{}); ok {
				raw = values
				break
			}
		}
	}
	items := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if issue, ok := item.(map[string]interface{}); ok {
			items = append(items, issue)
		}
	}
	return items
}

func totalCount(env *output.Envelope) int {
	if env == nil {
		return 0
	}
	if env.Meta != nil && env.Meta.TotalCount > 0 {
		return env.Meta.TotalCount
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return 0
	}
	return intFromAny(data["total_count"])
}

func normalizeIssueExportRecord(ctx *common.RuntimeContext, issue map[string]interface{}) issueExportRecord {
	number := firstStringValue(issue, "project_issues_index", "number", "index")
	databaseID := firstStringValue(issue, "id", "database_id")
	record := issueExportRecord{
		"number":      number,
		"database_id": databaseID,
		"title":       firstStringValue(issue, "subject", "title", "name"),
		"description": firstStringValue(issue, "description", "body"),
		"state":       issueState(issue),
		"status":      nestedName(issue, "status"),
		"status_id":   nestedID(issue, "status"),
		"priority":    nestedName(issue, "priority"),
		"priority_id": nestedID(issue, "priority"),
		"author":      nestedLoginOrName(issue, "author"),
		"assignees":   joinedNames(issue, "assigners", "assignees", "assigned_to"),
		"tags":        joinedNames(issue, "tags", "issue_tags"),
		"milestone":   nestedName(issue, "version", "fixed_version", "milestone"),
		"branch":      firstStringValue(issue, "branch_name"),
		"created_at":  firstStringValue(issue, "created_at", "created_on"),
		"updated_at":  firstStringValue(issue, "updated_at", "updated_on"),
		"closed_at":   firstStringValue(issue, "closed_at", "closed_on"),
	}
	if number != "" {
		record["url"] = fmt.Sprintf("https://www.gitlink.org.cn/%s/%s/issues/%s", ctx.Owner, ctx.Repo, url.PathEscape(number))
	}
	return record
}

func renderIssueExport(records []issueExportRecord, fields []string, format string) ([]byte, error) {
	switch format {
	case "csv":
		return renderIssueExportCSV(records, fields)
	case "json":
		return json.MarshalIndent(selectIssueExportFields(records, fields), "", "  ")
	case "markdown":
		return renderIssueExportMarkdown(records, fields), nil
	default:
		return nil, fmt.Errorf("unsupported export format %q", format)
	}
}

func selectIssueExportFields(records []issueExportRecord, fields []string) []issueExportRecord {
	selected := make([]issueExportRecord, 0, len(records))
	for _, record := range records {
		item := issueExportRecord{}
		for _, field := range fields {
			item[field] = record[field]
		}
		selected = append(selected, item)
	}
	return selected
}

func renderIssueExportCSV(records []issueExportRecord, fields []string) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	if err := writer.Write(fields); err != nil {
		return nil, err
	}
	for _, record := range records {
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = record[field]
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func renderIssueExportMarkdown(records []issueExportRecord, fields []string) []byte {
	var buf strings.Builder
	buf.WriteString("| ")
	buf.WriteString(strings.Join(fields, " | "))
	buf.WriteString(" |\n| ")
	separators := make([]string, len(fields))
	for i := range separators {
		separators[i] = "---"
	}
	buf.WriteString(strings.Join(separators, " | "))
	buf.WriteString(" |\n")
	for _, record := range records {
		values := make([]string, len(fields))
		for i, field := range fields {
			values[i] = escapeMarkdownCell(record[field])
		}
		buf.WriteString("| ")
		buf.WriteString(strings.Join(values, " | "))
		buf.WriteString(" |\n")
	}
	return []byte(buf.String())
}

func parseIssueExportFields(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		value = defaultIssueExportFields
	}
	allowed := issueExportAllowedFields()
	fields := []string{}
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		field := strings.ToLower(strings.TrimSpace(part))
		if field == "" {
			continue
		}
		if !allowed[field] {
			return nil, fmt.Errorf("unsupported export field %q", field)
		}
		if seen[field] {
			continue
		}
		seen[field] = true
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one export field is required")
	}
	return fields, nil
}

func issueExportAllowedFields() map[string]bool {
	fields := []string{
		"number", "database_id", "title", "description", "state", "status", "status_id",
		"priority", "priority_id", "author", "assignees", "tags", "milestone", "branch",
		"created_at", "updated_at", "closed_at", "url",
	}
	allowed := map[string]bool{}
	for _, field := range fields {
		allowed[field] = true
	}
	return allowed
}

func normalizeIssueExportFormat(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "csv":
		return "csv", nil
	case "json":
		return "json", nil
	case "md", "markdown":
		return "markdown", nil
	default:
		return "", fmt.Errorf("unsupported export format %q: use csv, json, or markdown", value)
	}
}

func boundedPositiveInt(value string, defaultValue, maxValue int, label string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", label)
	}
	if parsed > maxValue {
		return maxValue, nil
	}
	return parsed, nil
}

func nonNegativeInt(value, label string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", label)
	}
	return parsed, nil
}

func issueState(issue map[string]interface{}) string {
	for _, key := range []string{"state", "status_name"} {
		if value := stringFromAny(issue[key]); value != "" {
			return value
		}
	}
	statusID := nestedID(issue, "status")
	if statusID == "5" {
		return "closed"
	}
	return "open"
}

func firstStringValue(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringFromAny(data[key]); value != "" {
			return value
		}
	}
	return ""
}

func nestedName(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		nested, ok := data[key].(map[string]interface{})
		if !ok {
			continue
		}
		if value := firstStringValue(nested, "name", "title", "subject"); value != "" {
			return value
		}
	}
	return ""
}

func nestedID(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		nested, ok := data[key].(map[string]interface{})
		if !ok {
			continue
		}
		if value := stringFromAny(nested["id"]); value != "" {
			return value
		}
	}
	return ""
}

func nestedLoginOrName(data map[string]interface{}, key string) string {
	nested, ok := data[key].(map[string]interface{})
	if !ok {
		return ""
	}
	return firstStringValue(nested, "login", "name")
}

func joinedNames(data map[string]interface{}, keys ...string) string {
	values := []string{}
	for _, key := range keys {
		switch raw := data[key].(type) {
		case []interface{}:
			for _, item := range raw {
				if name := itemName(item); name != "" {
					values = append(values, name)
				}
			}
		case map[string]interface{}:
			if name := itemName(raw); name != "" {
				values = append(values, name)
			}
		}
		if len(values) > 0 {
			break
		}
	}
	sort.Strings(values)
	return strings.Join(values, ";")
}

func itemName(item interface{}) string {
	switch value := item.(type) {
	case map[string]interface{}:
		return firstStringValue(value, "name", "login", "title")
	case string:
		return value
	default:
		return stringFromAny(value)
	}
}

func stringFromAny(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func intFromAny(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		parsed, _ := strconv.Atoi(v.String())
		return parsed
	default:
		return 0
	}
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
