package feishu

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type BitableSchema struct {
	DryRun bool                 `json:"dry_run"`
	Tables []BitableTableSchema `json:"tables"`
}

type BitableTableSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Fields      []BitableField `json:"fields"`
}

type BitableField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type BitableRecords struct {
	DryRun     bool                       `json:"dry_run"`
	Repository string                     `json:"repository"`
	Tables     map[string][]BitableRecord `json:"tables"`
	Schema     []BitableTableSchema       `json:"schema"`
	Notes      []string                   `json:"notes,omitempty"`
}

type BitableRecord struct {
	UniqueKey string                 `json:"unique_key"`
	Fields    map[string]interface{} `json:"fields"`
}

func BuildBitableSchema(tables []string) BitableSchema {
	result := BitableSchema{DryRun: true}
	for _, table := range normalizeTables(tables) {
		result.Tables = append(result.Tables, schemaForTable(table))
	}
	return result
}

func BuildBitableRecords(report workflow.RepoReportResult, tables []string, docURL string) BitableRecords {
	tables = normalizeTables(tables)
	result := BitableRecords{
		DryRun:     true,
		Repository: report.Repository,
		Tables:     map[string][]BitableRecord{},
		Schema:     BuildBitableSchema(tables).Tables,
		Notes: []string{
			"Dry-run by default: +bitable-records does not call Feishu Bitable OpenAPI.",
			"Records use stable unique_key values so experimental +bitable-sync can search-before-update.",
			"Generated records are workflow-report summaries, not Feishu user-personalized records.",
		},
	}
	for _, table := range tables {
		switch table {
		case "reports":
			result.Tables[table] = reportRecords(report, docURL)
		case "issues":
			result.Tables[table] = issueRecords(report)
		case "prs":
			result.Tables[table] = prRecords(report)
		case "contributors":
			result.Tables[table] = contributorRecords(report)
		case "tasks":
			result.Tables[table] = taskRecords(report, docURL)
		}
	}
	return result
}

func normalizeTables(tables []string) []string {
	if len(tables) == 0 {
		tables = parseList(defaultTables)
	}
	allowed := map[string]bool{"reports": true, "issues": true, "prs": true, "contributors": true, "tasks": true}
	seen := map[string]bool{}
	result := []string{}
	for _, table := range tables {
		if table == "pulls" {
			table = "prs"
		}
		if !allowed[table] || seen[table] {
			continue
		}
		seen[table] = true
		result = append(result, table)
	}
	return result
}

func schemaForTable(table string) BitableTableSchema {
	switch table {
	case "issues":
		return BitableTableSchema{
			Name:        "issues",
			Description: "Issue summary and risk buckets from workflow repo report output.",
			Fields: bitableFields([]string{
				"unique_key:text",
				"repository:text",
				"issue_group:single_select",
				"priority:single_select",
				"count:number",
				"risk_reason:multi_text",
				"recommended_action:multi_text",
				"gitlink_url:url",
			}),
		}
	case "prs":
		return BitableTableSchema{
			Name:        "prs",
			Description: "Pull request summary and review-risk buckets from workflow repo report output.",
			Fields: bitableFields([]string{
				"unique_key:text",
				"repository:text",
				"pr_group:single_select",
				"risk_level:single_select",
				"count:number",
				"review_focus:multi_text",
				"recommended_action:multi_text",
				"gitlink_url:url",
			}),
		}
	case "contributors":
		return BitableTableSchema{
			Name:        "contributors",
			Description: "Role-oriented contributor summary records derived from workflow report signals.",
			Fields: bitableFields([]string{
				"unique_key:text",
				"repository:text",
				"contributor:text",
				"role:single_select",
				"open_items:number",
				"risk_items:number",
				"recommended_action:multi_text",
				"gitlink_url:url",
			}),
		}
	case "tasks":
		return BitableTableSchema{
			Name:        "tasks",
			Description: "Task candidates derived from workflow recommendations, high-risk issues, PRs, and missing information.",
			Fields: bitableFields([]string{
				"unique_key:text",
				"repository:text",
				"task_title:text",
				"task_type:single_select",
				"priority:single_select",
				"source_type:single_select",
				"source_key:text",
				"recommended_owner:text",
				"status:single_select",
				"due_hint:text",
				"gitlink_url:url",
			}),
		}
	default:
		return BitableTableSchema{
			Name:        "reports",
			Description: "One row per repository workflow report.",
			Fields: bitableFields([]string{
				"unique_key:text",
				"repository:text",
				"health_score:number",
				"risk_level:single_select",
				"report_score:number",
				"issue_total:number",
				"issue_high_risk:number",
				"issue_missing_info:number",
				"pr_total:number",
				"pr_high_risk:number",
				"review_focus_count:number",
				"generated_at:datetime",
				"source:text",
				"doc_url:url",
			}),
		}
	}
}

