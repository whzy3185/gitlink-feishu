# Codex Clean Task Chain: GitLink CLI Feishu Export

## Goal

Add a safe `feishu` shortcut module to `gitlink-cli`.

This task only implements:

```text
GitLink workflow JSON
  -> Feishu card preview
  -> Feishu bot send with explicit --send
  -> weekly report
  -> Bitable schema
  -> Bitable-ready records
  -> Agent Skill
```

## Boundaries

Do not implement:

```text
Feishu tasks
Feishu approval
callback server
button callbacks
GitLink remote writes
GitLink comments
Issue closure
code merge actions
direct GitLink webhook creation
real Bitable OpenAPI writes
Feishu Base creation
Feishu table creation
Feishu view creation
open_id person mapping
```

## Hard Rules

```text
1. Target repository: gitlink-cli.
2. Language: Go.
3. Do not add Ruby, Rails, Python, or Node services.
4. First implementation consumes local workflow JSON only.
5. Default behavior is local preview.
6. Real Feishu bot sending requires explicit --send.
7. If --send and --dry-run are both set, return an error.
8. Tests must use local fixtures or mock HTTP servers.
9. Tests must not depend on a real Feishu tenant, bot, token, or webhook.
10. Logs and errors must not print full webhook URLs or secrets.
11. Do not add non-engineering packaging language to code, docs, comments, or reports.
```

## Phase 0: Repository Scan

### Objective

Scan the current codebase and record implementation anchors.

### Read

```text
cmd/root.go
shortcuts/register.go
shortcuts/common/runner.go
shortcuts/common/types.go
shortcuts/workflow/workflow.go
shortcuts/workflow/repo_report.go
shortcuts/workflow/pr_summary.go
shortcuts/workflow/render.go
shortcuts/workflow/testdata/
internal/output/
skills/
docs/
examples/
go.mod
Makefile
.github/
```

### Create

```text
reports/FEISHU_IMPLEMENTATION_SCAN.md
```

### Required Content

```text
Repository anchors
Reusable workflow data and renderers
Reusable output behavior
Shortcut registration pattern
Test fixtures
Current test baseline
Implementation landing points
```

### Verification

```bash
go test ./shortcuts/workflow
go test ./shortcuts
```

Use this when the default Go proxy is unavailable:

```powershell
$env:GOPROXY='https://goproxy.cn,direct'
```

## Phase 1: Feishu Command Skeleton

### Objective

Add a `feishu` shortcut group with five commands. All commands must compile and support `--help`. No command sends network traffic by default.

### Add

```text
shortcuts/feishu/
  feishu.go
  options.go
  redact.go
  output.go
```

### Register

Update:

```text
shortcuts/register.go
```

Add:

```text
feishu -> feishu.Shortcuts()
```

Description:

```text
Export GitLink workflow data to Feishu cards and Bitable-ready records
```

### Commands

```bash
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
```

Do not add:

```text
+sync-bitable
+approval-create
+sync-tasks
+serve
+webhook-create
```

### Shared Flags

```text
--owner
--repo
--since
--include issues,prs,contributors,health
--format json|table|markdown
--lang zh-CN|en-US
--dry-run
--from-workflow-json
```

### Bot Flags

```text
--webhook-url
--secret
--send
```

### Bitable Preview Flags

```text
--tables issues,prs,contributors,reports
```

### Environment Variables

Read only:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET
```

Do not read in this task:

```text
FEISHU_TENANT_ACCESS_TOKEN
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_ISSUE_TABLE_ID
FEISHU_PR_TABLE_ID
FEISHU_CONTRIBUTOR_TABLE_ID
FEISHU_REPORT_TABLE_ID
```

### Verification

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts
```

Check help:

```bash
go run . feishu --help
go run . feishu +bot-test --help
go run . feishu +notify --help
go run . feishu +weekly-report --help
go run . feishu +bitable-schema --help
go run . feishu +bitable-records --help
```

## Phase 2: Options and Redaction

### Objective

Implement safe option loading, validation, and redaction.

### Files

```text
shortcuts/feishu/options.go
shortcuts/feishu/redact.go
shortcuts/feishu/options_test.go
shortcuts/feishu/redact_test.go
```

### Options

```go
type Options struct {
    Owner string
    Repo string
    Since string
    Include []string
    Format string
    Lang string
    DryRun bool
    Send bool
    FromWorkflowJSON string

    WebhookURL string
    WebhookSecret string

    Tables []string
}
```

