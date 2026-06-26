# Feishu Smoke Report

Date: 2026-06-26

## Branch

```text
feat/feishu-export-clean
```

## Commit

```text
working tree smoke before final implementation commit; base HEAD before this implementation: bcdab0b
```

## Test Environment

```text
Local OS: Windows / PowerShell
Repository: gitlink-cli-feishu-clean
Feishu test enterprise: available only when local environment variables are configured
Real GitLink repo data: public Gitlink/gitlink-cli workflow report generated through gitlink-cli
Previous 3 GitLink PR IDs: not available in current shell; GITLINK_TEST_PR_IDS was not set
```

## Redacted Environment Presence

This section must record presence only, not raw values:

| Variable | Present? |
| --- | --- |
| `FEISHU_WEBHOOK_URL` | missing |
| `FEISHU_WEBHOOK_SECRET` | missing |
| `FEISHU_APP_ID` | missing |
| `FEISHU_APP_SECRET` | missing |
| `FEISHU_WIKI_URL` | missing |
| `FEISHU_BASE_APP_TOKEN` | missing |
| `FEISHU_REPORT_TABLE_ID` | missing |
| `FEISHU_ISSUE_TABLE_ID` | missing |
| `FEISHU_PR_TABLE_ID` | missing |
| `FEISHU_TASK_PROJECT_ID` | missing |
| `GITLINK_OWNER` | missing |
| `GITLINK_REPO` | missing |
| `GITLINK_TEST_PR_IDS` | missing |

## Commands Run So Far

```bash
go run . feishu --help
go run . feishu +owner-digest --help
go run . feishu +bitable-sync --help
go run . feishu +task-create --help

go run . feishu +owner-digest --from-workflow-json shortcuts/workflow/testdata/repo_report.json --format table
go run . feishu +contributor-digest --from-workflow-json shortcuts/workflow/testdata/repo_report.json --format table
go run . feishu +bitable-records --from-workflow-json shortcuts/workflow/testdata/repo_report.json --tables reports,issues,prs,contributors,tasks --format table
go run . feishu +bitable-sync --from-workflow-json shortcuts/workflow/testdata/repo_report.json --tables reports,tasks --format table
go run . feishu +task-preview --from-workflow-json shortcuts/workflow/testdata/repo_report.json --format table

$report = Join-Path $env:TEMP 'gitlink-feishu-report-20260626.json'
go run . workflow +repo-report --owner Gitlink --repo gitlink-cli --format json | Set-Content -Encoding utf8 $report
go run . feishu +notify --from-workflow-json $report --format table
go run . feishu +owner-digest --from-workflow-json $report --format table
go run . feishu +contributor-digest --from-workflow-json $report --format table
go run . feishu +bitable-records --from-workflow-json $report --tables reports,issues,prs,contributors,tasks --format table
go run . feishu +task-preview --from-workflow-json $report --format table
go run . feishu +bitable-sync --from-workflow-json $report --tables reports,issues,prs,tasks --format table
go run . feishu +doc-export --from-workflow-json $report --format table

gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

## Outputs Summary

| Step | Result | Notes |
| --- | --- | --- |
| `feishu --help` | pass | new owner/contributor digest, bitable sync, task preview/create commands visible |
| owner digest preview | pass | role summary generated |
| contributor digest preview | pass | role summary generated |
| bitable records preview | pass | reports/issues/prs/contributors/tasks generated |
| bitable sync preview | pass | preview only, no OpenAPI call |
| task preview | pass | task candidates generated |
| Feishu unit/mock tests | pass | `go test ./shortcuts/feishu` |
| workflow tests | pass | `go test ./shortcuts/workflow` |
| shortcuts tests | pass | `go test ./shortcuts` |
| full repository tests | pass | `go test ./...` |
| public GitLink repo report | pass | `Gitlink/gitlink-cli` report generated in temp directory |
| public GitLink notify preview | pass | preview only, no webhook call |
| public GitLink owner digest | pass | risk/score summary generated |
| public GitLink contributor digest | pass | role-oriented summary generated |
| public GitLink Bitable records | pass | reports/issues/prs/contributors/tasks generated |
| public GitLink Bitable sync preview | pass | preview only, table IDs missing by design |
| public GitLink DocX/Wiki preview | pass | preview only, no Open Platform call |

## Real Feishu Webhook Result

```text
not executed: FEISHU_WEBHOOK_URL was not present in the current shell.
```

## DocX / Wiki Result

```text
not executed: FEISHU_APP_ID, FEISHU_APP_SECRET, and document target variables were not present in the current shell.
Preview passed with public GitLink report.
```

## Bitable Sync Result

```text
not executed: FEISHU_APP_ID, FEISHU_APP_SECRET, FEISHU_BASE_APP_TOKEN, and table IDs were not present in the current shell.
Preview passed with public GitLink report.
```

## Task Creation Result

```text
not executed: FEISHU_APP_ID and FEISHU_APP_SECRET were not present in the current shell.
Task preview passed with public GitLink report.
```

## Failure Diagnostics

```text
None from local preview, public GitLink read smoke, and unit/mock tests.
Real Open Platform failures must be recorded with endpoint category, HTTP status or Feishu code when available, redacted target type, likely reason, and required permission.
```

## Screenshots Or Terminal Logs

Expected screenshot paths are listed in `docs/PR_VISUAL_GUIDE.md`.

Do not fabricate screenshots.
