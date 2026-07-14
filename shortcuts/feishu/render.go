package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type DeliveryOutput struct {
	Mode       string           `json:"mode"`
	Send       bool             `json:"send"`
	DryRun     bool             `json:"dry_run"`
	WebhookURL string           `json:"webhook_url,omitempty"`
	Payload    WebhookPayload   `json:"payload"`
	Response   *WebhookResponse `json:"response,omitempty"`
}

func deliverOrPreview(ctx *common.RuntimeContext, opts DeliveryOptions, payload WebhookPayload, markdown string) error {
	output := DeliveryOutput{
		Mode:       "preview",
		Send:       opts.Send,
		DryRun:     !opts.Send,
		WebhookURL: redactWebhookURL(opts.WebhookURL),
		Payload:    payload,
	}
	if opts.Send {
		client := WebhookClient{
			URL:    opts.WebhookURL,
			Secret: opts.Secret,
		}
		resp, err := client.Send(context.Background(), payload)
		output.Mode = "sent"
		output.DryRun = false
		output.Response = resp
		if err != nil {
			_ = renderDeliveryOutput(os.Stdout, output, ctx.Format, markdown)
			return err
		}
	}
	return renderDeliveryOutput(os.Stdout, output, ctx.Format, markdown)
}

func renderDeliveryOutput(w io.Writer, output DeliveryOutput, format string, markdown string) error {
	switch normalizeFormat(format) {
	case "markdown":
		if markdown != "" {
			_, err := fmt.Fprint(w, markdown)
			return err
		}
		return writeDeliveryMarkdown(w, output)
	case "table":
		return writeDeliveryTable(w, output)
	default:
		return writeJSON(w, output)
	}
}

func writeDeliveryMarkdown(w io.Writer, output DeliveryOutput) error {
	lines := []string{
		"# Feishu Delivery Preview",
		"",
		fmt.Sprintf("- Mode: `%s`", output.Mode),
		fmt.Sprintf("- Send: `%t`", output.Send),
		fmt.Sprintf("- Dry run: `%t`", output.DryRun),
	}
	if output.WebhookURL != "" {
		lines = append(lines, fmt.Sprintf("- Webhook: `%s`", output.WebhookURL))
	}
	if output.Response != nil {
		lines = append(lines, fmt.Sprintf("- HTTP status: `%d`", output.Response.StatusCode))
		lines = append(lines, fmt.Sprintf("- Feishu code: `%d`", output.Response.Code))
		if output.Response.Message != "" {
			lines = append(lines, fmt.Sprintf("- Message: `%s`", output.Response.Message))
		}
	}
	_, err := fmt.Fprintln(w, strings.Join(lines, "\n"))
	return err
}

func writeDeliveryTable(w io.Writer, output DeliveryOutput) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "MODE\tSEND\tDRY_RUN\tWEBHOOK\tHTTP_STATUS\tFEISHU_CODE"); err != nil {
		return err
	}
	status := ""
	code := ""
	if output.Response != nil {
		status = fmt.Sprintf("%d", output.Response.StatusCode)
		code = fmt.Sprintf("%d", output.Response.Code)
	}
	if _, err := fmt.Fprintf(tw, "%s\t%t\t%t\t%s\t%s\t%s\n", output.Mode, output.Send, output.DryRun, output.WebhookURL, status, code); err != nil {
		return err
	}
	return tw.Flush()
}

func writeJSON(w io.Writer, data interface{}) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(encoded))
	return err
}

func normalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		return "json"
	}
	return format
}
