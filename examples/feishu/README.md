# Feishu Export Examples

## Stable Workflow

### 1. Generate Workflow JSON

```bash
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format json > report.json
```

### 2. Preview Notification Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

### 3. Send Notification Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

### 4. Render Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

### 5. Generate Bitable Schema

```bash
gitlink-cli feishu +bitable-schema --format markdown
```

### 6. Generate Bitable Records

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## Role-Aware Collaboration Direction

Owner digest:

```text
aggregate PR and workflow state
group PRs by review stage
send daily or weekly summary cards
link to Feishu Doc / Wiki full report
```

Contributor feedback:

```text
send fast notifications only for the contributor's own PR events
review comment
changes requested
needs rebase
approved
merged
closed
```

Planned color semantics:

```text
blue = new / unreviewed
green = close to merge
yellow = needs rebase
orange = major changes requested
red = blocked
grey = active review or closed
```

## Experimental DocX / Wiki Export

`+doc-export` is available for Feishu self-built app experiments. It is not part of the stable clean export path.

Preview:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --format markdown
```

Write:

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```
