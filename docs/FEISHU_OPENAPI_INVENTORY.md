# Feishu OpenAPI Inventory for GitLink CLI

Date: 2026-06-26

This document maps the Feishu / Lark Open Platform APIs collected for the
`gitlink-cli feishu` integration to the current command surface and the next
implementation gaps.

The inventory is intentionally split into three surfaces:

```text
Layer 1: Stable custom bot export
Layer 2: Experimental Open Platform validation
Layer 3: Future callback-based GitLink action gateway
```

No implemented command in this branch performs GitLink write operations.

## Source Index

Official Feishu / Lark references used for this inventory:

```text
Custom bot:
https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
https://open.feishu.cn/document/feishu-cards/quick-start/send-message-cards-with-custom-bot?lang=zh-CN

App authentication:
https://open.feishu.cn/document/server-docs/authentication-management/access-token/tenant_access_token_internal?lang=zh-CN

IM app bot:
https://open.feishu.cn/document/server-docs/im-v1/message/create?lang=zh-CN

DocX / Wiki:
https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document/create
https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document-block/create?lang=zh-CN
https://open.feishu.cn/document/server-docs/docs/wiki-v2/space/get_node

Base / Bitable:
https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/search
https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/create?lang=zh-CN
https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/update

Task:
https://open.feishu.cn/document/task-v2/task/create?lang=zh-CN

lark-cli:
https://github.com/larksuite/cli
https://open.larksuite.com/document/mcp_open_tools/feishu-cli-let-ai-actually-do-your-work-in-feishu
https://www.feishu.cn/feishu-cli
```

## Current API Usage Summary

| Area | Endpoint / API family | Current command | Status | Write target | Notes |
| --- | --- | --- | --- | --- | --- |
| Custom bot webhook | `POST /open-apis/bot/v2/hook/{token}` | `+bot-test`, `+notify`, `+weekly-report`, `+owner-digest`, `+contributor-digest` | Implemented stable | Feishu chat message | Requires `--send`; preview by default |
| Custom bot signature | timestamp + HMAC-SHA256 signing secret | same as above | Implemented stable | Request signature only | `FEISHU_WEBHOOK_SECRET` optional |
| Tenant token | `POST /auth/v3/tenant_access_token/internal` | `+doc-export`, `+bitable-sync`, `+task-create` | Implemented experimental | Tenant token | No token cache yet |
| Wiki node resolution | `GET /wiki/v2/spaces/get_node?token=...` | `+doc-export` | Implemented experimental | Wiki metadata read | Used to resolve Wiki node to DocX object token |
| DocX create | `POST /docx/v1/documents` | `+doc-export` | Implemented experimental | New DocX document | Requires folder/resource permission |
| DocX append blocks | `POST /docx/v1/documents/{document_id}/blocks/{block_id}/children` | `+doc-export` | Implemented experimental | DocX block tree | Real write can fail on scope or document permission |
| Bitable search | `POST /bitable/v1/apps/{app_token}/tables/{table_id}/records/search` | `+bitable-sync` | Implemented experimental | Existing Base table | Searches by `unique_key` field |
| Bitable create record | `POST /bitable/v1/apps/{app_token}/tables/{table_id}/records` | `+bitable-sync` | Implemented experimental | Existing Base table | No table/field/view creation |
| Bitable update record | `PUT /bitable/v1/apps/{app_token}/tables/{table_id}/records/{record_id}` | `+bitable-sync` | Implemented experimental | Existing Base table | Never deletes records |
| Task create | `POST /task/v2/tasks` | `+task-create` | Implemented experimental | Feishu task | Project/section placement is not mapped into request body yet |
| IM app bot send | `POST /im/v1/messages?receive_id_type=...` | none | Planned | App-bot message | Needed for direct/group app bot sends beyond custom bot |
| Card callbacks | Interactive card callback / event subscription | none | Future | Callback server | Required before Feishu-triggered GitLink actions |
| User identity | open_id / union_id / user lookup | none | Future | Identity mapping | Required before personalized contributor routing |

## Layer 1: Stable Custom Bot Export

### Implemented APIs

#### Custom Bot Webhook

Current commands:

```text
+bot-test
+notify
+weekly-report
+owner-digest
+contributor-digest
```

Inputs:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
--send required for real delivery
--dry-run conflicts with --send
```

Current behavior:

```text
Builds Feishu interactive card payloads.
Signs webhook requests when a secret is configured.
Prints local previews by default.
Redacts webhook URLs and secrets from normal output.
Only includes navigation buttons.
```

Limits:

```text
No personalized routing.
No app-level chat_id.
No callback execution.
No Feishu resource write.
No GitLink resource write.
```

Next hardening:

```text
Add more card color/stage variants for PR review state.
Add compact owner card and detailed digest variants.
Add screenshot-backed smoke evidence after real webhook env is restored.
```

## Layer 2: Experimental Open Platform Validation

### App Authentication

Endpoint:

```text
POST /auth/v3/tenant_access_token/internal
```

Current commands:

```text
+doc-export
+bitable-sync
+task-create
```

Inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
```

Current behavior:

