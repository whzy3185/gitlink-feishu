# Feishu Smoke Evidence

Date: 2026-06-27

Branch:

```text
feat/feishu-export-clean
```

Base commit:

```text
d7812df1af49519f9eb84def218bd3d5a9fdf02f
```

## Evidence Files

| Evidence | Expected file | Status | Notes |
| --- | --- | --- | --- |
| Custom bot notify card | `reports/images/feishu-card-notify-redacted.png` | not captured | computer-use initialization failed |
| Owner digest card | `reports/images/feishu-owner-digest-redacted.png` | not captured | corrected card was sent successfully |
| DocX append result | `reports/images/feishu-docx-append-redacted.png` | not captured | corrected 11-block append passed |
| Bitable sync result | `reports/images/feishu-bitable-sync-redacted.png` | not captured | real upsert passed |
| Task create result | `reports/images/feishu-task-create-redacted.png` | not captured | historical result retained; creation was not repeated |
| Diagnostics output | `reports/images/feishu-diagnostics-terminal-redacted.png` | not captured | local and remote checks passed |

No placeholder or fabricated image file is committed.

## Text Evidence

```text
reports/FEISHU_SMOKE_20260626.md
reports/FEISHU_SMOKE_20260627.md
reports/FEISHU_PERMISSION_MATRIX.md
reports/FEISHU_API_COLLECTION_CHECKLIST_20260626.md
docs/FEISHU_OPENAPI_INVENTORY.md
docs/FEISHU_PR_ACTIVITY_STRATEGY.md
```

## Redaction Checklist

```text
[x] No webhook URL committed
[x] No app secret committed
[x] No tenant_access_token committed
[x] No document token committed
[x] No Base app token committed
[x] No table ID committed
[x] No task ID committed
[x] No open_id / union_id committed
[x] No personal account credential committed
[x] No unredacted screenshot committed
```

## Capture Rule

Screenshots may be added only after the Windows automation connection works and
each image is reviewed for resource IDs, personal identities, and unrelated
conversation content. Until then, this document records the missing visual
evidence explicitly rather than presenting a fake pass.
