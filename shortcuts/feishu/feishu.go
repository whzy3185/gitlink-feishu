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
	defaultTables  = "reports,issues,prs,contributors,tasks"
	defaultLang    = "en"
)

// Shortcuts returns Feishu export shortcuts. Commands default to local preview;
// network delivery requires an explicit --send flag.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	return []*common.Shortcut{
		newBotTestShortcut(),
		newNotifyShortcut(),
		newWeeklyReportShortcut(),
		newOwnerDigestShortcut(),
		newContributorDigestShortcut(),
		newDocExportShortcut(),
		newBitableSchemaShortcut(),
		newBitableRecordsShortcut(),
		newBitableSyncShortcut(),
		newTaskPreviewShortcut(),
		newTaskCreateShortcut(),
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

func newOwnerDigestShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "owner-digest",
		Description: "Preview or send a role-aware owner digest from workflow JSON",
		Flags: append(deliveryFlags(),
			common.Flag{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			common.Flag{Name: "title", Usage: "Override card title"},
			common.Flag{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in the card"},
			common.Flag{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		),
		Run: runOwnerDigest,
	}
}

func newContributorDigestShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "contributor-digest",
		Description: "Preview or send a role-oriented contributor digest from workflow JSON",
		Flags: append(deliveryFlags(),
			common.Flag{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			common.Flag{Name: "title", Usage: "Override card title"},
			common.Flag{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in the card"},
			common.Flag{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		),
		Run: runContributorDigest,
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
			{Name: "tables", Usage: "Comma-separated tables: reports,issues,prs,contributors,tasks", Default: defaultTables},
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
			{Name: "tables", Usage: "Comma-separated tables: reports,issues,prs,contributors,tasks", Default: defaultTables},
			{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in generated records"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runBitableRecords,
	}
}

func newBitableSyncShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "bitable-sync",
		Description: "Experimental: preview or sync workflow records to Feishu Bitable",
		Flags: []common.Flag{
			{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			{Name: "tables", Usage: "Comma-separated tables: reports,issues,prs,contributors,tasks", Default: defaultTables},
			{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in generated records"},
			{Name: "app-id", Usage: "Feishu self-built app ID. Defaults to FEISHU_APP_ID"},
			{Name: "app-secret", Usage: "Feishu self-built app secret. Defaults to FEISHU_APP_SECRET"},
			{Name: "base-app-token", Usage: "Feishu Base app token. Defaults to FEISHU_BASE_APP_TOKEN"},
			{Name: "report-table-id", Usage: "Reports table ID. Defaults to FEISHU_REPORT_TABLE_ID"},
			{Name: "issue-table-id", Usage: "Issues table ID. Defaults to FEISHU_ISSUE_TABLE_ID"},
			{Name: "pr-table-id", Usage: "Pull requests table ID. Defaults to FEISHU_PR_TABLE_ID"},
			{Name: "contributor-table-id", Usage: "Contributors table ID. Defaults to FEISHU_CONTRIBUTOR_TABLE_ID"},
			{Name: "task-table-id", Usage: "Tasks table ID. Defaults to FEISHU_TASK_TABLE_ID"},
			{Name: "send", Usage: "Write to Feishu Bitable. Without --send, preview locally", Bool: true, Default: "false"},
			{Name: "dry-run", Usage: "Force local preview. Cannot be combined with --send", Bool: true, Default: "false"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runBitableSync,
	}
}

func newTaskPreviewShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "task-preview",
		Description: "Preview Feishu task candidates from workflow JSON",
		Flags: []common.Flag{
			{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in generated tasks"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runTaskPreview,
	}
}

func newTaskCreateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "task-create",
		Description: "Experimental: preview or create Feishu tasks from workflow JSON",
		Flags: []common.Flag{
			{Name: "from-workflow-json", Usage: "Read workflow repo report JSON from a file", Required: true},
			{Name: "doc-url", Usage: "Feishu DocX or Wiki URL to include in generated tasks"},
			{Name: "app-id", Usage: "Feishu self-built app ID. Defaults to FEISHU_APP_ID"},
			{Name: "app-secret", Usage: "Feishu self-built app secret. Defaults to FEISHU_APP_SECRET"},
			{Name: "task-project-id", Usage: "Feishu task project ID. Defaults to FEISHU_TASK_PROJECT_ID"},
			{Name: "task-section-id", Usage: "Feishu task section ID. Defaults to FEISHU_TASK_SECTION_ID"},
			{Name: "send", Usage: "Create Feishu tasks. Without --send, preview locally", Bool: true, Default: "false"},
			{Name: "dry-run", Usage: "Force local preview. Cannot be combined with --send", Bool: true, Default: "false"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: defaultLang},
		},
		Run: runTaskCreate,
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

func runOwnerDigest(ctx *common.RuntimeContext) error {
	opts, err := deliveryOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	digest := BuildOwnerDigest(report, ctx.Arg("doc-url"))
	if opts.Send {
		title := firstNonEmpty(ctx.Arg("title"), "GitLink owner digest: "+report.Repository)
		return deliverOrPreview(ctx, opts, NewInteractivePayload(BuildOwnerDigestCard(digest, title, normalizeLang(ctx.Arg("lang")))), "")
	}
	return renderDigest(os.Stdout, digest, formatOrDefault(ctx, "markdown"))
}

func runContributorDigest(ctx *common.RuntimeContext) error {
	opts, err := deliveryOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	digest := BuildContributorDigest(report, ctx.Arg("doc-url"))
	if opts.Send {
		title := firstNonEmpty(ctx.Arg("title"), "GitLink contributor digest: "+report.Repository)
		return deliverOrPreview(ctx, opts, NewInteractivePayload(BuildContributorDigestCard(digest, title, normalizeLang(ctx.Arg("lang")))), "")
	}
	return renderDigest(os.Stdout, digest, formatOrDefault(ctx, "markdown"))
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
	records := BuildBitableRecords(report, parseList(firstNonEmpty(ctx.Arg("tables"), defaultTables)), ctx.Arg("doc-url"))
	return renderBitableRecords(os.Stdout, records, formatOrDefault(ctx, "json"))
}

func runBitableSync(ctx *common.RuntimeContext) error {
	opts, err := bitableSyncOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	records := BuildBitableRecords(report, opts.Tables, ctx.Arg("doc-url"))
	return syncBitableOrPreview(ctx, opts, records)
}

func runTaskPreview(ctx *common.RuntimeContext) error {
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	tasks := BuildTaskCandidates(report, ctx.Arg("doc-url"))
	return renderTaskOutput(os.Stdout, TaskOutput{Mode: "preview", DryRun: true, Tasks: tasks}, formatOrDefault(ctx, "markdown"))
}

func runTaskCreate(ctx *common.RuntimeContext) error {
	opts, err := taskCreateOptionsFromContext(ctx)
	if err != nil {
		return err
	}
	report, err := readWorkflowReport(ctx.Arg("from-workflow-json"), normalizeLang(ctx.Arg("lang")))
	if err != nil {
		return err
	}
	tasks := BuildTaskCandidates(report, ctx.Arg("doc-url"))
	return createTasksOrPreview(ctx, opts, tasks)
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