### Functions

```go
func LoadOptions(ctx *common.RuntimeContext) (Options, error)
func ValidateCommonOptions(opts Options) error
func ValidateBotOptions(opts Options) error
func ValidateWorkflowJSONOptions(opts Options) error
func ValidateBitableRecordOptions(opts Options) error
func NormalizeInclude(value string) ([]string, error)
func NormalizeTables(value string) ([]string, error)
func ValidateSendMode(opts Options) error
```

### Send Mode

```text
Default: preview only.
--send: send Feishu bot message.
--send + --dry-run: error.
--send without webhook URL: error.
No --send: no HTTP request.
```

### Redaction

```go
func MaskSecret(value string) string
func MaskWebhookURL(value string) string
func MaskToken(value string) string
func MaskError(err error) string
```

Rules:

```text
secret -> ***
token -> ***
webhook URL -> scheme + host + final 4 path characters
empty string -> ""
short sensitive string -> ***
```

### Verification

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
```

## Phase 3: Bot Signature

### Objective

Implement Feishu custom bot signing.

### Files

```text
shortcuts/feishu/signer.go
shortcuts/feishu/signer_test.go
```

### Functions

```go
func BuildBotSign(timestamp int64, secret string) (string, error)
func BuildBotEnvelope(card map[string]any, secret string, now time.Time) (map[string]any, error)
```

### Behavior

```text
Empty secret: no timestamp/sign.
Non-empty secret: add timestamp/sign.
Timestamp: Unix seconds.
Errors must not include secret.
Tests use fixed timestamps.
```

## Phase 4: Card Builders

### Objective

Build Feishu interactive card JSON without network calls.

### Files

```text
shortcuts/feishu/model.go
shortcuts/feishu/card.go
shortcuts/feishu/card_test.go
```

### Models

```go
type ProjectActivity struct {
    Repository string
    Period string
    IssueSummary SummaryBlock
    PullRequestSummary SummaryBlock
    ContributorSummary SummaryBlock
    HealthSummary SummaryBlock
    Risks []string
    Recommendations []string
}

type SummaryBlock struct {
    Title string
    Count int
    Items []string
}

type WeeklyReport struct {
    Repository string
    Period string
    NewIssues int
    ClosedIssues int
    NewPullRequests int
    MergedPullRequests int
    ActiveContributors int
    Risks []string
    Recommendations []string
    Markdown string
}
```

### Functions

```go
func BuildBotTestCard(lang string) map[string]any
func BuildProjectActivityCard(activity ProjectActivity, lang string) map[string]any
func BuildWeeklyReportCard(report WeeklyReport, lang string) map[string]any
```

### Requirements

```text
Use interactive card.
Support zh-CN and en-US text.
Handle empty data.
Do not include secrets or webhook URLs.
```

## Phase 5: Bot Webhook Client

### Objective

Send Feishu bot payloads through an injectable HTTP client.

### Files

```text
shortcuts/feishu/client.go
shortcuts/feishu/client_test.go
```

### API

```go
type Client struct {
    HTTPClient *http.Client
}

func NewClient(httpClient *http.Client) *Client
func (c *Client) SendBotMessage(ctx context.Context, webhookURL string, payload any) error
```

### Behavior

```text
POST JSON.
2xx means success.
Non-2xx returns redacted error.
Limit response body read size.
Network errors are redacted.
Never print full webhook URL.
```

### Tests

Use `httptest.Server` for:

```text
200
400
429
500
network error
redaction
```

Do not implement Bitable OpenAPI client in this phase.

## Phase 6: Bot Test Command

### Objective

Preview or explicitly send a test card.

### Commands

Preview:

```bash
go run . feishu +bot-test --format json
```

Send:

```bash
go run . feishu +bot-test --webhook-url "$FEISHU_WEBHOOK_URL" --secret "$FEISHU_WEBHOOK_SECRET" --send
```

### Behavior

```text
Default preview.
Preview prints JSON.
--send sends through mockable client.
--send requires webhook URL.
Output shows only redacted target.
No GitLink data read.
No GitLink write.
```

## Phase 7: Workflow JSON Mapper

### Objective

Read local workflow JSON and map it into Feishu models.

### Files

```text
shortcuts/feishu/mapper.go
shortcuts/feishu/mapper_test.go
shortcuts/feishu/testdata/
  repo_report.json
  pr_summary.json
  health.json