```text
Fetches tenant_access_token before Open Platform writes.
Does not persist or cache tenant_access_token.
Does not print the raw token.
```

Next hardening:

```text
Add +app-check.
Cache token in memory during one command execution only.
Add scope diagnostics where official scope names are confirmed.
```

### DocX / Wiki

Endpoints:

```text
GET /wiki/v2/spaces/get_node?token=...
POST /docx/v1/documents
POST /docx/v1/documents/{document_id}/blocks/{parent_block_id}/children
```

Current command:

```text
+doc-export
```

Inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_WIKI_URL or FEISHU_WIKI_NODE_TOKEN
FEISHU_FOLDER_TOKEN optional
FEISHU_DOCUMENT_ID optional
--send required for real write
```

Current behavior:

```text
Preview renders workflow report content locally.
Wiki URL can be parsed into a node token.
Wiki node can be resolved to a DocX object token.
Existing DocX / Wiki target is appended when allowed.
Folder token can be used to create a new DocX when allowed.
Diagnostics preserve Feishu errors without leaking tokens.
```

Known blockers:

```text
The app must have approved document scopes.
The app must be able to edit the target Wiki / DocX page.
For folder creation, the app must be able to create files in the target folder.
The command does not modify document permissions.
```

Local UI observation:

```text
The Feishu desktop app currently shows a cloud-doc permission request flow.
This supports the current design decision that resource-level document access
must be handled by the owner/admin outside the CLI.
```

Next hardening:

```text
Add +app-check diagnostics for DocX/Wiki scopes.
Add clearer output for target type: wiki node, existing doc, folder creation.
Add optional markdown-only export for manual paste into Feishu Docs.
```

### Base / Bitable

Endpoints:

```text
POST /bitable/v1/apps/{app_token}/tables/{table_id}/records/search
POST /bitable/v1/apps/{app_token}/tables/{table_id}/records
PUT /bitable/v1/apps/{app_token}/tables/{table_id}/records/{record_id}
```

Current commands:

```text
+bitable-schema
+bitable-records
+bitable-sync
```

Inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_REPORT_TABLE_ID
FEISHU_ISSUE_TABLE_ID
FEISHU_PR_TABLE_ID
FEISHU_CONTRIBUTOR_TABLE_ID optional
FEISHU_TASK_TABLE_ID optional
--send required for real sync
```

Current tables:

```text
reports
issues
prs
contributors
tasks
```

Current behavior:

```text
+bitable-schema outputs a dry-run schema.
+bitable-records outputs summary-oriented local records.
+bitable-sync previews by default.
+bitable-sync --send searches by unique_key, updates if found, creates if missing.
If search fails, the command falls back to create-only for that record.
Slice values are flattened into newline-separated text before OpenAPI writes.
No records are deleted.
```

Local test-enterprise validation on 2026-06-26:

```text
The provided Base links resolved to one Base and one table with multiple views.
The table initially contained only default fields.
The missing fields were created manually through OpenAPI for validation.
+bitable-sync then successfully created and updated reports, issues, prs,
contributors, and task records in the test table.
```

Known blockers:

```text
The Base app must already exist.
The target tables must already exist.
The target tables must contain a compatible unique_key field.
Field types must be compatible with generated record values.
No Bitable view creation is implemented.
No table/field creation is implemented.
Current records are summary buckets, not full row-level PR/Issue/CI records.
```

Next hardening:

```text
Add table/field validation before writes.
Add row-level records for PRs, Issues, CI runs, milestones, releases, and audits.
Add optional Bitable view planning output for Kanban, Gantt, Calendar, Gallery, Form, and Dashboard.
Keep real view creation as a separate permissioned task.
```

### Task

Endpoint:

```text
POST /task/v2/tasks
```

Current commands:

```text
+task-preview
+task-create
```

Inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_TASK_PROJECT_ID optional
FEISHU_TASK_SECTION_ID optional
--send required for real creation
```

Current behavior:

```text
+task-preview generates local task candidates.
+task-create previews by default and creates tasks only with --send.
Task candidates are derived from workflow recommendations, high-risk issues,
missing-info issues, high-risk PRs, and review-focus items.
Local dedupe uses stable unique_key generation.
```

Local test-enterprise validation on 2026-06-26:

```text
+task-preview generated 7 task candidates from the Gitlink/gitlink-cli report.
+task-create --send created 7 Feishu tasks.
The task result table now shows per-task create status and redacted task IDs.
```

Known blockers:

```text
Feishu-side dedupe/search is not implemented.
Task project and section IDs are collected and redacted in output, but the
current OpenAPI request body only sends summary and description. Project/section
placement must be wired only after the official request fields and tenant
behavior are confirmed in the test enterprise.
```

Next hardening:

```text
Confirm official Task project/section placement fields.
Add Feishu-side dedupe or external unique_key linking when a stable API path exists.
Add scope diagnostics through +app-check.
```

## i18n Validation

Current Feishu commands can consume a Chinese workflow report and render
localized Feishu output:

```text
workflow +repo-report --lang zh-CN
feishu +notify --lang zh-CN
feishu +owner-digest --lang zh-CN
feishu +contributor-digest --lang zh-CN
feishu +doc-export --lang zh-CN
feishu +task-preview --lang zh-CN
feishu +task-create --lang zh-CN
```

Validated output surfaces:

```text
card field labels
owner/contributor digest headings
common workflow recommendations
DocX block headings
task candidate titles and descriptions
table/markdown preview labels
```

Repository-wide i18n check still reports that `internal/i18n/locales/en-US.json`
needs formatting. That is outside the Feishu module and was left untouched to
avoid unrelated locale-file churn.

## Layer 3: Future Callback-Based GitLink Action Gateway

No callback server or GitLink write action is implemented in this branch.

Planned Feishu API families:

```text
Card callback verification
Event subscription / long connection or callback endpoint
IM message update or follow-up message
User identity lookup: open_id / union_id / email
Chat membership or chat metadata where needed
```

Planned GitLink command families:

```text
Read:
workflow +repo-report
issue +list / +view
pr +list / +view / +files / +diff / +reviews
ci +builds / +logs
pipeline +list / +view / +runs / +results

