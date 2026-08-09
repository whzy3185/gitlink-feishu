package feishu

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type DocExportOptions struct {
	AppID         string `json:"-"`
	AppSecret     string `json:"-"`
	FolderToken   string `json:"folder_token,omitempty"`
	DocumentID    string `json:"document_id,omitempty"`
	WikiURL       string `json:"wiki_url,omitempty"`
	WikiNodeToken string `json:"wiki_node_token,omitempty"`
	Title         string `json:"title"`
	Send          bool   `json:"send"`
	DryRun        bool   `json:"dry_run"`
}

type DocExportOutput struct {
	Mode          string           `json:"mode"`
	Send          bool             `json:"send"`
	DryRun        bool             `json:"dry_run"`
	TargetType    string           `json:"target_type"`
	Operation     string           `json:"operation"`
	Title         string           `json:"title"`
	DocumentID    string           `json:"document_id,omitempty"`
	DocumentURL   string           `json:"document_url,omitempty"`
	WikiNodeToken string           `json:"wiki_node_token,omitempty"`
	WikiNode      *WikiNodeSummary `json:"wiki_node,omitempty"`
	BlockCount    int              `json:"block_count"`
	TokenExpire   int              `json:"token_expire,omitempty"`
	RevisionID    int              `json:"revision_id,omitempty"`
	Preview       string           `json:"preview,omitempty"`
	Diagnostics   []string         `json:"diagnostics,omitempty"`
}

type WikiNodeSummary struct {
	NodeType string `json:"node_type,omitempty"`
	ObjType  string `json:"obj_type,omitempty"`
	Title    string `json:"title,omitempty"`
}

type DocBlock map[string]interface{}

func docExportOptionsFromContext(ctx *common.RuntimeContext) (DocExportOptions, error) {
	opts := DocExportOptions{
		AppID:         firstNonEmpty(ctx.Arg("app-id"), os.Getenv("FEISHU_APP_ID")),
		AppSecret:     firstNonEmpty(ctx.Arg("app-secret"), os.Getenv("FEISHU_APP_SECRET")),
		FolderToken:   firstNonEmpty(ctx.Arg("folder-token"), os.Getenv("FEISHU_FOLDER_TOKEN"), os.Getenv("FEISHU_DOC_FOLDER_TOKEN")),
		DocumentID:    firstNonEmpty(ctx.Arg("document-id"), os.Getenv("FEISHU_DOCUMENT_ID")),
		WikiURL:       firstNonEmpty(ctx.Arg("wiki-url"), os.Getenv("FEISHU_WIKI_URL")),
		WikiNodeToken: firstNonEmpty(ctx.Arg("wiki-node-token"), os.Getenv("FEISHU_WIKI_NODE_TOKEN")),
		Title:         strings.TrimSpace(ctx.Arg("title")),
		Send:          parseBool(ctx.Arg("send")),
		DryRun:        parseBool(ctx.Arg("dry-run")),
	}
	if opts.WikiNodeToken == "" && opts.WikiURL != "" {
		opts.WikiNodeToken = wikiNodeTokenFromURL(opts.WikiURL)
	}
	if opts.DocumentID == "" && opts.WikiURL != "" {
		opts.DocumentID = docxTokenFromURL(opts.WikiURL)
	}
	if opts.Send && opts.DryRun {
		return DocExportOptions{}, fmt.Errorf("--send and --dry-run cannot be used together")
	}
	if opts.Send {
		if strings.TrimSpace(opts.AppID) == "" {
			return DocExportOptions{}, fmt.Errorf("--send requires --app-id or FEISHU_APP_ID")
		}
		if strings.TrimSpace(opts.AppSecret) == "" {
			return DocExportOptions{}, fmt.Errorf("--send requires --app-secret or FEISHU_APP_SECRET")
		}
		if opts.FolderToken == "" && opts.DocumentID == "" && opts.WikiNodeToken == "" {
			return DocExportOptions{}, fmt.Errorf("--send requires --folder-token, --document-id, --wiki-url, or --wiki-node-token")
		}
	}
	return opts, nil
}

