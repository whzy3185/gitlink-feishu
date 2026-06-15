# Feishu Integration

`gitlink-cli feishu` exports local GitLink workflow JSON to Feishu.

The stable command path is intentionally narrow:

```text
workflow JSON -> local preview
workflow JSON -> Feishu custom bot card
workflow JSON -> weekly report
workflow JSON -> Bitable schema / records dry-run
```

## Stable Commands

```text
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
```

`feishu +doc-export` exists as an experimental command. It uses Feishu self-built app OpenAPI and is not part of the clean first-path workflow.

## Safety Model

- Default behavior is local preview.
- Real Feishu bot delivery requires `--send`.
- `--send` and `--dry-run` cannot be used together.
- Webhook URLs are redacted in command output.
- Secrets and tokens are never intentionally printed.
- The stable commands do not write to GitLink resources.
- Bitable commands are dry-run only and do not call Bitable OpenAPI.

## Custom Bot Setup

Use a Feishu custom group bot for notification cards.

Environment:

```powershell
$env:FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/..."
$env:FEISHU_WEBHOOK_SECRET="optional signing secret"
```

Preview a test card:

```bash
gitlink-cli feishu +bot-test --format json
```

Send a test card:

```bash
gitlink-cli feishu +bot-test --send --format table
```

## Workflow Report Card

Generate workflow JSON:

```bash
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format json > report.json
```

On Windows PowerShell, redirected files may be written with UTF-16 encoding. `feishu` workflow JSON readers accept UTF-8 and UTF-16 BOM files so the redirected output above can be consumed directly.

Preview a card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

Send a card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Send a card with an existing Feishu document or Wiki link:

```bash
gitlink-cli feishu +notify \
  --from-workflow-json report.json \
  --doc-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```

## Weekly Report

Render markdown:

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

Send a weekly summary card:

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --send --format table
```

## Bitable Dry Run

Generate recommended table schemas:

```bash
gitlink-cli feishu +bitable-schema --format markdown
```

Generate Bitable-ready records:

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

These records are summary records derived from workflow repo-report JSON. They are not a per-issue or per-PR synchronization.

## Experimental DocX / Wiki Export

`feishu +doc-export` is experimental because it uses Feishu self-built app credentials and writes to DocX / Wiki through OpenAPI.

Environment:

```powershell
$env:FEISHU_APP_ID="cli_xxx"
$env:FEISHU_APP_SECRET="..."
```

Preview:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --format markdown
```

Write to DocX / Wiki:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```

Required Feishu setup:

```text
1. The self-built app must have approved DocX / Drive scopes.
2. The target Wiki / DocX / folder must grant the app write permission.
3. If Feishu returns 1770032: forBidden, credentials are valid but the app cannot write the target document.
```
