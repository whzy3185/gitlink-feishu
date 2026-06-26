# Feishu Local Testing Guide

Date: 2026-06-26

This guide verifies the layered Feishu integration without committing secrets.

## 1. Configure GitLink Test Repository

```powershell
$env:GITLINK_OWNER="OWNER"
$env:GITLINK_REPO="REPO"
```

If the current workflow command cannot filter specific PR IDs, keep the PR IDs in the smoke report:

```powershell
$env:GITLINK_TEST_PR_IDS="1,2,3"
```

## 2. Generate Workflow Report JSON

```bash
gitlink-cli workflow +repo-report \
  --owner "$GITLINK_OWNER" \
  --repo "$GITLINK_REPO" \
  --format json > report.json
```

Windows PowerShell redirection may produce UTF-16 with BOM. The Feishu workflow JSON reader supports UTF-8 and UTF-16 BOM inputs.

## 3. Preview Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

## 4. Send Feishu Notify Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

Requires:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

## 5. Render Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
```

## 6. Send Weekly Report

```bash
gitlink-cli feishu +weekly-report --from-workflow-json report.json --send --format table
```

## 7. Generate Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --format markdown
```

## 8. Send Owner Digest

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
```

## 9. Generate Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --format markdown
```

## 10. Send Contributor Digest

```bash
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
```

## 11. Generate Bitable-Ready Records

```bash
gitlink-cli feishu +bitable-schema --tables reports,issues,prs,contributors,tasks --format markdown
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## 12. Preview Bitable Sync

```bash
gitlink-cli feishu +bitable-sync \
  --from-workflow-json report.json \
  --tables reports,issues,prs,contributors,tasks \
  --format table
```

## 13. Execute Bitable Sync

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

## 14. Preview DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --format markdown
```

## 15. Execute DocX / Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "$FEISHU_WIKI_URL" \
  --send \
  --format table
```

## 16. Preview Feishu Tasks

```bash
gitlink-cli feishu +task-preview --from-workflow-json report.json --format markdown
```

## 17. Create Feishu Tasks

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

## 18. Run Go Tests

```bash
gofmt -w shortcuts/feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

## 19. Capture Evidence

Capture terminal logs and screenshots listed in `docs/PR_VISUAL_GUIDE.md`.

Do not capture raw secrets. Redact webhook URLs, app secrets, app tokens, table IDs, Wiki node tokens, folder tokens, GitLink tokens, tenant tokens, open IDs, and union IDs.