func bitableFields(specs []string) []BitableField {
	fields := make([]BitableField, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		fieldType := "text"
		if len(parts) == 2 {
			fieldType = parts[1]
		}
		fields = append(fields, BitableField{Name: parts[0], Type: fieldType})
	}
	return fields
}

func reportRecords(report workflow.RepoReportResult, docURL string) []BitableRecord {
	healthScore := interface{}(nil)
	if report.Health != nil {
		healthScore = report.Health.HealthScore
	}
	fields := map[string]interface{}{
		"unique_key":         stableKey("report", report.Repository),
		"repository":         report.Repository,
		"health_score":       healthScore,
		"risk_level":         report.RiskLevel,
		"report_score":       report.ReportScore,
		"issue_total":        report.IssueSummary.Total,
		"issue_high_risk":    report.IssueSummary.HighRisk,
		"issue_missing_info": report.IssueSummary.MissingInfo,
		"pr_total":           report.PRSummary.Total,
		"pr_high_risk":       report.PRSummary.HighRisk,
		"review_focus_count": len(report.PRSummary.ReviewFocus),
		"generated_at":       time.Now().UTC().Format(time.RFC3339),
		"source":             report.Source,
	}
	if strings.TrimSpace(docURL) != "" {
		fields["doc_url"] = strings.TrimSpace(docURL)
	}
	return []BitableRecord{{UniqueKey: fields["unique_key"].(string), Fields: fields}}
}

