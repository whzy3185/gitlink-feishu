package feishu

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type BitableSyncOptions struct {
	AppID        string            `json:"-"`
	AppSecret    string            `json:"-"`
	BaseAppToken string            `json:"-"`
	TableIDs     map[string]string `json:"-"`
	Tables       []string          `json:"tables"`
	Send         bool              `json:"send"`
	DryRun       bool              `json:"dry_run"`
}

type BitableSyncOutput struct {
	Mode         string                   `json:"mode"`
	Send         bool                     `json:"send"`
	DryRun       bool                     `json:"dry_run"`
	BaseAppToken string                   `json:"base_app_token,omitempty"`
	Tables       []BitableSyncTableResult `json:"tables"`
	Warnings     []string                 `json:"warnings,omitempty"`
}

type BitableSyncTableResult struct {
	Table       string                    `json:"table"`
	TableID     string                    `json:"table_id,omitempty"`
	RecordCount int                       `json:"record_count"`
	Created     int                       `json:"created,omitempty"`
	Updated     int                       `json:"updated,omitempty"`
	Skipped     bool                      `json:"skipped,omitempty"`
	Error       string                    `json:"error,omitempty"`
	Records     []BitableSyncRecordResult `json:"records,omitempty"`
}

type BitableSyncRecordResult struct {
	UniqueKey string `json:"unique_key"`
	Action    string `json:"action"`
	RecordID  string `json:"record_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

func bitableSyncOptionsFromContext(ctx *common.RuntimeContext) (BitableSyncOptions, error) {
	opts := BitableSyncOptions{
		AppID:        firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID")),
		AppSecret:    firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET")),
		BaseAppToken: firstNonEmpty(ctx.Arg("base-app-token"), os.Getenv("FEISHU_BASE_APP_TOKEN")),
		TableIDs: map[string]string{
			"reports":      firstNonEmpty(ctx.Arg("report-table-id"), os.Getenv("FEISHU_REPORT_TABLE_ID")),
			"issues":       firstNonEmpty(ctx.Arg("issue-table-id"), os.Getenv("FEISHU_ISSUE_TABLE_ID")),
			"prs":          firstNonEmpty(ctx.Arg("pr-table-id"), os.Getenv("FEISHU_PR_TABLE_ID")),
			"contributors": firstNonEmpty(ctx.Arg("contributor-table-id"), os.Getenv("FEISHU_CONTRIBUTOR_TABLE_ID")),
			"tasks":        firstNonEmpty(ctx.Arg("task-table-id"), os.Getenv("FEISHU_TASK_TABLE_ID")),
		},
		Tables: normalizeTables(parseList(firstNonEmpty(ctx.Arg("tables"), defaultTables))),
		Send:   parseBool(ctx.Arg("send")),
		DryRun: parseBool(ctx.Arg("dry-run")),
	}
	if opts.Send && opts.DryRun {
		return BitableSyncOptions{}, fmt.Errorf("--send and --dry-run cannot be used together")
	}
	if opts.Send {
		if opts.AppID == "" {
			return BitableSyncOptions{}, fmt.Errorf("--send requires --app-id or FEISHU_APP_ID")
		}
		if opts.AppSecret == "" {
			return BitableSyncOptions{}, fmt.Errorf("--send requires --app-secret or FEISHU_APP_SECRET")
		}
		if opts.BaseAppToken == "" {
			return BitableSyncOptions{}, fmt.Errorf("--send requires --base-app-token or FEISHU_BASE_APP_TOKEN")
		}
	}
	return opts, nil
}

func syncBitableOrPreview(ctx *common.RuntimeContext, opts BitableSyncOptions, records BitableRecords) error {
	output := BitableSyncOutput{
		Mode:         "preview",
		Send:         opts.Send,
		DryRun:       !opts.Send,
		BaseAppToken: redactToken(opts.BaseAppToken),
		Warnings: []string{
			"Experimental: Feishu Base writes require self-built app Base/Bitable scopes and table-level access.",
			"Records are matched by the unique_key field. No records are deleted.",
		},
	}
	for _, table := range opts.Tables {
		output.Tables = append(output.Tables, BitableSyncTableResult{
			Table:       table,
			TableID:     redactToken(opts.TableIDs[table]),
			RecordCount: len(records.Tables[table]),
		})
	}
	if !opts.Send {
		return renderBitableSyncOutput(os.Stdout, output, formatOrDefault(ctx, "markdown"))
	}

	client := NewOpenAPIClient(nil)
	token, err := client.TenantAccessToken(context.Background(), opts.AppID, opts.AppSecret)
	if err != nil {
		return err
	}
	output.Mode = "sent"
	output.DryRun = false
	for i, tableResult := range output.Tables {
		tableID := opts.TableIDs[tableResult.Table]
		if strings.TrimSpace(tableID) == "" {
			output.Tables[i].Skipped = true
			output.Tables[i].Error = fmt.Sprintf("missing table ID for %s", tableResult.Table)
			continue
		}
		for _, record := range records.Tables[tableResult.Table] {
			result := BitableSyncRecordResult{UniqueKey: record.UniqueKey}
			fields := normalizeBitableWriteFields(record.Fields)
			search, err := client.SearchBitableRecord(context.Background(), token.Value, opts.BaseAppToken, tableID, record.UniqueKey)
			if err != nil {
				output.Warnings = append(output.Warnings, diagnoseOpenAPIError(err, "bitable", tableResult.Table)+"; falling back to create-only for this record")
				created, createErr := client.CreateBitableRecord(context.Background(), token.Value, opts.BaseAppToken, tableID, fields)
				if createErr != nil {
					result.Action = "create"
					result.Error = diagnoseOpenAPIError(createErr, "bitable", tableResult.Table)
					output.Tables[i].Records = append(output.Tables[i].Records, result)
					_ = renderBitableSyncOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
					return createErr
				}
				result.Action = "create"
				result.RecordID = redactToken(created.RecordID)
				output.Tables[i].Created++
				output.Tables[i].Records = append(output.Tables[i].Records, result)
				continue
			}
			if search.Found {
				updated, err := client.UpdateBitableRecord(context.Background(), token.Value, opts.BaseAppToken, tableID, search.RecordID, fields)
				if err != nil {
					result.Action = "update"
					result.RecordID = redactToken(search.RecordID)
					result.Error = diagnoseOpenAPIError(err, "bitable", tableResult.Table)
					output.Tables[i].Records = append(output.Tables[i].Records, result)
					_ = renderBitableSyncOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
					return err
				}
				result.Action = "update"
				result.RecordID = redactToken(updated.RecordID)
				output.Tables[i].Updated++
			} else {
				created, err := client.CreateBitableRecord(context.Background(), token.Value, opts.BaseAppToken, tableID, fields)
				if err != nil {
					result.Action = "create"
					result.Error = diagnoseOpenAPIError(err, "bitable", tableResult.Table)
					output.Tables[i].Records = append(output.Tables[i].Records, result)
					_ = renderBitableSyncOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
					return err
				}
				result.Action = "create"
				result.RecordID = redactToken(created.RecordID)
				output.Tables[i].Created++
			}
			output.Tables[i].Records = append(output.Tables[i].Records, result)
		}
	}
	return renderBitableSyncOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
}

func normalizeBitableWriteFields(fields map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		switch typed := value.(type) {
		case []string:
			normalized[key] = strings.Join(typed, "\n")
		case []interface{}:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				parts = append(parts, fmt.Sprint(item))
			}
			normalized[key] = strings.Join(parts, "\n")
		default:
			normalized[key] = value
		}
	}
	return normalized
}

func renderBitableSyncOutput(w io.Writer, output BitableSyncOutput, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeBitableSyncMarkdown(w, output)
	case "table":
		return writeBitableSyncTable(w, output)
	default:
		return writeJSON(w, output)
	}
}

func writeBitableSyncMarkdown(w io.Writer, output BitableSyncOutput) error {
	if _, err := fmt.Fprintf(w, "# Feishu Bitable Sync %s\n\n", titleWord(output.Mode)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Send: `%t`\n- Dry run: `%t`\n- Base app token: `%s`\n\n", output.Send, output.DryRun, firstNonEmpty(output.BaseAppToken, "not configured")); err != nil {
		return err
	}
	for _, warning := range output.Warnings {
		if _, err := fmt.Fprintf(w, "- %s\n", warning); err != nil {
			return err
		}
	}
	if len(output.Warnings) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	for _, table := range output.Tables {
		if _, err := fmt.Fprintf(w, "## %s\n\n- Table ID: `%s`\n- Records: `%d`\n- Created: `%d`\n- Updated: `%d`\n", table.Table, firstNonEmpty(table.TableID, "not configured"), table.RecordCount, table.Created, table.Updated); err != nil {
			return err
		}
		if table.Skipped || table.Error != "" {
			if _, err := fmt.Fprintf(w, "- Skipped: `%t`\n- Error: `%s`\n", table.Skipped, table.Error); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func writeBitableSyncTable(w io.Writer, output BitableSyncOutput) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TABLE\tTABLE_ID\tRECORDS\tCREATED\tUPDATED\tSKIPPED\tERROR"); err != nil {
		return err
	}
	for _, table := range output.Tables {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%t\t%s\n", table.Table, firstNonEmpty(table.TableID, "not configured"), table.RecordCount, table.Created, table.Updated, table.Skipped, table.Error); err != nil {
			return err
		}
	}
	return tw.Flush()
}
