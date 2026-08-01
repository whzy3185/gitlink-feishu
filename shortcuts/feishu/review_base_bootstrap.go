package feishu

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	defaultReviewBaseName  = "GitLink Review Queue - Round 2 Test"
	defaultReviewTableName = "Review WorkItems"
	defaultReviewViewName  = "Review Queue"
	defaultReviewEnvPath   = ".local/feishu-review-resources.env.ps1"
)

type ReviewBaseBootstrapOptions struct {
	AppID          string `json:"-"`
	AppSecret      string `json:"-"`
	BaseName       string `json:"base_name"`
	TableName      string `json:"table_name"`
	ViewName       string `json:"view_name"`
	FolderToken    string `json:"-"`
	TimeZone       string `json:"time_zone"`
	ResumeAppToken string `json:"-"`
	OutputEnv      string `json:"output_env"`
	Send           bool   `json:"send"`
}

type ReviewBaseBootstrapOutput struct {
	SchemaVersion    string             `json:"schema_version"`
	Mode             string             `json:"mode"`
	Status           string             `json:"status"`
	BaseName         string             `json:"base_name"`
	TableName        string             `json:"table_name"`
	ViewName         string             `json:"view_name"`
	Fields           []BitableFieldSpec `json:"fields"`
	OutputEnv        string             `json:"output_env"`
	BaseAction       string             `json:"base_action"`
	TableAction      string             `json:"table_action"`
	BaseTokenHash    string             `json:"base_token_hash,omitempty"`
	TableIDHash      string             `json:"table_id_hash,omitempty"`
	FeishuWrites     int                `json:"feishu_writes"`
	GitLinkWrites    int                `json:"gitlink_writes"`
	RecoverySaved    bool               `json:"recovery_saved,omitempty"`
	DefaultTableKept bool               `json:"default_table_kept"`
	Warnings         []string           `json:"warnings,omitempty"`
}

func reviewBaseBootstrapOptionsFromContext(ctx *common.RuntimeContext) (ReviewBaseBootstrapOptions, error) {
	opts := ReviewBaseBootstrapOptions{
		AppID:          firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID")),
		AppSecret:      firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET")),
		BaseName:       firstNonEmpty(ctx.Arg("base-name"), defaultReviewBaseName),
		TableName:      firstNonEmpty(ctx.Arg("table-name"), defaultReviewTableName),
		ViewName:       firstNonEmpty(ctx.Arg("view-name"), defaultReviewViewName),
		FolderToken:    firstNonEmpty(ctx.Arg("folder-token"), os.Getenv("FEISHU_REVIEW_BASE_FOLDER_TOKEN")),
		TimeZone:       firstNonEmpty(ctx.Arg("time-zone"), "Asia/Shanghai"),
		ResumeAppToken: firstNonEmpty(ctx.Arg("resume-app-token"), os.Getenv("FEISHU_REVIEW_BASE_APP_TOKEN")),
		OutputEnv:      firstNonEmpty(ctx.Arg("output-env"), defaultReviewEnvPath),
		Send:           parseBool(ctx.Arg("send")),
	}
	if !opts.Send {
		return opts, nil
	}
	if opts.AppID == "" || opts.AppSecret == "" {
		return ReviewBaseBootstrapOptions{}, fmt.Errorf("--send requires FEISHU_APP_ID and FEISHU_APP_SECRET (or matching flags)")
	}
	if info, err := os.Stat(opts.OutputEnv); err == nil && !info.IsDir() && opts.ResumeAppToken == "" {
		return ReviewBaseBootstrapOptions{}, fmt.Errorf("output env already exists; dot-source it or pass --resume-app-token before retrying")
	}
	return opts, nil
}

func reviewWorkItemFieldSpecs() []BitableFieldSpec {
	return []BitableFieldSpec{
		{FieldName: "unique_key", Type: 1},
		{FieldName: "pr_key", Type: 1},
		{FieldName: "repository", Type: 1},
		{FieldName: "pr_number", Type: 2},
		{FieldName: "review_stage", Type: 1},
		{FieldName: "decision", Type: 1},
		{FieldName: "collection_status", Type: 1},
		{FieldName: "head_sha", Type: 1},
		{FieldName: "source_fingerprint", Type: 1},
		{FieldName: "collaboration_status", Type: 1},
		{FieldName: "assigned_to", Type: 1},
		{FieldName: "due_at", Type: 1},
		{FieldName: "archived", Type: 7},
		{FieldName: "updated_at", Type: 1},
	}
}

