package feishu

import (
	"fmt"
	"os"
	"strings"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const (
	defaultInclude = "issues,prs,contributors,health"
	defaultTables  = "issues,prs,contributors,reports"
	defaultLang    = "en"
)

// Shortcuts returns Feishu export shortcuts. Commands default to local preview;
// network delivery requires an explicit --send flag.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	return []*common.Shortcut{
		newBotTestShortcut(),
		newNotifyShortcut(),
		newWeeklyReportShortcut(),
		newDocExportShortcut(),
		newBitableSchemaShortcut(),
		newBitableRecordsShortcut(),
	}
}

func newBotTestShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "bot-test",
		Description: "Preview or send a Feishu custom bot test card",
		Flags: append(deliveryFlags(),
			common.Flag{Name: "title", Usage: "Card title", Default: "GitLink Feishu integration test"},
			common.Flag{Name: "message", Usage: "Card message", Default: "gitlink-cli can build and send Feishu custom bot cards."},
			common.Flag{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		),
		Run: runBotTest,
	}
}

func newNotifyShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "notify",
		Description: "Preview or send a Feishu card from workflow JSON",
		Flags: append(deliveryFlags(),
			common.Flag{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			common.Flag{Name: "include", Usage: "Comma-separated sections: issues,prs,contributors,health", Default: defaultInclude},
			common.Flag{Name: "title", Usage: "Override card title"},
			common.Flag{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in the card"},
			common.Flag{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		),
		Run: runNotify,
	}
}

func newWeeklyReportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "weekly-report",
		Description: "Render a weekly report from workflow JSON and optionally send it to Feishu",
		Flags: append(deliveryFlags(),
			common.Flag{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			common.Flag{Name: "include", Usage: "Comma-separated sections: issues,prs,contributors,health", Default: defaultInclude},
			common.Flag{Name: "title", Usage: "Override card title"},
			common.Flag{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in the card"},
			common.Flag{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		),
		Run: runWeeklyReport,
	}
}

func newDocExportShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "doc-export",
		Description: "Experimental: preview or export a workflow report to Feishu DocX or Wiki",
		Flags: []common.Flag{
			{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			{Name: "title", Usage: "Document title"},
			{Name: "folder-token", Usage: "Feishu folder token for creating a new DocX. Defaults to FEISHU_DOC_FOLDER_TOKEN"},
			{Name: "document-id", Usage: "Existing Feishu DocX document ID. Defaults to FEISHU_DOCUMENT_ID"},
			{Name: "wiki-url", Usage: "Existing Feishu Wiki URL. Defaults to FEISHU_WIKI_URL"},
			{Name: "wiki-node-token", Usage: "Existing Feishu Wiki node token. Defaults to FEISHU_WIKI_NODE_TOKEN"},
			{Name: "app-id", Usage: "Feishu self-built app ID. Defaults to FEISHU_APP_ID"},
			{Name: "app-secret", Usage: "Feishu self-built app secret. Defaults to FEISHU_APP_SECRET"},
			{Name: "send", Usage: "Create or update a Feishu document. Without --send, preview locally", Bool: true, Default: "false"},
			{Name: "dry-run", Usage: "Force local preview. Cannot be combined with --send", Bool: true, Default: "false"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runDocExport,
	}
}

func newBitableSchemaShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "bitable-schema",
		Description: "Generate a dry-run Feishu Bitable schema",
		Flags: []common.Flag{
			{Name: "tables", Usage: "Comma-separated tables: issues,prs,contributors,reports", Default: defaultTables},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runBitableSchema,
	}
}

func newBitableRecordsShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "bitable-records",
		Description: "Generate dry-run Feishu Bitable-ready records from workflow JSON",
		Flags: []common.Flag{
			{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			{Name: "tables", Usage: "Comma-separated tables: issues,prs,contributors,reports", Default: defaultTables},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runBitableRecords,
	}
}

func deliveryFlags() []common.Flag {
	return []common.Flag{
		{Name: "webhook-url", Usage: "Feishu custom bot webhook URL. Defaults to FEISHU_WEBHOOK_URL"},
		{Name: "secret", Usage: "Feishu custom bot signing secret. Defaults to FEISHU_WEBHOOK_SECRET"},
		{Name: "send", Usage: "Send to Feishu. Without --send, commands only preview locally", Bool: true, Default: "false"},
		{Name: "dry-run", Usage: "Force local preview. Cannot be combined with --send", Bool: true, Default: "false"},
	}
}

func runBotTest(ctx *common.RuntimeContext) error {
	opts, err := deliveryOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	card := BuildBotTestCard(ctx.Arg("title"), ctx.Arg("message"), normalizeLang(ctx.Arg("lang")))
	payload := NewInteractivePayload(card)
	return deliverOrPreview(ctx, opts, payload, "")
}

func runNotify(ctx *common.RuntimeContext) error {
	opts, err := deliveryOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	include := parseList(firstNonEmpty(ctx.Arg("include"), defaultInclude))
	card := BuildWorkflowCard(report, include, firstNonEmpty(ctx.Arg("title"), ""), normalizeLang(ctx.Arg("lang")), ctx.Arg("doc-url"))
	payload := NewInteractivePayload(card)
	return deliverOrPreview(ctx, opts, payload, "")
}

func runWeeklyReport(ctx *common.RuntimeContext) error {
	opts, err := deliveryOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	if opts.Send {
		include := parseList(firstNonEmpty(ctx.Arg("include"), defaultInclude))
		card := BuildWorkflowCard(report, include, firstNonEmpty(ctx.Arg("title"), "GitLink weekly workflow report"), normalizeLang(ctx.Arg("lang")), ctx.Arg("doc-url"))
		return deliverOrPreview(ctx, opts, NewInteractivePayload(card), "")
	}
	rendered, err := workflow.RenderRepoReport(report, formatOrDefault(ctx, "markdown"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func runDocExport(ctx *common.RuntimeContext) error {
	opts, err := docExportOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	return exportDocOrPreview(ctx, opts, report, normalizeLang(ctx.Arg("lang")))
}

func runBitableSchema(ctx *common.RuntimeContext) error {
	schema := BuildBitableSchema(parseList(firstNonEmpty(ctx.Arg("tables"), defaultTables)))
	return renderBitableSchema(os.Stdout, schema, formatOrDefault(ctx, "markdown"))
}

func runBitableRecords(ctx *common.RuntimeContext) error {
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	records := BuildBitableRecords(report, parseList(firstNonEmpty(ctx.Arg("tables"), defaultTables)))
	return renderBitableRecords(os.Stdout, records, formatOrDefault(ctx, "json"))
}

func normalizeLang(lang string) string {
	switch strings.TrimSpace(lang) {
	case "zh-CN":
		return "zh-CN"
	default:
		return defaultLang
	}
}

func formatOrDefault(ctx *common.RuntimeContext, defaultFormat string) string {
	if strings.TrimSpace(cmdutil.Format) == "" {
		return defaultFormat
	}
	return ctx.Format
}