func exportDocOrPreview(ctx *common.RuntimeContext, opts DocExportOptions, report workflow.RepoReportResult, lang string) error {
	title := firstNonEmpty(opts.Title, "GitLink workflow report: "+report.Repository)
	markdown, err := workflow.RenderRepoReport(report, "markdown", lang)
	if err != nil {
		return err
	}
	blocks := BuildDocBlocks(report, lang)
	output := DocExportOutput{
		Mode:        "preview",
		Send:        opts.Send,
		DryRun:      !opts.Send,
		TargetType:  docTargetType(opts),
		Operation:   docOperation(opts),
		Title:       title,
		DocumentID:  redactToken(opts.DocumentID),
		DocumentURL: redactResourceURL(firstNonEmpty(opts.WikiURL)),
		BlockCount:  len(blocks),
		Preview:     markdown,
	}
	if opts.WikiNodeToken != "" {
		output.WikiNodeToken = redactToken(opts.WikiNodeToken)
	}
	if !opts.Send {
		return renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "markdown"))
	}

	client := NewOpenAPIClient(http.DefaultClient)
	token, err := client.TenantAccessToken(context.Background(), opts.AppID, opts.AppSecret)
	if err != nil {
		output.Diagnostics = append(output.Diagnostics, diagnoseOpenAPIError(err, "docx", "tenant_access_token"))
		_ = renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
		return err
	}
	output.TokenExpire = token.Expire

	documentID := opts.DocumentID
	if opts.WikiNodeToken != "" {
		node, err := client.GetWikiNode(context.Background(), token.Value, opts.WikiNodeToken)
		if err != nil {
			output.Diagnostics = append(output.Diagnostics, diagnoseOpenAPIError(err, "docx", "wiki node"))
			_ = renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
			return err
		}
		output.TargetType = "wiki"
		if node.ObjType != "" && node.ObjType != "docx" {
			output.Diagnostics = append(output.Diagnostics, "wiki node object type is not supported; expected docx")
			_ = renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
			return fmt.Errorf("Feishu wiki node object type %q is not supported; expected docx", node.ObjType)
		}
		documentID = node.ObjToken
		output.DocumentID = redactToken(documentID)
		output.WikiNode = &WikiNodeSummary{
			NodeType: node.NodeType,
			ObjType:  node.ObjType,
			Title:    node.Title,
		}
		if output.DocumentURL == "" {
			output.DocumentURL = redactResourceURL(node.URL)
		}
	}
	if documentID == "" {
		created, err := client.CreateDocument(context.Background(), token.Value, opts.FolderToken, title)
		if err != nil {
			output.Diagnostics = append(output.Diagnostics, diagnoseOpenAPIError(err, "docx", "folder"))
			_ = renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
			return err
		}
		documentID = created.DocumentID
		output.DocumentID = redactToken(created.DocumentID)
		output.DocumentURL = redactResourceURL(created.URL)
		output.RevisionID = created.RevisionID
		output.Operation = "create"
	}
	createdBlocks, err := client.CreateBlocks(context.Background(), token.Value, documentID, documentID, blocks)
	if err != nil {
		output.Diagnostics = append(output.Diagnostics, diagnoseOpenAPIError(err, "docx", output.TargetType))
		output.Diagnostics = append(output.Diagnostics, "required permission: app can edit the target DocX/Wiki page, or create documents in the target folder")
		_ = renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
		return fmt.Errorf("%w\nhint: grant the Feishu self-built app edit access to the target DocX/Wiki page, or export to a folder where the app has document creation permission", err)
	}
	if createdBlocks.RevisionID != 0 {
		output.RevisionID = createdBlocks.RevisionID
	}
	output.Mode = "sent"
	output.DryRun = false
	output.Preview = ""
	return renderDocExportOutput(os.Stdout, output, formatOrDefault(ctx, "json"))
}

func BuildDocBlocks(report workflow.RepoReportResult, lang string) []DocBlock {
	healthScore := "N/A"
	healthRisk := "N/A"
	if report.Health != nil {
		healthScore = fmt.Sprintf("%d", report.Health.HealthScore)
		healthRisk = report.Health.RiskLevel
	}
	blocks := []DocBlock{
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_title"), report.Repository)),
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_report_score"), report.ReportScore)),
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_risk"), firstNonEmpty(report.RiskLevel, "unknown"))),
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_health"), healthScore, healthRisk)),
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_issues"), report.IssueSummary.Total, report.IssueSummary.HighRisk, report.IssueSummary.MissingInfo)),
		textBlock(fmt.Sprintf(feishuLabel(lang, "doc_prs"), report.PRSummary.Total, report.PRSummary.HighRisk)),
		textBlock(feishuLabel(lang, "analysis_scope")),
	}
	if report.PRLifecycle != nil {
		blocks = append(blocks, textBlock(fmt.Sprintf(
			"%s=%d; %s=%d; %s=%d",
			feishuLabel(lang, "open_prs"), report.PRLifecycle.Open,
			feishuLabel(lang, "merged_prs"), report.PRLifecycle.Merged,
			feishuLabel(lang, "closed_prs"), report.PRLifecycle.ClosedOrRejected,
		)))
	}
	if report.PRReviewAudit != nil {
		blocks = append(blocks, textBlock(fmt.Sprintf(
			"%s=%d; %s=%d; %s=%d; %s=%d; %s=%d",
			feishuLabel(lang, "review_audited"), report.PRReviewAudit.Audited,
			feishuLabel(lang, "reviewed_prs"), report.PRReviewAudit.Reviewed,
			feishuLabel(lang, "unreviewed_prs"), report.PRReviewAudit.Unreviewed,
			feishuLabel(lang, "needs_re_review"), report.PRReviewAudit.NeedsReReview,
			feishuLabel(lang, "formal_reviews"), report.PRReviewAudit.FormalReviews,
		)))
		blocks = append(blocks, textBlock(feishuLabel(lang, "review_actor_attribution")+":\n"+joinLines(reviewAuditActorLines(report.PRReviewAudit, lang), 6)))
	}
	if len(report.PRSummary.ReviewFocus) > 0 {
		blocks = append(blocks, textBlock(feishuLabel(lang, "doc_review_focus")+":\n"+joinLines(localizeFeishuLines(report.PRSummary.ReviewFocus, lang), 6)))
	}
	if len(report.Recommendations) > 0 {
		blocks = append(blocks, textBlock(feishuLabel(lang, "doc_recommendations")+":\n"+joinLines(localizeFeishuLines(report.Recommendations, lang), 8)))
	}
	if len(report.Reasoning) > 0 {
		blocks = append(blocks, textBlock(feishuLabel(lang, "doc_reasoning")+":\n"+joinLines(localizeFeishuLines(report.Reasoning, lang), 8)))
	}
	blocks = append(blocks, textBlock(fmt.Sprintf(feishuLabel(lang, "doc_source"), firstNonEmpty(report.Source, "workflow-json"))))
	return blocks
}