```

### Functions

```go
func LoadWorkflowJSON(path string) (map[string]any, error)
func MapWorkflowToProjectActivity(data map[string]any, opts Options) (ProjectActivity, error)
func MapWorkflowToWeeklyReport(data map[string]any, opts Options) (WeeklyReport, error)
```

### Rules

```text
Support workflow +repo-report JSON first.
Missing fields must not panic.
Unknown fields are ignored.
Do not duplicate workflow analysis logic.
Do not call GitLink remote APIs.
Do not change existing workflow behavior.
```

## Phase 8: Notify Command

### Objective

Create a project activity card from workflow JSON.

### Commands

Preview:

```bash
go run . feishu +notify --from-workflow-json report.json --include issues,prs,contributors,health --format json
```

Send:

```bash
go run . feishu +notify --from-workflow-json report.json --webhook-url "$FEISHU_WEBHOOK_URL" --secret "$FEISHU_WEBHOOK_SECRET" --send
```

### Behavior

```text
Requires --from-workflow-json.
Uses include filter.
Default output is JSON preview.
--send sends a card.
No GitLink write.
```

## Phase 9: Weekly Report Command

### Objective

Create a weekly report from workflow JSON.

### Commands

```bash
go run . feishu +weekly-report --from-workflow-json report.json --format markdown
go run . feishu +weekly-report --from-workflow-json report.json --format json
go run . feishu +weekly-report --from-workflow-json report.json --webhook-url "$FEISHU_WEBHOOK_URL" --secret "$FEISHU_WEBHOOK_SECRET" --send
```

### Behavior

```text
Default output is markdown.
JSON output returns structured WeeklyReport.
--send sends a Feishu card after local report generation.
No GitLink write.
```

## Phase 10: Bitable Schema

### Objective

Generate recommended Bitable schema. Do not connect to Feishu OpenAPI.

### Files

```text
shortcuts/feishu/schema.go
shortcuts/feishu/schema_test.go
```

### Command

```bash
go run . feishu +bitable-schema --tables issues,prs,contributors,reports --format markdown
go run . feishu +bitable-schema --tables issues,prs,contributors,reports --format json
```

### Tables

```text
Issues
Pull Requests
Contributors
Weekly Reports
```

### Rules

```text
Do not create Base.
Do not create tables.
Do not create fields.
Do not create views.
Do not call Feishu OpenAPI.
```

## Phase 11: Bitable Records

### Objective

Generate Bitable-ready records from workflow JSON. Output only.

### Files

```text
shortcuts/feishu/bitable.go
shortcuts/feishu/bitable_test.go
```

### Command

```bash
go run . feishu +bitable-records --from-workflow-json report.json --tables issues,prs,contributors,reports --format json
```

### Rules

```text
Requires --from-workflow-json.
Only outputs records.
Does not read app token.
Does not read table IDs.
Does not call Feishu OpenAPI.
Does not create, update, or upsert.
```

## Phase 12: Skill and Docs

### Add

```text
skills/gitlink-feishu/SKILL.md
docs/feishu-integration.md
docs/feishu-security.md
docs/feishu-bitable-schema.md
examples/feishu/
  bot-test.md
  notify.md
  weekly-report.md
  bitable-schema.md
  bitable-records.md
```

### Content Rules

```text
Describe engineering usage only.
Default to preview examples.
Show --send only for bot cards.
State that Bitable output is local-only in this task.
Do not include non-engineering packaging language.
```

## Phase 13: Final Verification

### Commands

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
```

If dependencies are already available or a temporary proxy is set:

```bash
go test ./...
```

### Completion Report

Create:

```text
reports/FEISHU_TASK_COMPLETION.md
```

Include:

```text
implemented commands
changed files
preview behavior
explicit send behavior
mock test coverage
redaction checks
test results
excluded capabilities
```

## Final Acceptance

```text
feishu --help works.
+bot-test previews JSON.
+bot-test --send is covered by mock HTTP tests.
+notify reads workflow JSON and previews card JSON.
+notify --send is covered by mock HTTP tests.
+weekly-report reads workflow JSON and outputs markdown/json.
+bitable-schema outputs markdown/json.
+bitable-records outputs records JSON.
No output leaks full webhook URL or secret.
No Feishu send happens without --send.
No GitLink remote write exists.
No real Bitable write exists.
Targeted tests pass.
```

