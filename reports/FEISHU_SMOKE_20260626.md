# Feishu Smoke Report

Date: 2026-06-26 16:29:18 +08:00

## Branch

```text
feat/feishu-export-clean
```

## Commit

```text
9255518304e1a6b0fba9f9e5eee9bdf4f62d8e04
```

## Mode

```text
preview
```

## Redacted Environment Presence

| Variable | Present? |
| --- | --- |
| `FEISHU_WEBHOOK_URL` | missing |
| `FEISHU_WEBHOOK_SECRET` | missing |
| `FEISHU_APP_ID` | missing |
| `FEISHU_APP_SECRET` | missing |
| `FEISHU_WIKI_URL` | missing |
| `FEISHU_WIKI_NODE_TOKEN` | missing |
| `FEISHU_FOLDER_TOKEN` | missing |
| `FEISHU_BASE_APP_TOKEN` | missing |
| `FEISHU_REPORT_TABLE_ID` | missing |
| `FEISHU_ISSUE_TABLE_ID` | missing |
| `FEISHU_PR_TABLE_ID` | missing |
| `FEISHU_CONTRIBUTOR_TABLE_ID` | missing |
| `FEISHU_TASK_TABLE_ID` | missing |
| `FEISHU_TASK_PROJECT_ID` | missing |
| `FEISHU_TASK_SECTION_ID` | missing |
| `GITLINK_OWNER` | missing |
| `GITLINK_REPO` | missing |
| `GITLINK_TEST_PR_IDS` | missing |
| `GITLINK_TOKEN` | missing |

## Results

| Command | Result | Details |
| --- | --- | --- |
| feishu help | pass | exit=0 |
| feishu +owner-digest help | pass | exit=0 |
| feishu +contributor-digest help | pass | exit=0 |
| feishu +bitable-sync help | pass | exit=0 |
| feishu +task-preview help | pass | exit=0 |
| feishu +task-create help | pass | exit=0 |
| workflow +repo-report | pass | report=.local/report.json; owner=Gitlink; repo=gitlink-cli |
| notify preview | pass | exit=0 |
| weekly report preview | pass | exit=0 |
| owner digest preview | pass | exit=0 |
| contributor digest preview | pass | exit=0 |
| bitable records preview | pass | exit=0 |
| task preview | pass | exit=0 |

## Notes

- No .local/feishu-gitlink.env.ps1 file found. Preview smoke can run with public fallback data; real sends are skipped.
- GITLINK_OWNER/GITLINK_REPO were missing. Preview smoke used public Gitlink/gitlink-cli as a fallback.

## Terminal Log

Local redacted terminal log: `reports/feishu-real-smoke-terminal.log`

This log file is ignored and should not be committed after real runs.

## Screenshot Checklist

Run:

```powershell
.\scripts\feishu-gitlink-screenshot-check.ps1
```

Do not fabricate screenshots. Capture missing images manually after real Feishu runs.
