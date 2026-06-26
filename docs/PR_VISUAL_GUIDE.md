# PR Visual Guide

Date: 2026-06-26

This file lists the manual screenshots to capture after local and real smoke testing.
The 2026-06-26 smoke run successfully delivered Feishu cards, appended DocX
content, synced Bitable records, and created Feishu tasks in the test
enterprise. Screenshots still need to be captured manually from the UI.

Do not fabricate screenshots. If a capability is not available in the test enterprise, keep the placeholder and record the failure in `reports/FEISHU_SMOKE_20260626.md`.

Use the helper to check current screenshot status:

```powershell
.\scripts\feishu-gitlink-screenshot-check.ps1
```

| Screenshot | Expected path | Capture note |
| --- | --- | --- |
| Feishu bot card in test group | `docs/images/feishu-bot-card.png` | Capture after `+bot-test --send` or `+notify --send` |
| Weekly report card | `docs/images/feishu-weekly-report.png` | Capture after `+weekly-report --send` |
| Owner digest card | `docs/images/feishu-owner-digest.png` | Capture after `+owner-digest --send` |
| Contributor digest card | `docs/images/feishu-contributor-digest.png` | Capture after `+contributor-digest --send` |
| Bitable records preview | `docs/images/feishu-bitable-preview.png` | Capture terminal output or JSON preview |
| Bitable Base after sync | `docs/images/feishu-bitable-sync.png` | Real sync succeeded in the test Base; capture the updated table or target view |
| DocX / Wiki report | `docs/images/feishu-docx-wiki.png` | Real DocX append succeeded; capture the appended report blocks |
| Feishu task list | `docs/images/feishu-task-create.png` | Real task creation succeeded; capture the created task list and redact IDs if visible |
| Terminal smoke test summary | `docs/images/feishu-smoke-terminal.png` | Redact IDs and tokens |
| Redacted env check | `docs/images/feishu-env-redacted.png` | Show presence/absence only |

Suggested capture commands:

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format table
gitlink-cli feishu +notify --from-workflow-json report.zh-CN.json --lang zh-CN --send --format table
```

Manual redaction checklist:

```text
webhook URL
app secret
tenant_access_token
Base app token
table IDs
Wiki node token
folder token
GitLink token
open_id / union_id
```
