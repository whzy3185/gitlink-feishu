package feishu

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

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
	DryRun bool                       `json:"dry_run"`
	Tables map[string][]BitableRecord `json:"tables"`
	Schema []BitableTableSchema       `json:"schema"`
	Notes  []string                   `json:"notes,omitempty"`
}

type BitableRecord struct {
	Fields map[string]interface{} `json:"fields"`
}

func BuildBitableSchema(tables []string) BitableSchema {
	result := BitableSchema{DryRun: true}
	for _, table := range normalizeTables(tables) {
		result.Tables = append(result.Tables, schemaForTable(table))
	}
	return result
}

func BuildBitableRecords(report workflow.RepoReportResult, tables []string) BitableRecords {
	tables = normalizeTables(tables)
	result := BitableRecords{
		DryRun: true,
		Tables: map[string][]BitableRecord{},
		Schema: BuildBitableSchema(tables).Tables,
		Notes: []string{
			"Dry-run only: this command does not call Feishu Bitable OpenAPI.",
			"Use these records to validate table shape before adding app authentication and upsert behavior.",
		},
	}
	for _, table := range tables {
		switch table {
		case "reports":
			result.Tables[table] = reportRecords(report)
		case "issues":
			result.Tables[table] = issueRecords(report)
		case "prs":
			result.Tables[table] = prRecords(report)
		case "contributors":
			result.Tables[table] = []BitableRecord{}
		}
	}
	return result
}

func normalizeTables(tables []string) []string {
	if len(tables) == 0 {
		tables = parseList(defaultTables)
	}
	allowed := map[string]bool{"issues": true, "prs": true, "contributors": true, "reports": true}
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
			Description: "Issue summary buckets from workflow repo report output.",
			Fields: []BitableField{
				{Name: "repository", Type: "text"},
				{Name: "bucket_type", Type: "single_select", Description: "type or priority"},
				{Name: "bucket", Type: "text"},
				{Name: "count", Type: "number"},
				{Name: "high_risk_total", Type: "number"},
				{Name: "missing_info_total", Type: "number"},
			},
		}
	case "prs":
		return BitableTableSchema{
			Name:        "prs",
			Description: "Pull request summary buckets from workflow repo report output.",
			Fields: []BitableField{
				{Name: "repository", Type: "text"},
				{Name: "bucket_type", Type: "single_select", Description: "change_type or risk"},
				{Name: "bucket", Type: "text"},
				{Name: "count", Type: "number"},
				{Name: "high_risk_total", Type: "number"},
				{Name: "review_focus", Type: "multi_text"},
			},
		}
	case "contributors":
		return BitableTableSchema{
			Name:        "contributors",
			Description: "Reserved table for contributor activity once workflow JSON includes contributor data.",
			Fields: []BitableField{
				{Name: "repository", Type: "text"},
				{Name: "login", Type: "text"},
				{Name: "role", Type: "single_select"},
				{Name: "activity_count", Type: "number"},
			},
		}
	default:
		return BitableTableSchema{
			Name:        "reports",
			Description: "One row per repository workflow report.",
			Fields: []BitableField{
				{Name: "repository", Type: "text"},
				{Name: "report_score", Type: "number"},
				{Name: "risk_level", Type: "single_select"},
				{Name: "health_score", Type: "number"},
				{Name: "issues_total", Type: "number"},
				{Name: "high_risk_issues", Type: "number"},
				{Name: "prs_total", Type: "number"},
				{Name: "high_risk_prs", Type: "number"},
				{Name: "source", Type: "text"},
				{Name: "recommendations", Type: "multi_text"},
			},
		}
	}
}

func reportRecords(report workflow.RepoReportResult) []BitableRecord {
	healthScore := interface{}(nil)
	if report.Health != nil {
		healthScore = report.Health.HealthScore
	}
	return []BitableRecord{{
		Fields: map[string]interface{}{
			"repository":       report.Repository,
			"report_score":     report.ReportScore,
			"risk_level":       report.RiskLevel,
			"health_score":     healthScore,
			"issues_total":     report.IssueSummary.Total,
			"high_risk_issues": report.IssueSummary.HighRisk,
			"prs_total":        report.PRSummary.Total,
			"high_risk_prs":    report.PRSummary.HighRisk,
			"source":           report.Source,
			"recommendations":  report.Recommendations,
		},
	}}
}

func issueRecords(report workflow.RepoReportResult) []BitableRecord {
	records := []BitableRecord{}
	records = appendCountMapRecords(records, report.Repository, "type", report.IssueSummary.ByType, map[string]interface{}{
		"high_risk_total":    report.IssueSummary.HighRisk,
		"missing_info_total": report.IssueSummary.MissingInfo,
	})
	records = appendCountMapRecords(records, report.Repository, "priority", report.IssueSummary.ByPriority, map[string]interface{}{
		"high_risk_total":    report.IssueSummary.HighRisk,
		"missing_info_total": report.IssueSummary.MissingInfo,
	})
	if len(records) == 0 {
		records = append(records, BitableRecord{Fields: map[string]interface{}{
			"repository":         report.Repository,
			"bucket_type":        "summary",
			"bucket":             "total",
			"count":              report.IssueSummary.Total,
			"high_risk_total":    report.IssueSummary.HighRisk,
			"missing_info_total": report.IssueSummary.MissingInfo,
		}})
	}
	return records
}

func prRecords(report workflow.RepoReportResult) []BitableRecord {
	records := []BitableRecord{}
	records = appendCountMapRecords(records, report.Repository, "change_type", report.PRSummary.ByType, map[string]interface{}{
		"high_risk_total": report.PRSummary.HighRisk,
		"review_focus":    report.PRSummary.ReviewFocus,
	})
	records = appendCountMapRecords(records, report.Repository, "risk", report.PRSummary.ByRisk, map[string]interface{}{
		"high_risk_total": report.PRSummary.HighRisk,
		"review_focus":    report.PRSummary.ReviewFocus,
	})
	if len(records) == 0 {
		records = append(records, BitableRecord{Fields: map[string]interface{}{
			"repository":      report.Repository,
			"bucket_type":     "summary",
			"bucket":          "total",
			"count":           report.PRSummary.Total,
			"high_risk_total": report.PRSummary.HighRisk,
			"review_focus":    report.PRSummary.ReviewFocus,
		}})
	}
	return records
}

func appendCountMapRecords(records []BitableRecord, repository string, bucketType string, values map[string]int, extras map[string]interface{}) []BitableRecord {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fields := map[string]interface{}{
			"repository":  repository,
			"bucket_type": bucketType,
			"bucket":      key,
			"count":       values[key],
		}
		for extraKey, extraValue := range extras {
			fields[extraKey] = extraValue
		}
		records = append(records, BitableRecord{Fields: fields})
	}
	return records
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
	tableNames := make([]string, 0, len(records.Tables))
	for table := range records.Tables {
		tableNames = append(tableNames, table)
	}
	sort.Strings(tableNames)
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
	tableNames := make([]string, 0, len(records.Tables))
	for table := range records.Tables {
		tableNames = append(tableNames, table)
	}
	sort.Strings(tableNames)
	for _, table := range tableNames {
		if _, err := fmt.Fprintf(tw, "%s\t%d\n", table, len(records.Tables[table])); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func joinStrings(values []string) string {
	return strings.Join(values, ", ")
}
