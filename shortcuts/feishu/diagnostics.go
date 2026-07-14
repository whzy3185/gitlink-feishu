package feishu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type DiagnosticOutput struct {
	Mode     string            `json:"mode"`
	Remote   bool              `json:"remote"`
	Layer    string            `json:"layer"`
	Summary  DiagnosticSummary `json:"summary"`
	Checks   []DiagnosticCheck `json:"checks"`
	Warnings []string          `json:"warnings,omitempty"`
}

type DiagnosticSummary struct {
	Passed  int `json:"passed"`
	Warned  int `json:"warned"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type DiagnosticCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Target   string `json:"target,omitempty"`
	Value    string `json:"value,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

func runFeishuAppCheck(ctx *common.RuntimeContext) error {
	output := DiagnosticOutput{
		Mode:   "check",
		Remote: parseBool(ctx.Arg("remote")),
		Layer:  "app",
		Warnings: []string{
			"Diagnostic commands never write Feishu resources or GitLink resources.",
			"Use --remote to call Feishu OpenAPI read/check endpoints.",
		},
	}
	webhookURL := firstNonEmpty(ctx.Arg("webhook-url"), os.Getenv("FEISHU_WEBHOOK_URL"))
	webhookSecret := firstNonEmpty(ctx.Arg("secret"), os.Getenv("FEISHU_WEBHOOK_SECRET"))
	appID := firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID"))
	appSecret := firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET"))

	if strings.TrimSpace(webhookURL) == "" {
		output.addCheck(warnCheck("custom bot webhook", "FEISHU_WEBHOOK_URL", "missing", "stable webhook send commands need FEISHU_WEBHOOK_URL"))
	} else if err := validateWebhookURL(webhookURL); err != nil {
		output.addCheck(failCheck("custom bot webhook", "FEISHU_WEBHOOK_URL", redactWebhookURL(webhookURL), err.Error(), "copy the full custom bot webhook URL from the Feishu group bot settings"))
	} else {
		output.addCheck(passCheck("custom bot webhook", "FEISHU_WEBHOOK_URL", redactWebhookURL(webhookURL), "configured"))
	}
	if strings.TrimSpace(webhookSecret) == "" {
		output.addCheck(warnCheck("custom bot signing secret", "FEISHU_WEBHOOK_SECRET", "missing", "only required when the custom bot enables signature verification"))
	} else {
		output.addCheck(passCheck("custom bot signing secret", "FEISHU_WEBHOOK_SECRET", redactToken(webhookSecret), "configured"))
	}
	output.addCheck(requiredSecretCheck("self-built app id", "FEISHU_APP_ID", appID, "required for DocX, Bitable, and Task OpenAPI validation"))
	output.addCheck(requiredSecretCheck("self-built app secret", "FEISHU_APP_SECRET", appSecret, "required for tenant_access_token"))
	if output.Remote && output.Summary.Failed == 0 {
		output.remoteTenantToken(appID, appSecret)
	} else if output.Remote {
		output.addCheck(skipCheck("tenant_access_token", "Feishu OpenAPI", "skipped because app credentials are incomplete"))
	}
	return finishDiagnostic(ctx, output)
}

func runFeishuDocCheck(ctx *common.RuntimeContext) error {
	opts := DocExportOptions{
		AppID:         firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID")),
		AppSecret:     firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET")),
		FolderToken:   firstNonEmpty(ctx.Arg("folder-token"), os.Getenv("FEISHU_FOLDER_TOKEN"), os.Getenv("FEISHU_DOC_FOLDER_TOKEN")),
		DocumentID:    firstNonEmpty(ctx.Arg("document-id"), os.Getenv("FEISHU_DOCUMENT_ID")),
		WikiURL:       firstNonEmpty(ctx.Arg("wiki-url"), os.Getenv("FEISHU_WIKI_URL")),
		WikiNodeToken: firstNonEmpty(ctx.Arg("wiki-node-token"), os.Getenv("FEISHU_WIKI_NODE_TOKEN")),
	}
	if opts.WikiNodeToken == "" && opts.WikiURL != "" {
		opts.WikiNodeToken = wikiNodeTokenFromURL(opts.WikiURL)
	}
	if opts.DocumentID == "" && opts.WikiURL != "" {
		opts.DocumentID = docxTokenFromURL(opts.WikiURL)
	}
	output := DiagnosticOutput{
		Mode:   "check",
		Remote: parseBool(ctx.Arg("remote")),
		Layer:  "docx",
		Warnings: []string{
			"Doc check does not append blocks or create documents.",
			"Actual DocX/Wiki writes still require --send on +doc-export.",
		},
	}
	output.addCheck(requiredSecretCheck("self-built app id", "FEISHU_APP_ID", opts.AppID, "required for DocX/Wiki OpenAPI"))
	output.addCheck(requiredSecretCheck("self-built app secret", "FEISHU_APP_SECRET", opts.AppSecret, "required for tenant_access_token"))

	targets := 0
	if opts.DocumentID != "" {
		targets++
		output.addCheck(passCheck("existing document target", "FEISHU_DOCUMENT_ID", redactToken(opts.DocumentID), "configured for append path"))
	}
	if opts.WikiNodeToken != "" {
		targets++
		output.addCheck(passCheck("wiki node target", "FEISHU_WIKI_NODE_TOKEN", redactToken(opts.WikiNodeToken), "configured or parsed from wiki URL"))
	}
	if opts.FolderToken != "" {
		targets++
		output.addCheck(passCheck("folder target", "FEISHU_FOLDER_TOKEN", redactToken(opts.FolderToken), "configured for create path"))
	}
	if opts.WikiURL != "" {
		output.addCheck(passCheck("wiki url", "FEISHU_WIKI_URL", redactResourceURL(opts.WikiURL), "configured"))
	}
	if targets == 0 {
		output.addCheck(failCheck("docx target", "DocX/Wiki", "missing", "no document, wiki, or folder target configured", "set FEISHU_DOCUMENT_ID, FEISHU_WIKI_URL, FEISHU_WIKI_NODE_TOKEN, or FEISHU_FOLDER_TOKEN"))
	}
	if output.Remote && output.Summary.Failed == 0 {
		token, ok := output.remoteTenantToken(opts.AppID, opts.AppSecret)
		if ok && opts.WikiNodeToken != "" {
			output.remoteWikiNode(token.Value, opts.WikiNodeToken)
		}
		if ok && opts.WikiNodeToken == "" {
			output.addCheck(skipCheck("wiki node read", "Feishu Wiki", "skipped because no wiki node token was configured"))
		}
		if ok && opts.DocumentID != "" {
			output.addCheck(skipCheck("document edit permission", "Feishu DocX", "not checked without writing blocks"))
		}
		if ok && opts.FolderToken != "" {
			output.addCheck(skipCheck("folder create permission", "Feishu Drive", "not checked without creating a document"))
		}
	} else if output.Remote {
		output.addCheck(skipCheck("remote docx check", "Feishu OpenAPI", "skipped because required config is incomplete"))
	}
	return finishDiagnostic(ctx, output)
}

func runFeishuBitableCheck(ctx *common.RuntimeContext) error {
	opts, err := bitableSyncOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	output := DiagnosticOutput{
		Mode:   "check",
		Remote: parseBool(ctx.Arg("remote")),
		Layer:  "bitable",
		Warnings: []string{
			"Bitable check does not create, update, or delete records.",
			"Remote mode searches a sentinel unique_key to verify table access and the unique_key field.",
		},
	}
	output.addCheck(requiredSecretCheck("self-built app id", "FEISHU_APP_ID", opts.AppID, "required for Base/Bitable OpenAPI"))
	output.addCheck(requiredSecretCheck("self-built app secret", "FEISHU_APP_SECRET", opts.AppSecret, "required for tenant_access_token"))
	output.addCheck(requiredSecretCheck("base app token", "FEISHU_BASE_APP_TOKEN", opts.BaseAppToken, "required for Bitable tables"))

	for _, table := range opts.Tables {
		envName := tableEnvName(table)
		tableID := opts.TableIDs[table]
		output.addCheck(requiredSecretCheck(table+" table id", envName, tableID, "required for "+table+" sync"))
		fields := schemaForTable(table).Fields
		output.addCheck(passCheck(table+" expected fields", table, strings.Join(fieldNames(fields), ","), "schema expected by +bitable-sync"))
	}
	if output.Remote && output.Summary.Failed == 0 {
		token, ok := output.remoteTenantToken(opts.AppID, opts.AppSecret)
		if ok {
			client := NewOpenAPIClient(http.DefaultClient)
			checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			for _, table := range opts.Tables {
				_, err := client.SearchBitableRecord(checkCtx, token.Value, opts.BaseAppToken, opts.TableIDs[table], "__gitlink_cli_check__")
				if err != nil {
					output.addCheck(failCheck("remote bitable table search", table, redactToken(opts.TableIDs[table]), sanitizeDiagnostic(diagnoseOpenAPIError(err, "bitable", table), opts.AppID, opts.AppSecret, opts.BaseAppToken, opts.TableIDs[table]), "grant Base scopes, share the Base with the app, and ensure unique_key exists"))
					continue
				}
				output.addCheck(passCheck("remote bitable table search", table, redactToken(opts.TableIDs[table]), "table accessible and unique_key search completed"))
			}
		}
	} else if output.Remote {
		output.addCheck(skipCheck("remote bitable check", "Feishu Base", "skipped because required config is incomplete"))
	}
	return finishDiagnostic(ctx, output)
}

func runFeishuTaskCheck(ctx *common.RuntimeContext) error {
	opts, err := taskCreateOptionsFromContext(&common.RuntimeContext{Args: map[string]string{
		"app-id":          ctx.Arg("app-id"),
		"app-secret":      ctx.Arg("app-secret"),
		"task-project-id": ctx.Arg("task-project-id"),
		"task-section-id": ctx.Arg("task-section-id"),
	}})
	if err != nil {
		return err
	}
	output := DiagnosticOutput{
		Mode:   "check",
		Remote: parseBool(ctx.Arg("remote")),
		Layer:  "task",
		Warnings: []string{
			"Task check does not create tasks.",
			"Current +task-create only sends basic summary and description.",
		},
	}
	output.addCheck(requiredSecretCheck("self-built app id", "FEISHU_APP_ID", opts.AppID, "required for Task OpenAPI"))
	output.addCheck(requiredSecretCheck("self-built app secret", "FEISHU_APP_SECRET", opts.AppSecret, "required for tenant_access_token"))
	if opts.TaskProjectID == "" {
		output.addCheck(warnCheck("task project id", "FEISHU_TASK_PROJECT_ID", "missing", "currently collected for future placement; not mapped into +task-create request body"))
	} else {
		output.addCheck(passCheck("task project id", "FEISHU_TASK_PROJECT_ID", redactToken(opts.TaskProjectID), "configured but not yet mapped into +task-create request body"))
	}
	if opts.TaskSectionID == "" {
		output.addCheck(warnCheck("task section id", "FEISHU_TASK_SECTION_ID", "missing", "currently collected for future placement; not mapped into +task-create request body"))
	} else {
		output.addCheck(passCheck("task section id", "FEISHU_TASK_SECTION_ID", redactToken(opts.TaskSectionID), "configured but not yet mapped into +task-create request body"))
	}
	output.addCheck(warnCheck("task dedupe", "Feishu Task", "not implemented", "dedupe is local unique_key only; no Feishu-side task search/linking"))
	if output.Remote && output.Summary.Failed == 0 {
		if _, ok := output.remoteTenantToken(opts.AppID, opts.AppSecret); ok {
			output.addCheck(skipCheck("remote task create permission", "Feishu Task", "not checked without creating a task"))
		} else {
			output.addCheck(skipCheck("remote task create permission", "Feishu Task", "not checked because tenant_access_token acquisition failed"))
		}
	} else if output.Remote {
		output.addCheck(skipCheck("remote task check", "Feishu Task", "skipped because app credentials are incomplete"))
	}
	return finishDiagnostic(ctx, output)
}

func finishDiagnostic(ctx *common.RuntimeContext, output DiagnosticOutput) error {
	if err := renderDiagnosticOutput(os.Stdout, output, formatOrDefault(ctx, "table")); err != nil {
		return err
	}
	if output.Summary.Failed > 0 {
		return fmt.Errorf("Feishu %s check failed: %d failed check(s)", output.Layer, output.Summary.Failed)
	}
	return nil
}

func (o *DiagnosticOutput) remoteTenantToken(appID, appSecret string) (TenantToken, bool) {
	client := NewOpenAPIClient(http.DefaultClient)
	checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	token, err := client.TenantAccessToken(checkCtx, appID, appSecret)
	if err != nil {
		o.addCheck(failCheck("tenant_access_token", "Feishu OpenAPI", "failed", sanitizeDiagnostic(diagnoseOpenAPIError(err, "tenant token", "self-built app"), appID, appSecret), "verify app_id/app_secret and app availability"))
		return TenantToken{}, false
	}
	o.addCheck(passCheck("tenant_access_token", "Feishu OpenAPI", fmt.Sprintf("expire=%d", token.Expire), "acquired"))
	return token, true
}

func (o *DiagnosticOutput) remoteWikiNode(token string, wikiNodeToken string) {
	client := NewOpenAPIClient(http.DefaultClient)
	checkCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	node, err := client.GetWikiNode(checkCtx, token, wikiNodeToken)
	if err != nil {
		o.addCheck(failCheck("wiki node read", "Feishu Wiki", redactToken(wikiNodeToken), sanitizeDiagnostic(diagnoseOpenAPIError(err, "docx", "wiki node"), wikiNodeToken, token), "grant Wiki/DocX scopes and make the target Wiki node visible to the app"))
		return
	}
	detail := "obj_type=" + firstNonEmpty(node.ObjType, "unknown")
	if node.Title != "" {
		detail += "; title=" + node.Title
	}
	o.addCheck(passCheck("wiki node read", "Feishu Wiki", redactToken(wikiNodeToken), detail))
}

func (o *DiagnosticOutput) addCheck(check DiagnosticCheck) {
	o.Checks = append(o.Checks, check)
	switch check.Status {
	case "pass":
		o.Summary.Passed++
	case "warn":
		o.Summary.Warned++
	case "fail":
		o.Summary.Failed++
	case "skip":
		o.Summary.Skipped++
	}
}

func requiredSecretCheck(name, target, value, hint string) DiagnosticCheck {
	if strings.TrimSpace(value) == "" {
		return failCheck(name, target, "missing", "required value is not configured", hint)
	}
	return passCheck(name, target, redactToken(value), "configured")
}

func passCheck(name, target, value, detail string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: "pass", Target: target, Value: value, Detail: detail}
}

