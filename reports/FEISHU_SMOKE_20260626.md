# Feishu Smoke Report

Date: 2026-06-26 20:58:49 +08:00

## Branch

```text
feat/feishu-export-clean
```

## Commit

```text
73da46c143b37cb2b26e9e624b8c39963ad52d77
```

The worktree was dirty during this smoke run because the Feishu implementation
and documentation were still being updated.

## Mode

```text
real Feishu test enterprise plus local previews
```

## Test Environment

```text
Feishu test enterprise: used
Custom bot in test group: used
Self-built app with broad test permissions: used
Feishu DocX target: used
Feishu Base target: used
Feishu Task API: used
GitLink real repository data: Gitlink/gitlink-cli
Reference PR IDs for smoke notes: 95, 29, 75
GitLink write operations: not used
```

All Feishu resource IDs, tokens, webhook URLs, app credentials, table IDs, and
document IDs were kept in `.local/feishu-gitlink.env.ps1` and are not committed.

## Redacted Environment Presence

| Variable | Present? | Notes |
| --- | --- | --- |
| `FEISHU_WEBHOOK_URL` | present | redacted in CLI output |
| `FEISHU_WEBHOOK_SECRET` | present | redacted in CLI output |
| `FEISHU_APP_ID` | present | redacted where printed |
| `FEISHU_APP_SECRET` | present | never printed |
| `FEISHU_FOLDER_TOKEN` | present | redacted |
| `FEISHU_DOCUMENT_ID` | present | redacted |
| `FEISHU_BASE_APP_TOKEN` | present | redacted |
| `FEISHU_REPORT_TABLE_ID` | present | same test table as other table envs |
| `FEISHU_ISSUE_TABLE_ID` | present | same test table as other table envs |
| `FEISHU_PR_TABLE_ID` | present | same test table as other table envs |
| `FEISHU_CONTRIBUTOR_TABLE_ID` | present | same test table as other table envs |
| `FEISHU_TASK_TABLE_ID` | present | same test table as other table envs |
| `FEISHU_TASK_PROJECT_ID` | missing | optional; current request body does not place tasks into project/section |
| `FEISHU_TASK_SECTION_ID` | missing | optional; current request body does not place tasks into project/section |
| `GITLINK_OWNER` | present | `Gitlink` |
| `GITLINK_REPO` | present | `gitlink-cli` |
| `GITLINK_TEST_PR_IDS` | present | `95,29,75` |
| `GITLINK_TOKEN` | missing | not required for the read-only workflow report in this run |

## GitLink Report Source

Command:

```powershell
go run . workflow +repo-report --owner $env:GITLINK_OWNER --repo $env:GITLINK_REPO --format json > .local\report.json
go run . workflow +repo-report --owner $env:GITLINK_OWNER --repo $env:GITLINK_REPO --lang zh-CN --format json > .local\report.zh-CN.json
```

Result:

| Item | Value |
| --- | --- |
| Repository | `Gitlink/gitlink-cli` |
| Report score | `49` |
| Risk level | `high` |
| Health score | `58` |
| Issues | `19` |
| Pull requests | `10` |
| Source | `remote-read-only-fetch` |

The workflow command does not currently filter the report by explicit PR IDs, so
`GITLINK_TEST_PR_IDS` is recorded as smoke context rather than a hard filter.

## Real Feishu Results