func runReviewBaseBootstrap(runtime *common.RuntimeContext) error {
	opts, err := reviewBaseBootstrapOptionsFromContext(runtime)
	if err != nil {
		return err
	}
	output, bootstrapErr := bootstrapReviewBase(context.Background(), opts, NewOpenAPIClient(nil))
	if renderErr := writeJSON(os.Stdout, output); renderErr != nil {
		return renderErr
	}
	return bootstrapErr
}

func bootstrapReviewBase(
	ctx context.Context,
	opts ReviewBaseBootstrapOptions,
	client OpenAPIClient,
) (ReviewBaseBootstrapOutput, error) {
	output := ReviewBaseBootstrapOutput{
		SchemaVersion:    "feishu.review-base-bootstrap/v1",
		Mode:             "preview",
		Status:           "planned",
		BaseName:         opts.BaseName,
		TableName:        opts.TableName,
		ViewName:         opts.ViewName,
		Fields:           reviewWorkItemFieldSpecs(),
		OutputEnv:        filepath.Clean(opts.OutputEnv),
		BaseAction:       "create",
		TableAction:      "create",
		GitLinkWrites:    0,
		DefaultTableKept: true,
		Warnings: []string{
			"The blank default table created with the Base is retained; no Feishu resource is deleted.",
			"The output environment file contains resource identifiers and must remain local.",
		},
	}
	if strings.TrimSpace(opts.ResumeAppToken) != "" {
		output.BaseAction = "reuse"
	}
	if !opts.Send {
		return output, nil
	}

	output.Mode = "send"
	output.Status = "running"
	token, err := client.TenantAccessToken(ctx, opts.AppID, opts.AppSecret)
	if err != nil {
		output.Status = "failed"
		return output, err
	}
	appToken := strings.TrimSpace(opts.ResumeAppToken)
	if appToken == "" {
		created, createErr := client.CreateBitableApp(
			ctx,
			token.Value,
			opts.BaseName,
			opts.FolderToken,
			opts.TimeZone,
		)
		if createErr != nil {
			output.Status = "failed"
			return output, createErr
		}
		appToken = created.AppToken
		output.FeishuWrites++
		output.BaseAction = "created"
		output.BaseTokenHash = stableResourceHash(appToken)
		if saveErr := writeReviewResourceEnv(opts.OutputEnv, appToken, ""); saveErr != nil {
			output.Status = "unknown_needs_reconciliation"
			return output, fmt.Errorf("Base created but recovery env could not be saved: %w", saveErr)
		}
		output.RecoverySaved = true
	} else {
		output.BaseTokenHash = stableResourceHash(appToken)
	}

	tables, err := client.ListBitableTables(ctx, token.Value, appToken)
	if err != nil {
		output.Status = "partial"
		return output, err
	}
	tableID := ""
	for _, table := range tables {
		if strings.TrimSpace(table.Name) == strings.TrimSpace(opts.TableName) {
			tableID = strings.TrimSpace(table.TableID)
			break
		}
	}
	if tableID == "" {
		created, createErr := client.CreateBitableTable(
			ctx,
			token.Value,
			appToken,
			opts.TableName,
			opts.ViewName,
			output.Fields,
		)
		if createErr != nil {
			output.Status = "partial"
			return output, createErr
		}
		tableID = created.TableID
		output.FeishuWrites++
		output.TableAction = "created"
	} else {
		output.TableAction = "reused"
	}
	output.TableIDHash = stableResourceHash(tableID)
	if err := writeReviewResourceEnv(opts.OutputEnv, appToken, tableID); err != nil {
		output.Status = "unknown_needs_reconciliation"
		return output, fmt.Errorf("resources created but final environment file could not be saved: %w", err)
	}
	output.RecoverySaved = true
	output.Status = "completed"
	return output, nil
}

func writeReviewResourceEnv(path, appToken, tableID string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || strings.TrimSpace(path) == "" {
		return fmt.Errorf("output env path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content := "# Local Review collaboration resources. Do not commit.\r\n" +
		"$env:FEISHU_REVIEW_BASE_APP_TOKEN = '" + escapePowerShellSingleQuoted(appToken) + "'\r\n" +
		"$env:FEISHU_REVIEW_TABLE_ID = '" + escapePowerShellSingleQuoted(tableID) + "'\r\n"
	return os.WriteFile(path, []byte(content), 0o600)
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "'", "''")
}

func stableResourceHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return fmt.Sprintf("%x", sum[:6])
}
