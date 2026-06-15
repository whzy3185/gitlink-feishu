# Feishu Export Technical Plan

## Latest Master Anchors

Current command registration:

```text
shortcuts/register.go
```

Add imports:

```go
github.com/gitlink-org/gitlink-cli/shortcuts/feishu
```

Add group:

```go
"feishu": feishu.Shortcuts(),
```

Add description:

```go
"feishu": "Export GitLink workflow data to Feishu cards and Bitable-ready records",
```

Shortcut mounting:

```text
shortcuts/common/runner.go
```

Important behavior:

- Boolean flags are read from Cobra as strings in `RuntimeContext.Args`.
- A bool flag with default `true` cannot tell whether the user explicitly set it. Therefore `--send` should be the positive action flag, and `--dry-run` should remain a preview indicator.
- If implementation must detect explicit `--dry-run`, it needs a command-specific Cobra command instead of the generic shortcut runner. Avoid that for the first implementation.

## Send Flag Decision

Use this rule:

```text
--send=false or omitted: preview only.
--send=true and dry-run=false: send.
--send=true and dry-run=true: error.
```

Since generic shortcut flags cannot detect whether `--dry-run` was explicitly passed, set `--dry-run` default to `false` in Feishu commands and treat preview as `!Send`. This avoids a default conflict where `--send` would always collide with default `--dry-run=true`.

Effective mode:

```go
Preview := !opts.Send
```

Validation:

```go
if opts.Send && opts.DryRun {
    return error
}
if opts.Send && opts.WebhookURL == "" {
    return error
}
```

User-facing meaning:

```text
No --send: preview.
--dry-run: preview and forbid send.
--send: real bot send.
--send --dry-run: invalid.
```

## Output Strategy

Do not extend `internal/output` in the first implementation.

Reason:

- Global output supports `json`, `yaml`, and generic `table`.
- Workflow uses local renderers for markdown.
- Feishu needs markdown for weekly reports and schema docs.

Implement local Feishu render helpers:

```go
func renderJSON(w io.Writer, value any) error
func renderMarkdown(w io.Writer, value any) error
func renderTable(w io.Writer, value any) error
func normalizeFormat(format string, defaultFormat string) string
```

Default formats:

```text
+bot-test: json
+notify: json
+weekly-report: markdown
+bitable-schema: markdown
+bitable-records: json
```

## Workflow JSON Input

First implementation only supports local JSON files.

Expected source:

```bash
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format json > report.json
```

Mapping should accept both shapes if found:

1. Raw `RepoReportResult`.
2. Envelope-like object with `data`.

Fields to use:

```text
repository
health.health_score
health.risk_level
issue_summary.total
issue_summary.high_risk
issue_summary.missing_info
pr_summary.total
pr_summary.high_risk
pr_summary.review_focus
recommendations
risk_level
report_score
source
```

Do not import unexported workflow readers. Use JSON mapping to avoid changing workflow internals.

## Feishu Signature

Use Feishu custom bot signing:

```text
string_to_sign = timestamp + "\n" + secret
sign = base64(hmac_sha256(string_to_sign, secret))
```

Implementation notes:

- `timestamp` is Unix seconds and serialized as a string.
- Empty secret means no timestamp/sign.
- Tests use fixed timestamp.
- Errors must be redacted.

## Feishu Card Shape

Use custom bot interactive card payload:

```json
{
  "msg_type": "interactive",
  "card": {
    "header": {
      "title": {
        "tag": "plain_text",
        "content": "GitLink Project Activity"
      }
    },
    "elements": []
  }
}
```

Envelope with signature:

```json
{
  "timestamp": "1710000000",
  "sign": "redacted in logs",
  "msg_type": "interactive",
  "card": {}
}
```

## HTTP Client

Use package-local client:

```go
type Client struct {
    HTTPClient *http.Client
}
```

Use `httptest.Server` for all tests.

Do not use `internal/client.Client` for Feishu because it appends GitLink `.json` suffixes and injects GitLink auth.

## Naming

CLI flags:

```text
prs
--include issues,prs,contributors,health
--tables issues,prs,contributors,reports
```

User-facing labels:

```text
Pull Requests
```

Internal model names:

```text
PullRequestSummary
NewPullRequests
MergedPullRequests
```

Do not use `pulls` in new user-facing flags.

## Bitable Scope

First implementation:

```text
+bitable-schema
+bitable-records
```

No Feishu OpenAPI calls.

No environment variables for app tokens or table IDs.

No tenant token.

No create/update/upsert.

The records command outputs local JSON only:

```go
type BitableRecordsOutput struct {
    Repository string `json:"repository"`
    Tables []BitableTableRecords `json:"tables"`
}

type BitableTableRecords struct {
    Table string `json:"table"`
    Records []BitableRecord `json:"records"`
}

type BitableRecord struct {
    UniqueKey string `json:"unique_key"`
    Fields map[string]any `json:"fields"`
}
```

## Tests To Add

```text
shortcuts/feishu/options_test.go
shortcuts/feishu/redact_test.go
shortcuts/feishu/signer_test.go
shortcuts/feishu/card_test.go
shortcuts/feishu/client_test.go
shortcuts/feishu/mapper_test.go
shortcuts/feishu/schema_test.go
shortcuts/feishu/bitable_test.go
shortcuts/feishu/commands_test.go
```

Minimum assertions:

```text
No --send means no HTTP.
--send means HTTP in mock tests.
--send --dry-run errors.
Missing webhook URL with --send errors.
Errors redact webhook URL and secret.
Workflow JSON can be loaded from fixture.
Missing fields do not panic.
Weekly report markdown is stable.
Bitable schema JSON is parseable.
Bitable records JSON is parseable.
```

## Baseline Verification

Current latest master passes with:

```powershell
$env:GOPROXY='https://goproxy.cn,direct'; go test ./...
```

Keep this as the baseline before implementation.