| Command | Result | Details |
| --- | --- | --- |
| `feishu +bot-test --send` | pass | custom bot returned Feishu code `0` |
| `feishu +notify --send` | pass | English/default workflow card delivered |
| `feishu +weekly-report --send` | pass | weekly report card delivered |
| `feishu +owner-digest --send` | pass | owner digest card delivered |
| `feishu +contributor-digest --send` | pass | contributor digest card delivered |
| `feishu +notify --lang zh-CN --send` | pass | Chinese workflow card delivered |
| `feishu +owner-digest --lang zh-CN --send` | pass | Chinese owner digest delivered |
| `feishu +contributor-digest --lang zh-CN --send` | pass | Chinese contributor digest delivered |
| `feishu +doc-export --send` | pass | appended 9 DocX blocks to the configured document |
| `feishu +doc-export --lang zh-CN --send` | pass | appended 9 localized DocX blocks |
| `feishu +bitable-sync --tables reports --send` | pass after table fields were added | created the report record |
| `feishu +bitable-sync --tables reports,issues,prs,contributors,tasks --send` | pass | updated 1 report, created 5 issue buckets, 2 PR buckets, 1 contributor bucket, 7 task buckets |
| `feishu +bitable-sync --lang zh-CN --send` | pass | updated existing records from the Chinese workflow JSON |
| `feishu +task-preview --lang zh-CN` | pass | generated 7 Chinese task candidates |
| `feishu +task-create --lang zh-CN --send` | pass | created 7 Feishu tasks |

## Bitable Setup Observation

The provided Feishu Base URLs pointed to one Base and one table with multiple
views. The test enterprise initially had only the default fields. A direct
OpenAPI inspection found one table and the default fields only, so the test
table was expanded with the fields expected by the CLI records:

```text
unique_key, repository, health_score, risk_level, report_score,
issue_total, issue_high_risk, issue_missing_info, pr_total, pr_high_risk,
review_focus_count, generated_at, source, doc_url, issue_group, priority,
count, risk_reason, recommended_action, gitlink_url, pr_group, review_focus,
contributor, role, open_items, risk_items, task_title, task_type, source_type,
source_key, recommended_owner, status, due_hint
```

This confirms that `+bitable-sync` can search, create, and update records when
the target table already has compatible fields. It does not yet create Base
tables or views itself.

## i18n Result

Feishu command-level Chinese output is usable:

```text
workflow +repo-report --lang zh-CN
feishu +notify --lang zh-CN
feishu +owner-digest --lang zh-CN
feishu +contributor-digest --lang zh-CN
feishu +doc-export --lang zh-CN
feishu +task-preview --lang zh-CN
feishu +task-create --lang zh-CN
```

The Feishu module localizes stable card labels, digest headings, common
recommendations, DocX block headings, and task candidate titles. For best
results, generate the source workflow report with `--lang zh-CN` and pass
`--lang zh-CN` again to the Feishu command.

Repository-wide i18n formatting check:

```text
go run ./internal/i18n/cmd/check
```

Result:

```text
fail: internal/i18n/locales/en-US.json is not formatted
```

That appears to be an existing locale formatting issue outside the Feishu
module. It was not fixed in this smoke run to avoid unrelated locale churn.

## Tests

| Check | Result |
| --- | --- |
| `go test ./shortcuts/feishu` | pass |
| `go test ./shortcuts/workflow` | pass |
| `go test ./shortcuts` | pass |
| `go test ./...` | pass |
| Raw secret scan over tracked/unignored candidate files | pass |

## Known Limitations

```text
1. Bitable sync requires existing Base/table/fields; CLI does not create tables or views.
2. The current smoke used one test table for all record groups because the provided links were one table with multiple views.
3. Current Bitable records are summary buckets, not row-level PR/Issue/CI records.
4. Feishu task creation does not yet map project/section placement into the request body.
5. Feishu-side task dedupe/search is not implemented; avoid repeated real task-create runs unless duplicates are acceptable.
6. No Feishu callback server is implemented.
7. No GitLink write operation is implemented.
8. Screenshots still need to be captured manually from the Feishu UI.
```

## Screenshot Checklist

Run:

```powershell
.\scripts\feishu-gitlink-screenshot-check.ps1
```

Manual captures still needed:

```text
docs/images/feishu-bot-card.png
docs/images/feishu-weekly-report.png
docs/images/feishu-owner-digest.png
docs/images/feishu-contributor-digest.png
docs/images/feishu-bitable-preview.png
docs/images/feishu-bitable-sync.png
docs/images/feishu-docx-wiki.png
docs/images/feishu-task-create.png
docs/images/feishu-smoke-terminal.png
docs/images/feishu-env-redacted.png
```

Do not fabricate screenshots. Redact IDs and tokens before committing any image.