func warnCheck(name, target, value, hint string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: "warn", Target: target, Value: value, Hint: hint}
}

func failCheck(name, target, value, detail, hint string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: "fail", Required: true, Target: target, Value: value, Detail: detail, Hint: hint}
}

func skipCheck(name, target, detail string) DiagnosticCheck {
	return DiagnosticCheck{Name: name, Status: "skip", Target: target, Detail: detail}
}

func renderDiagnosticOutput(w io.Writer, output DiagnosticOutput, format string) error {
	switch normalizeFormat(format) {
	case "markdown":
		return writeDiagnosticMarkdown(w, output)
	case "json":
		return writeJSON(w, output)
	default:
		return writeDiagnosticTable(w, output)
	}
}

func writeDiagnosticTable(w io.Writer, output DiagnosticOutput) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(tw, "LAYER\tREMOTE\tPASS\tWARN\tFAIL\tSKIP\n%s\t%t\t%d\t%d\t%d\t%d\n\n", output.Layer, output.Remote, output.Summary.Passed, output.Summary.Warned, output.Summary.Failed, output.Summary.Skipped); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(tw, "CHECK\tSTATUS\tTARGET\tVALUE\tDETAIL\tHINT"); err != nil {
		return err
	}
	for _, check := range output.Checks {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", check.Name, check.Status, check.Target, check.Value, oneLine(check.Detail), oneLine(check.Hint)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeDiagnosticMarkdown(w io.Writer, output DiagnosticOutput) error {
	if _, err := fmt.Fprintf(w, "# Feishu %s Check\n\n", titleWord(output.Layer)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Remote: `%t`\n- Passed: `%d`\n- Warned: `%d`\n- Failed: `%d`\n- Skipped: `%d`\n\n", output.Remote, output.Summary.Passed, output.Summary.Warned, output.Summary.Failed, output.Summary.Skipped); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Check | Status | Target | Value | Detail | Hint |\n| --- | --- | --- | --- | --- | --- |"); err != nil {
		return err
	}
	for _, check := range output.Checks {
		if _, err := fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n", check.Name, check.Status, check.Target, check.Value, oneLine(check.Detail), oneLine(check.Hint)); err != nil {
			return err
		}
	}
	return nil
}

func tableEnvName(table string) string {
	switch table {
	case "reports":
		return "FEISHU_REPORT_TABLE_ID"
	case "issues":
		return "FEISHU_ISSUE_TABLE_ID"
	case "prs":
		return "FEISHU_PR_TABLE_ID"
	case "contributors":
		return "FEISHU_CONTRIBUTOR_TABLE_ID"
	case "tasks":
		return "FEISHU_TASK_TABLE_ID"
	default:
		return "FEISHU_TABLE_ID"
	}
}

func fieldNames(fields []BitableField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}

func sanitizeDiagnostic(message string, values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		message = strings.ReplaceAll(message, value, redactToken(value))
	}
	return message
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
