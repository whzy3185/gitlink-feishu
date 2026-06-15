---
name: gitlink-feishu
version: 1.0.0
description: "Export GitLink workflow JSON to Feishu bot cards, DocX/Wiki reports, and Bitable-ready dry-run records."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli feishu --help"
---

# gitlink-feishu

Use this skill when a user needs to send or export GitLink workflow analysis to Feishu.

## Rules

- Prefer local preview first.
- Use `--send` only when the user explicitly wants a Feishu network write.
- Never use BotBuilder or Robot Assistant workflows.
- Do not write to GitLink resources.
- Do not print webhook URLs, app secrets, or access tokens.
- Use `+bitable-records` for dry-run output only; do not claim that Bitable has been written.

## Workflow

Generate workflow JSON:

```bash
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format json > report.json
```

Preview a Feishu card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

Send a Feishu card:

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Preview a document export:

```bash
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url "<wiki_url>" --format markdown
```

Export to DocX or Wiki:

```bash
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url "<wiki_url>" --send --format table
```

Generate Bitable-ready records:

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## Feishu Setup

Custom bot commands need:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

DocX/Wiki export needs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
```

The self-built app must also have permission to write the target DocX/Wiki page or folder.

