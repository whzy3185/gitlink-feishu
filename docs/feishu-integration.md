# Feishu Integration

`gitlink-cli feishu` exports local GitLink workflow JSON to Feishu collaboration surfaces.

The commands are intentionally one-way:

```text
workflow JSON -> local preview
workflow JSON -> Feishu bot card
workflow JSON -> Feishu DocX / Wiki report
workflow JSON -> Bitable-ready dry-run records
```

## Safety Model

Default behavior is local preview.

Network operations require `--send`.

`--send` and `--dry-run` cannot be used together.

The implementation does not write to GitLink resources, does not close issues, does not comment on PRs, and does not merge code.

## Custom Bot

Use a Feishu custom group bot for notification cards.

Environment:

```powershell
$env:FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/..."
$env:FEISHU_WEBHOOK_SECRET="optional signing secret"
```

Preview:

```bash
gitlink-cli feishu +bot-test --format json
```

Send:

```bash
gitlink-cli feishu +bot-test --send --format table
```

Send a workflow report card:

```bash
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format json > report.json
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Send a card with a DocX or Wiki link:

```bash
gitlink-cli feishu +notify \
  --from-workflow-json report.json \
  --doc-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```

## DocX / Wiki Export

Use a Feishu self-built app for document export.

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

Append report blocks to a Wiki-backed DocX:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```

Create a new DocX in a folder:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --folder-token "<folder_token>" \
  --title "GitLink workflow report" \
  --send \
  --format table
```

Required Feishu setup:

```text
1. The self-built app must have DocX / Drive application scopes.
2. The app permission version must be published and approved.
3. The target Wiki / DocX / folder must grant the app write permission.
```

If Feishu returns `1770032: forBidden`, the token is valid but the app cannot write to the target document. Grant the app access to that Wiki/DocX page or use a folder where the app can create documents.

## Bitable Dry Run

Generate recommended table schemas:

```bash
gitlink-cli feishu +bitable-schema --format markdown
```

Generate Bitable-ready records:

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

These commands do not call Bitable OpenAPI and do not create, update, or upsert records.

