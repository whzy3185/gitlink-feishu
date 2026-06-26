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

## 3. Configure GitLink Test Repository Manually If Needed

```powershell
$env:GITLINK_OWNER="OWNER"
$env:GITLINK_REPO="REPO"
```

If the current workflow command cannot filter specific PR IDs, keep the PR IDs in the smoke report:

```powershell
$env:GITLINK_TEST_PR_IDS="1,2,3"
```

## 4. Run Scripted Smoke Tests

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

## 5. Generate Workflow Report JSON Manually

```bash
gitlink-cli workflow +repo-report \
  --owner "$GITLINK_OWNER" \
  --repo "$GITLINK_REPO" \
  --format json > report.json
```

Windows PowerShell redirection may produce UTF-16 with BOM. The Feishu workflow JSON reader supports UTF-8 and UTF-16 BOM inputs.

## 6. Preview Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

## 7. Send Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Requires:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

## 8. Render Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

## 9. Send Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --send --format table
```

## 10. Generate Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --format markdown
```

## 11. Send Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
```

## 12. Generate Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --format markdown
```

## 13. Send Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
```

## 14. Generate Bitable-Ready Records

```bash
gitlink-cli feishu +bitable-schema --tables reports,issues,prs,contributors,tasks --format markdown
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## 15. Preview Bitable Sync

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,contributors,tasks \
  --format table
```

## 16. Execute Bitable Sync

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

## 17. Preview DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --format markdown
```

## 18. Execute DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --send \
  --format table
```

## 19. Preview Feishu Tasks

```bash
gitlink-cli feishu +task-preview --from-workflow-json report.json --format markdown
```

## 20. Create Feishu Tasks

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

## 21. Image Evidence

Image evidence is deferred for this round. Do not add screenshots or image files
to the upload.

## 22. Run Go Tests

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

## 23. Capture Evidence

Capture terminal logs and command output only.

Do not capture raw secrets. Redact webhook URLs, app secrets, app tokens, table IDs, Wiki node tokens, folder tokens, GitLink tokens, tenant tokens, open IDs, and union IDs.
