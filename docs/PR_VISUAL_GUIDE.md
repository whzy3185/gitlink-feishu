# PR Visual Guide

Date: 2026-06-26

This file lists the manual screenshots to capture after local and real smoke testing.

Do not fabricate screenshots. If a capability is not available in the test enterprise, keep the placeholder and record the failure in `reports/FEISHU_SMOKE_20260626.md`.

| Screenshot | Expected path | Capture note |
| --- | --- | --- |
| Feishu bot card in test group | `docs/images/feishu-bot-card.png` | Capture after `+bot-test --send` or `+notify --send` |
| Weekly report card | `docs/images/feishu-weekly-report.png` | Capture after `+weekly-report --send` |
| Owner digest card | `docs/images/feishu-owner-digest.png` | Capture after `+owner-digest --send` |
| Contributor digest card | `docs/images/feishu-contributor-digest.png` | Capture after `+contributor-digest --send` |
| Bitable records preview | `docs/images/feishu-bitable-preview.png` | Capture terminal output or JSON preview |
| Bitable Base after sync | `docs/images/feishu-bitable-sync.png` | Capture only if real sync succeeds |
| DocX / Wiki report | `docs/images/feishu-docx-wiki.png` | Capture only if real document write succeeds |
| Feishu task list | `docs/images/feishu-task-create.png` | Capture only if real task creation succeeds |
| Terminal smoke test summary | `docs/images/feishu-smoke-terminal.png` | Redact IDs and tokens |
| Redacted env check | `docs/images/feishu-env-redacted.png` | Show presence/absence only |

Suggested capture commands:

```bash
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send --format table
gitlink-cli feishu +contributor-digest --from-workflow-json report.json --send --format table
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format table
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