func textBlock(content string) DocBlock {
	return DocBlock{
		"block_type": 2,
		"text": map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"text_run": map[string]interface{}{
						"content": content,
					},
				},
			},
		},
	}
}

func renderDocExportOutput(w io.Writer, output DocExportOutput, format string) error {
	switch normalizeFormat(format) {
	case "json":
		return writeJSON(w, output)
	case "table":
		return writeDocExportTable(w, output)
	default:
		return writeDocExportMarkdown(w, output)
	}
}

func writeDocExportMarkdown(w io.Writer, output DocExportOutput) error {
	if _, err := fmt.Fprintf(w, "# Feishu Doc Export\n\n"); err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("- Mode: `%s`", output.Mode),
		fmt.Sprintf("- Send: `%t`", output.Send),
		fmt.Sprintf("- Dry run: `%t`", output.DryRun),
		fmt.Sprintf("- Target: `%s`", output.TargetType),
		fmt.Sprintf("- Operation: `%s`", output.Operation),
		fmt.Sprintf("- Title: `%s`", output.Title),
		fmt.Sprintf("- Blocks: `%d`", output.BlockCount),
	}
	if output.DocumentID != "" {
		lines = append(lines, fmt.Sprintf("- Document ID: `%s`", output.DocumentID))
	}
	if output.DocumentURL != "" {
		lines = append(lines, fmt.Sprintf("- Document URL: %s", output.DocumentURL))
	}
	if _, err := fmt.Fprintln(w, strings.Join(lines, "\n")); err != nil {
		return err
	}
	for _, diagnostic := range output.Diagnostics {
		if _, err := fmt.Fprintf(w, "- Diagnostic: %s\n", diagnostic); err != nil {
			return err
		}
	}
	if output.Preview != "" {
		if _, err := fmt.Fprint(w, "\n## Preview\n\n"); err != nil {
			return err
		}
		_, err := fmt.Fprint(w, output.Preview)
		return err
	}
	return nil
}

func writeDocExportTable(w io.Writer, output DocExportOutput) error {
	_, err := fmt.Fprintf(w, "MODE\tSEND\tDRY_RUN\tTARGET\tOPERATION\tBLOCKS\tDOCUMENT\n%s\t%t\t%t\t%s\t%s\t%d\t%s\n",
		output.Mode,
		output.Send,
		output.DryRun,
		output.TargetType,
		output.Operation,
		output.BlockCount,
		firstNonEmpty(output.DocumentURL, output.DocumentID),
	)
	return err
}

func docTargetType(opts DocExportOptions) string {
	switch {
	case opts.WikiNodeToken != "":
		return "wiki"
	case opts.DocumentID != "":
		return "docx"
	case opts.FolderToken != "":
		return "folder"
	default:
		return "preview"
	}
}

func docOperation(opts DocExportOptions) string {
	if opts.FolderToken != "" && opts.DocumentID == "" && opts.WikiNodeToken == "" {
		return "create"
	}
	if opts.DocumentID != "" || opts.WikiNodeToken != "" {
		return "append"
	}
	return "preview"
}

func wikiNodeTokenFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, part := range parts {
		if part == "wiki" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func docxTokenFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, part := range parts {
		if part == "docx" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func joinLines(values []string, limit int) string {
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	lines := make([]string, 0, limit)
	for _, value := range values[:limit] {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		lines = append(lines, "- "+value)
	}
	return strings.Join(lines, "\n")
}