func issueRecords(report workflow.RepoReportResult) []BitableRecord {
	records := []BitableRecord{}
	keys := sortedIntMapKeys(report.IssueSummary.ByPriority)
	for _, priority := range keys {
		fields := issueRecordFields(report, "priority", priority, report.IssueSummary.ByPriority[priority])
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	keys = sortedIntMapKeys(report.IssueSummary.ByType)
	for _, issueType := range keys {
		fields := issueRecordFields(report, "type", issueType, report.IssueSummary.ByType[issueType])
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	if len(records) == 0 {
		fields := issueRecordFields(report, "summary", "total", report.IssueSummary.Total)
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	return records
}

func issueRecordFields(report workflow.RepoReportResult, groupType string, group string, count int) map[string]interface{} {
	recommended := []string{"Review issue triage details in GitLink."}
	if report.IssueSummary.MissingInfo > 0 {
		recommended = append(recommended, "Request missing reproduction steps, logs, or environment details.")
	}
	if report.IssueSummary.HighRisk > 0 {
		recommended = append(recommended, "Prioritize high-risk issue review.")
	}
	fields := map[string]interface{}{
		"unique_key":         stableKey("issue", report.Repository, groupType, group),
		"repository":         report.Repository,
		"issue_group":        groupType + ":" + group,
		"priority":           group,
		"count":              count,
		"risk_reason":        []string{fmt.Sprintf("high_risk=%d", report.IssueSummary.HighRisk), fmt.Sprintf("missing_info=%d", report.IssueSummary.MissingInfo)},
		"recommended_action": uniqueDigestStrings(recommended),
	}
	if repoURL := gitlinkRepoURL(report.Repository); repoURL != "" {
		fields["gitlink_url"] = repoURL + "/issues"
	}
	return fields
}

func prRecords(report workflow.RepoReportResult) []BitableRecord {
	records := []BitableRecord{}
	keys := sortedIntMapKeys(report.PRSummary.ByRisk)
	for _, risk := range keys {
		fields := prRecordFields(report, "risk", risk, report.PRSummary.ByRisk[risk])
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	keys = sortedIntMapKeys(report.PRSummary.ByType)
	for _, changeType := range keys {
		fields := prRecordFields(report, "change_type", changeType, report.PRSummary.ByType[changeType])
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	if len(records) == 0 {
		fields := prRecordFields(report, "summary", "total", report.PRSummary.Total)
		records = append(records, BitableRecord{UniqueKey: fields["unique_key"].(string), Fields: fields})
	}
	return records
}

func prRecordFields(report workflow.RepoReportResult, groupType string, group string, count int) map[string]interface{} {
	recommended := []string{"Review PR focus items in GitLink."}
	if report.PRSummary.HighRisk > 0 {
		recommended = append(recommended, "Prioritize high-risk pull requests before lower-risk changes.")
	}
	fields := map[string]interface{}{
		"unique_key":         stableKey("pr", report.Repository, groupType, group),
		"repository":         report.Repository,
		"pr_group":           groupType + ":" + group,
		"risk_level":         group,
		"count":              count,
		"review_focus":       report.PRSummary.ReviewFocus,
		"recommended_action": uniqueDigestStrings(recommended),
	}
	if repoURL := gitlinkRepoURL(report.Repository); repoURL != "" {
		fields["gitlink_url"] = repoURL + "/pulls"
	}
	return fields
}

func contributorRecords(report workflow.RepoReportResult) []BitableRecord {
	openItems := report.PRSummary.Total + report.IssueSummary.Total
	riskItems := report.PRSummary.HighRisk + report.IssueSummary.HighRisk
	fields := map[string]interface{}{
		"unique_key":         stableKey("contributor", report.Repository, "role-oriented"),
		"repository":         report.Repository,
		"contributor":        "role-oriented digest",
		"role":               "contributor",
		"open_items":         openItems,
		"risk_items":         riskItems,
		"recommended_action": BuildContributorDigest(report, "").NextSteps,
	}
	if repoURL := gitlinkRepoURL(report.Repository); repoURL != "" {
		fields["gitlink_url"] = repoURL
	}
	return []BitableRecord{{UniqueKey: fields["unique_key"].(string), Fields: fields}}
}

func taskRecords(report workflow.RepoReportResult, docURL string) []BitableRecord {
	tasks := BuildTaskCandidates(report, docURL)
	records := make([]BitableRecord, 0, len(tasks))
	for _, task := range tasks {
		fields := map[string]interface{}{
			"unique_key":        task.UniqueKey,
			"repository":        task.Repository,
			"task_title":        task.Title,
			"task_type":         task.TaskType,
			"priority":          task.Priority,
			"source_type":       task.SourceType,
			"source_key":        task.SourceKey,
			"recommended_owner": task.RecommendedOwner,
			"status":            task.Status,
			"due_hint":          task.DueHint,
		}
		if task.GitLinkURL != "" {
			fields["gitlink_url"] = task.GitLinkURL
		}
		records = append(records, BitableRecord{UniqueKey: task.UniqueKey, Fields: fields})
	}
	return records
}

func sortedIntMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stableKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		part = strings.NewReplacer(" ", "-", "/", "_", "\\", "_", ":", "-", "#", "").Replace(part)
		if part == "" {
			part = "unknown"
		}
		cleaned = append(cleaned, part)
	}
	return strings.Join(cleaned, ":")
}

func renderBitableSchema(w io.Writer, schema BitableSchema, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeSchemaMarkdown(w, schema)
	case "table":
		return writeSchemaTable(w, schema)
	default:
		return writeJSON(w, schema)
	}
}

func renderBitableRecords(w io.Writer, records BitableRecords, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeRecordsMarkdown(w, records)
	case "table":
		return writeRecordsTable(w, records)
	default:
		return writeJSON(w, records)
	}
}

func writeSchemaMarkdown(w io.Writer, schema BitableSchema) error {
	if _, err := fmt.Fprint(w, "# Feishu Bitable Schema\n\nDry run: `true`\n\n"); err != nil {
		return err
	}
	for _, table := range schema.Tables {
		if _, err := fmt.Fprintf(w, "## %s\n\n%s\n\n", table.Name, table.Description); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "| Field | Type | Description |\n| --- | --- | --- |"); err != nil {
			return err
		}
		for _, field := range table.Fields {
			if _, err := fmt.Fprintf(w, "| %s | %s | %s |\n", field.Name, field.Type, field.Description); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func writeSchemaTable(w io.Writer, schema BitableSchema) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TABLE\tFIELD\tTYPE\tDESCRIPTION"); err != nil {
		return err
	}
	for _, table := range schema.Tables {
		for _, field := range table.Fields {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", table.Name, field.Name, field.Type, field.Description); err != nil {
				return err
			}
		}
	}
	return tw.Flush()
}

func writeRecordsMarkdown(w io.Writer, records BitableRecords) error {
	if _, err := fmt.Fprint(w, "# Feishu Bitable Records\n\nDry run: `true`\n\n"); err != nil {
		return err
	}
	for _, note := range records.Notes {
		if _, err := fmt.Fprintf(w, "- %s\n", note); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	tableNames := sortedTableNames(records.Tables)
	for _, table := range tableNames {
		rows := records.Tables[table]
		if _, err := fmt.Fprintf(w, "## %s\n\nRecords: `%d`\n\n", table, len(rows)); err != nil {
			return err
		}
	}
	return nil
}

func writeRecordsTable(w io.Writer, records BitableRecords) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TABLE\tRECORDS"); err != nil {
		return err
	}
	for _, table := range sortedTableNames(records.Tables) {
		if _, err := fmt.Fprintf(tw, "%s\t%d\n", table, len(records.Tables[table])); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func sortedTableNames(records map[string][]BitableRecord) []string {
	tableNames := make([]string, 0, len(records))
	for table := range records {
		tableNames = append(tableNames, table)
	}
	sort.Strings(tableNames)
	return tableNames
}