Low-risk future writes:
issue +comment
pr +comment
pr +review

High-risk future writes disabled by default:
pr +merge
issue +close
member +add / +remove / +role
webhook +create / +update / +delete
branch or release deletion
```

Required gateway controls:

```text
Verify Feishu callback signature.
Resolve repo binding.
Map Feishu identity to GitLink identity.
Check GitLink permission.
Generate GitLink dry-run preview.
Require explicit confirmation.
Write audit logs.
Disable high-risk actions by default.
Never execute GitLink writes from a custom bot webhook.
```

## GitLink Data Source Inventory

Current Feishu commands primarily consume:

```text
workflow +repo-report --format json
```

Current source properties:

```text
Read-only.
Works with local JSON fixture or remote GitLink report generation.
Does not require the Feishu module to know a GitLink token.
Provides summary-level issue, PR, contributor, recommendation, and health fields.
```

Required future source expansion:

```text
PR row source:
pr +list
pr +view
pr +files
pr +diff
pr +versions
pr +reviews

Issue row source:
issue +list
issue +view
issue metadata commands

CI / pipeline row source:
ci +builds
ci +logs
pipeline +runs
pipeline +results

Milestone / release source:
milestone and release commands where available

Audit source:
future action gateway audit log
```

Reason:

```text
Summary buckets are enough for cards and weekly reports.
Kanban, Gantt, Calendar, Gallery, dashboard, and personal task panels require
row-level GitLink records.
```

## Manual Setup Required From User

Stable webhook validation:

```text
1. Add a custom bot to the target Feishu test group.
2. Copy the webhook URL into FEISHU_WEBHOOK_URL.
3. Copy the signing secret into FEISHU_WEBHOOK_SECRET if signing is enabled.
4. Run +bot-test or +notify with --send.
```

DocX / Wiki validation:

```text
1. Confirm FEISHU_APP_ID and FEISHU_APP_SECRET for the self-built app.
2. Approve required DocX / Drive scopes in Feishu Open Platform.
3. Grant the app edit access to the target Wiki / DocX page, or provide a
   folder token where the app can create documents.
4. Run +doc-export first without --send, then with --send.
```

Bitable validation:

```text
1. Create or choose a Base manually.
2. Create reports, issues, prs, contributors, and tasks tables manually.
3. Add a unique_key field to every table.
4. Copy FEISHU_BASE_APP_TOKEN and each table ID into local env.
5. Grant the self-built app Base/Bitable access.
6. Run +bitable-sync without --send first, then with --send.
```

For a quick validation, multiple table env vars can point to the same test
table if that table has every required field. For a real project cockpit, prefer
separate tables or a row-level model that supports Kanban, Gantt, Calendar,
Gallery, Form, and Dashboard views without mixing incompatible record groups.

Task validation:

```text
1. Confirm Task API scopes for the self-built app.
2. Decide whether tasks should be created as plain tasks first.
3. Do not rely on project/section placement until request fields are verified.
4. Run +task-preview first, then +task-create --send.
```

GitLink real data validation:

```text
1. Set GITLINK_OWNER and GITLINK_REPO.
2. Set GITLINK_TEST_PR_IDS for smoke report reference.
3. Ensure gitlink-cli can generate workflow +repo-report JSON.
4. Do not commit GitLink tokens or account credentials.
```

## Acceptance Checklist For API Collection

```text
[x] Custom bot webhook API identified.
[x] Custom bot signing behavior mapped to current code.
[x] tenant_access_token API identified and implemented.
[x] Wiki node resolution API identified and implemented.
[x] DocX create and block append APIs identified and implemented.
[x] Bitable record search/create/update APIs identified and implemented.
[x] Task create API identified and implemented at minimal summary/description level.
[x] IM app bot send API identified as planned, not implemented.
[x] Card callback/event subscription identified as future, not implemented.
[x] User identity APIs identified as future, not implemented.
[x] GitLink read data source boundary documented.
[x] GitLink write action boundary documented as not implemented.
[x] Required local env variables documented.
[x] Resource-level permission requirements documented.
[x] Remaining user/manual setup steps documented.
```
