---
name: gitlink-feishu
version: 1.0.0
description: "Export GitLink workflow JSON to Feishu custom bot cards, digests, Bitable-ready records, and experimental Open Platform validation commands."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli feishu --help"
---

# gitlink-feishu

Use this skill when a user needs to export GitLink workflow analysis into Feishu.

## Purpose

Stable path:

```text
workflow JSON -> Feishu bot card / weekly report / Bitable dry-run records
workflow JSON -> owner digest / contributor digest / task preview
```

Experimental path:

```text
workflow JSON -> Feishu DocX / Wiki export / Bitable sync / Task create
```

## Inputs

Workflow JSON should usually come from:

```bash
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format json > report.json
```

## Safety Rules

- Preview first.
- Use `--send` only when the user explicitly wants a Feishu network write.
- Never use BotBuilder or Robot Assistant workflows.
- Do not write to GitLink resources.
- Do not print webhook URLs, app secrets, access tokens, or table tokens.
- Treat `+bitable-schema`, `+bitable-records`, and `+task-preview` as local dry-run commands only.
- Treat `+doc-export` as experimental because it uses self-built app OpenAPI and document write permissions.
- Treat `+bitable-sync` and `+task-create` as experimental because they use self-built app OpenAPI and resource permissions.

## Preview Flow

Preview a card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

Render a weekly report:

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

Preview owner and contributor digests:

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --format markdown
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --format markdown
```

Generate Bitable schemas:

```bash
gitlink-cli feishu +bitable-schema --format markdown
```

Generate Bitable-ready records:

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

Preview task candidates:

```bash
gitlink-cli feishu +task-preview --from-workflow-json report.json --format markdown
```

## Send Flow

Custom bot commands need:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

Send a card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Send a weekly report card:

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --send --format table
```

Send owner and contributor digest cards:

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
```

## Experimental Doc Export

DocX / Wiki export needs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
```

Preview only:

```bash
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url "<wiki_url>" --format markdown
```

Write to Feishu:

```bash
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url "<wiki_url>" --send --format table
```

If Feishu returns `1770032: forBidden`, the app token is valid but the app cannot write to the target DocX/Wiki page or folder.

## Experimental Bitable Sync And Task Create

Open Platform validation commands need:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN for bitable-sync
FEISHU_REPORT_TABLE_ID / FEISHU_ISSUE_TABLE_ID / FEISHU_PR_TABLE_ID for selected tables
```

Preview Bitable sync:

```bash
gitlink-cli feishu +bitable-sync --from-workflow-json report.json --tables reports,issues,prs,tasks --format table
```

Write Bitable records:

```bash
gitlink-cli feishu +bitable-sync --from-workflow-json report.json --tables reports,issues,prs,tasks --send --format table
```

Create Feishu tasks:

```bash
gitlink-cli feishu +task-create --from-workflow-json report.json --send --format table
```

## Non-Goals

- No GitLink remote writes.
- No GitLink comments, issue closure, merge, or webhook creation.
- No Bitable real writes in the stable path.
- No Feishu task creation in the stable path.
- No BotBuilder integration.
- No automatic Feishu permission changes.

