# Feishu Local Testing Guide

Date: 2026-06-26

This guide verifies the layered Feishu integration without committing secrets.

## 1. Run Local Setup Wizard

```powershell
.\scripts\feishu-gitlink-setup.ps1
```

The setup wizard opens these pages as needed:

```text
Feishu custom bot documentation
Feishu developer console
Feishu Docs / Wiki
Feishu Base / Bitable
Feishu Tasks
GitLink
```

The user logs in and authorizes in the browser. Values are pasted into the local PowerShell prompt, not into ChatGPT.

The wizard writes only to:

```text
.local/feishu-gitlink.env.ps1
```

This file is ignored and must not be committed. A tracked empty example is available at:

```text
.local/feishu-gitlink.env.example.ps1
```

## 2. Check Local Environment

```powershell
.\scripts\feishu-gitlink-env-check.ps1 -Layer stable
.\scripts\feishu-gitlink-env-check.ps1 -Layer open-platform
.\scripts\feishu-gitlink-env-check.ps1 -Layer all
```

The checker prints only redacted values.

## 3. Run Feishu CLI Diagnostics

Local diagnostics:

```bash
gitlink-cli feishu +app-check --format table
gitlink-cli feishu +doc-check --format table
gitlink-cli feishu +bitable-check --tables reports,issues,prs,contributors,tasks --format table
gitlink-cli feishu +task-check --format table
```

Optional remote diagnostics:

```bash
gitlink-cli feishu +app-check --remote --format table
gitlink-cli feishu +doc-check --remote --format table
gitlink-cli feishu +bitable-check --tables reports,issues,prs,contributors,tasks --remote --format table
gitlink-cli feishu +task-check --remote --format table
```

Remote diagnostics call only read/check endpoints. They do not create DocX
blocks, Bitable records, Feishu tasks, or GitLink writes.

## 4. Configure GitLink Test Repository Manually If Needed

```powershell
$env:GITLINK_OWNER="OWNER"
$env:GITLINK_REPO="REPO"
```

If the current workflow command cannot filter specific PR IDs, keep the PR IDs in the smoke report:

```powershell
$env:GITLINK_TEST_PR_IDS="1,2,3"
```

## 5. Run Scripted Smoke Tests

Preview only:

```powershell
.\scripts\feishu-gitlink-smoke.ps1 -Mode preview
```

Stable custom bot sends:

```powershell
.\scripts\feishu-gitlink-smoke.ps1 -Mode stable
```

Experimental Open Platform sends:

```powershell
.\scripts\feishu-gitlink-smoke.ps1 -Mode open-platform
```

All available tests:

```powershell
.\scripts\feishu-gitlink-smoke.ps1 -Mode all
```

The smoke runner writes:

```text
.local/report.json
reports/FEISHU_SMOKE_YYYYMMDD.md
reports/feishu-real-smoke-terminal.log
```

The terminal log is ignored and must not be committed after real runs.

## 6. Generate Workflow Report JSON Manually

```bash
gitlink-cli workflow +repo-report \
  --owner "$GITLINK_OWNER" \
  --repo "$GITLINK_REPO" \
  --format json > report.json
```

Windows PowerShell redirection may produce UTF-16 with BOM. The Feishu workflow JSON reader supports UTF-8 and UTF-16 BOM inputs.

## 7. Preview Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

## 8. Send Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Requires:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

## 9. Render Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

## 10. Send Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --send --format table
```

## 11. Generate Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --format markdown
```

## 12. Send Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
```

## 13. Generate Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --format markdown
```

## 14. Send Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
```

## 15. Generate Bitable-Ready Records

```bash
gitlink-cli feishu +bitable-schema --tables reports,issues,prs,contributors,tasks --format markdown
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## 16. Preview Bitable Sync

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,contributors,tasks \
  --format table
```

## 17. Execute Bitable Sync

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,contributors,tasks \
  --send \
  --format table
```

Requires:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_REPORT_TABLE_ID
FEISHU_ISSUE_TABLE_ID
FEISHU_PR_TABLE_ID
FEISHU_CONTRIBUTOR_TABLE_ID optional
FEISHU_TASK_TABLE_ID optional
```

## 18. Preview DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --format markdown
```

## 19. Execute DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --send \
  --format table
```

## 20. Preview Feishu Tasks

```bash
gitlink-cli feishu +task-preview --from-workflow-json report.json --format markdown
```

## 21. Create Feishu Tasks

```bash
gitlink-cli feishu +task-create --from-workflow-json report.json --send --format table
```

Requires:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_TASK_PROJECT_ID optional
FEISHU_TASK_SECTION_ID optional
```

## 22. Image Evidence

Image evidence is deferred for this round. Do not add screenshots or image files
to the upload.

## 23. Run Go Tests

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

## 24. Capture Evidence

Capture terminal logs and command output only.

Do not capture raw secrets. Redact webhook URLs, app secrets, app tokens, table IDs, Wiki node tokens, folder tokens, GitLink tokens, tenant tokens, open IDs, and union IDs.
