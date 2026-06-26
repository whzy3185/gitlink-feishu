# Feishu Integration

`gitlink-cli feishu` exports local GitLink workflow JSON to Feishu.

The stable command path is intentionally narrow:

```text
workflow JSON -> local preview
workflow JSON -> Feishu custom bot card
workflow JSON -> weekly report
workflow JSON -> owner / contributor digest
workflow JSON -> Bitable schema / records dry-run
workflow JSON -> task candidates
```

## Stable Commands

```text
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +owner-digest
gitlink-cli feishu +contributor-digest
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
gitlink-cli feishu +task-preview
```

Experimental commands:

```text
gitlink-cli feishu +doc-export
gitlink-cli feishu +bitable-sync
gitlink-cli feishu +task-create
```

Experimental commands use Feishu self-built app OpenAPI and are not part of the stable custom-bot path.

## Safety Model

- Default behavior is local preview.
- Real Feishu bot delivery requires `--send`.
- `--send` and `--dry-run` cannot be used together.
- Webhook URLs are redacted in command output.
- Secrets and tokens are never intentionally printed.
- The stable commands do not write to GitLink resources.
- `+bitable-schema`, `+bitable-records`, and `+task-preview` are dry-run only and do not call Feishu OpenAPI.
- `+doc-export`, `+bitable-sync`, and `+task-create` require explicit `--send` before attempting Open Platform writes.
- Feishu card buttons are navigation-only.
- GitLink write operations are not implemented.

## Custom Bot Setup

Use a Feishu custom group bot for notification cards.

Recommended local setup:

```powershell
.\scripts\feishu-gitlink-setup.ps1
.\scripts\feishu-gitlink-env-check.ps1 -Layer stable
```

The setup wizard opens Feishu / GitLink pages and stores values only in `.local/feishu-gitlink.env.ps1`.

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

## Owner and Contributor Digests

Owner digests summarize the repository state for maintainers:

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --format markdown
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
```

Contributor digests summarize role-oriented follow-up work:

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --format markdown
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
```

These digests are not Feishu-user-personalized. They do not use `open_id`, `union_id`, or personal routing.

## Bitable Dry Run

Generate recommended table schemas:

```bash
gitlink-cli feishu +bitable-schema --format markdown
```

Generate Bitable-ready records:

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

Default tables:

```text
reports
issues
prs
contributors
tasks
```

These records are summary records derived from workflow repo-report JSON. They are not a per-issue or per-PR synchronization.

## Experimental Bitable Sync

`feishu +bitable-sync` reuses the records produced by `+bitable-records`.

Preview:

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,contributors,tasks \
  --format table
```

Write with existing Base app and table IDs:

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,tasks \
  --send \
  --format table
```

The command searches by `unique_key`, updates when found, creates when missing, and never deletes records. If search fails, it falls back to create-only and prints diagnostics.

## Task Preview and Experimental Task Create

Preview task candidates:

```bash
gitlink-cli feishu +task-preview --from-workflow-json report.json --format markdown
```

Attempt real Feishu task creation:

```bash
gitlink-cli feishu +task-create --from-workflow-json report.json --send --format table
```

`+task-create` is experimental and requires Feishu Task scopes. It does not create or update GitLink issues.

## Role-Aware Collaboration Roadmap

The Feishu integration is designed to support two different notification modes:

```text
Owner / maintainer: summarized digest.
Contributor: immediate personal feedback.
```

Owner-oriented cards should group PRs by review stage instead of sending one message for every PR event. Recommended stages:

```text
blue: new or unreviewed
grey: active review
green: close to merge or merged
yellow: needs rebase
orange: major changes requested
red: blocked
```

Contributor notifications are different. A contributor should receive fast feedback when their own PR is reviewed, commented on, blocked by rebase/conflict, approved, merged, or closed.

Long-form project material should be exported to Feishu Docs / Wiki:

```text
README summary
contribution guide
owner digest archive
milestone plan
PR stage table
```

Milestone and Gantt support should start as Bitable-ready records and document sections. Real Bitable writes and view creation require a separate permissioned OpenAPI design.

Detailed design:

```text
feishu-export-design/ROLE_BASED_COLLABORATION.md
```

## Experimental DocX / Wiki Export

`feishu +doc-export` is experimental because it uses Feishu self-built app credentials and writes to DocX / Wiki through OpenAPI.

Environment:

See `docs/FEISHU_ENVIRONMENT.md` for all Open Platform variables.

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

## Layered Documentation

Detailed boundaries:

```text
docs/FEISHU_CAPABILITY_LAYERS.md
docs/FEISHU_ENVIRONMENT.md
reports/FEISHU_PERMISSION_MATRIX.md
reports/FEISHU_LOCAL_TESTING_GUIDE.md
```

Scripted smoke test:

```powershell
.\scripts\feishu-gitlink-smoke.ps1 -Mode preview
.\scripts\feishu-gitlink-smoke.ps1 -Mode stable
.\scripts\feishu-gitlink-smoke.ps1 -Mode open-platform
.\scripts\feishu-gitlink-smoke.ps1 -Mode all
```
