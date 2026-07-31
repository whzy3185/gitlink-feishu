# Feishu Integration Design Plan

## Design Goal

Add a safe, testable `feishu` shortcut module to `gitlink-cli`.

The module exports GitLink repository workflow data to Feishu group cards and, later, Feishu Bitable records. It must not write back to GitLink and must default to dry-run behavior for any Feishu write path.

## Non-Goals

Do not implement:

- Feishu tasks
- Feishu approval
- Callback servers
- Button callbacks
- GitLink remote writes
- GitLink comments
- Issue closure
- Code merge actions
- Direct webhook creation on GitLink
- Automatic Base/view/Gantt initialization in Feishu

## Actual Repository Anchors

Command registration:

- `cmd/root.go`
- `shortcuts/register.go`
- `shortcuts/common/types.go`

Workflow reuse:

- `shortcuts/workflow/workflow.go`
- `shortcuts/workflow/repo_report.go`
- `shortcuts/workflow/pr_summary.go`
- `shortcuts/workflow/render.go`
- `shortcuts/workflow/testdata/`

Output behavior:

- Global output supports `json`, `yaml`, and generic `table` through `internal/output`.
- Workflow has local `json`, `table`, and `markdown` renderers.
- Feishu should use local renderers first and avoid changing global output unless repeated cross-module need appears.

Agent skills:

- Existing skills live under `skills/`.
- New skill should live under `skills/gitlink-feishu/SKILL.md`.

## Proposed Directory Layout

First implementation:

```text
shortcuts/feishu/
  feishu.go
  options.go
  redact.go
  signer.go
  card.go
  client.go
  mapper.go
  report.go
  schema.go
  bitable.go
  *_test.go
  testdata/
```

Docs and examples:

```text
docs/feishu-integration.md
docs/feishu-security.md
docs/feishu-bitable-schema.md
examples/feishu/
  bot-test.md
  notify.md
  weekly-report.md
  sync-bitable.md
skills/gitlink-feishu/SKILL.md
```

## Command Surface

Initial commands:

```bash
gitlink-cli feishu +doctor
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +sync-bitable
```

Recommended safety rule:

- All commands are dry-run unless `--dry-run=false` is explicitly provided.
- Sending to Feishu additionally requires `--to-feishu`.
- Missing credentials should produce redacted, actionable errors.

## Shared Options

```text
--owner
--repo
--since
--include issues,pulls,contributors,health
--format json|table|markdown
--lang zh-CN|en-US
--dry-run
--from-workflow-json
```

Bot options:

```text
--webhook-url
--secret
--to-feishu
```

Bitable options:

```text
--app-token
--issue-table-id
--pull-table-id
--contributor-table-id
--report-table-id
--tables issues,pulls,contributors,reports
```

Future Bitable auth options:

```text
--tenant-access-token
--app-id
--app-secret
```

Environment variables:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_ISSUE_TABLE_ID
FEISHU_PULL_TABLE_ID
FEISHU_CONTRIBUTOR_TABLE_ID
FEISHU_REPORT_TABLE_ID
FEISHU_TENANT_ACCESS_TOKEN
FEISHU_APP_ID
FEISHU_APP_SECRET
```

## Implementation Phases

### Phase 1: Baseline Scan

Create `reports/FEISHU_IMPLEMENTATION_SCAN.md` with:

- Command registration entry
- Reusable output utilities
- Reusable HTTP/client utilities
- Reusable workflow DTOs and renderers
- Current test baseline
- Feishu module landing points

No business code changes in this phase.

### Phase 2: Command Skeleton

Add `shortcuts/feishu` with all commands wired but no network writes.

Each command should:

- Provide `--help`
- Compile
- Return stable placeholder JSON/markdown/table output
- Avoid HTTP by default

Registration change:

- Add `feishu.Shortcuts()` to `shortcuts/register.go`
- Add description `"feishu": "Export GitLink workflow data to Feishu"`

### Phase 3: Options and Redaction

Implement:

```go
type Options struct { ... }
LoadOptions(ctx *common.RuntimeContext) (Options, error)
ValidateBotOptions(opts Options) error
ValidateBitableOptions(opts Options) error
MaskSecret(value string) string
MaskWebhookURL(value string) string
MaskAppToken(value string) string
MaskTableID(value string) string
MaskError(err error) string
```

Tests must verify:

- Environment fallback
- CLI flag precedence
- Include/table parsing
- Redaction
- Empty required values

### Phase 4: Bot Card, Signature, and Client

Implement:

```go
BuildBotSign(timestamp int64, secret string) (string, error)
BuildBotEnvelope(card map[string]any, secret string, now time.Time) map[string]any
BuildBotTestCard(...)
BuildProjectActivityCard(...)
BuildWeeklyReportCard(...)
SendBotMessage(ctx, webhookURL string, payload any) error
```

Tests use `httptest.Server` and fixed timestamps.

No secret, full webhook URL, app token, or tenant token may appear in errors or logs.

### Phase 5: Workflow Mapper

Reuse workflow package data instead of reimplementing analysis.

Supported inputs:

- `--from-workflow-json` for stable local input
- Later, read-only remote workflow collection via existing workflow fetch functions

Model examples:

```go
type ProjectActivity struct {
    Repository string
    Since string
    IssueSummary any
    PullSummary any
    HealthSummary any
    Recommendations []string
}

type WeeklyReport struct {
    Repository string
    Period string
    NewIssues int
    ClosedIssues int
    NewPulls int
    MergedPulls int
    ActiveContributors int
    Risks []string
    Recommendations []string
    Markdown string
}
```

### Phase 6: Bot Commands

Implement:

- `feishu +bot-test`
- `feishu +notify`
- `feishu +weekly-report`

Behavior:

- Dry-run prints card JSON or report markdown.
- Real send requires `--to-feishu --dry-run=false`.
- Missing webhook URL fails before any HTTP request.
- All request failures are redacted.

### Phase 7: Bitable Schema and Record Mapping

Implement dry-run-only first:

- `feishu +bitable-schema`
- `feishu +sync-bitable --dry-run`

This phase maps workflow data to records but does not require real Feishu auth.

Output tables:

- Issues
- Pull Requests
- Contributors
- Weekly Reports

### Phase 8: Bitable Real Writes

Only start after auth is explicit.

Required design decisions:

- Accept `FEISHU_TENANT_ACCESS_TOKEN`, or
- Obtain tenant token from `FEISHU_APP_ID` and `FEISHU_APP_SECRET`
- Define token cache policy, or avoid caching for CLI simplicity
- Choose create-only or find-by-unique-key then update
- Define pagination and batch behavior

This can be a later PR.

### Phase 9: Skill, Docs, Examples

Add:

- `skills/gitlink-feishu/SKILL.md`
- `docs/feishu-integration.md`
- `docs/feishu-security.md`
- `docs/feishu-bitable-schema.md`
- `examples/feishu/*.md`

Docs must describe engineering usage only. Avoid non-engineering packaging language.

## Test Plan

Targeted tests during development:

```bash
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
```

Final desired check:

```bash
go test ./...
```

Current known baseline issue:

- `go test ./...` fails before Feishu work in `cmd/auth` and `internal/auth` on this Windows checkout.
- Record that separately in scan/report notes so new Feishu work is not blamed for unrelated failures.

## First PR Recommendation

The first clean contribution should include:

- `feishu` command skeleton
- options/redaction
- bot signing/card/client
- `+bot-test`
- `+weekly-report` from local workflow JSON
- mock tests
- minimal docs and skill

Defer real Bitable writes to a follow-up PR.
