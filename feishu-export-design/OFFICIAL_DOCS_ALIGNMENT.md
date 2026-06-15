# Feishu Official Docs Alignment

## Sources Checked

- Custom bot usage guide: https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- Send message cards with custom bot: https://open.feishu.cn/document/feishu-cards/quick-start/send-message-cards-with-custom-bot?lang=zh-CN
- Custom app tenant access token: https://open.feishu.cn/document/server-docs/authentication-management/access-token/tenant_access_token_internal?lang=zh-CN
- Send message API: https://open.feishu.cn/document/server-docs/im-v1/message/create?lang=zh-CN
- Create DocX document: https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document/create
- Create DocX blocks: https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document-block/create?lang=zh-CN
- Bitable create record: https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/create?lang=zh-CN
- Bitable batch create records: https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/batch_create?lang=zh-CN
- Docs token FAQ: https://open.feishu.cn/document/faq/trouble-shooting/how-to-get-docs-tokens
- Docs permission FAQ: https://open.feishu.cn/document/server-docs/docs/faq?lang=zh-CN

## Important Product Boundary

The BotBuilder shutdown notice does not affect this design if the implementation uses:

```text
Feishu Open Platform custom bot webhooks
Feishu Open Platform custom app APIs
Feishu Docs / Bitable OpenAPI
```

Do not integrate:

```text
botbuilder.feishu.cn
Feishu Robot Assistant workflows
```

## Correct Integration Modes

### Mode A: Custom Group Bot Webhook

Use this for the first working proof.

Required inputs:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

Capabilities:

```text
Send one-way group notifications.
Send interactive card JSON to a group.
No tenant token.
No app_id/app_secret.
No user, tenant, document, or Bitable data access.
```

Fit in this project:

```text
feishu +bot-test
feishu +notify
feishu +weekly-report --send
```

### Mode B: Open Platform Custom App

Use this for real document and Bitable operations.

Required inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
```

Token flow:

```text
POST /open-apis/auth/v3/tenant_access_token/internal
request: app_id + app_secret
response: tenant_access_token, expire
```

Required implementation:

```text
Token client
token cache with expiry
redacted errors
permission diagnostics
mocked HTTP tests
```

Fit in this project:

```text
Phase 2: feishu +doc-export
Phase 3: feishu +bitable-sync or +bitable-upsert
Optional: app bot message send through im/v1/messages
```

### Mode C: Low-Code Alternatives

Multidimensional table workflows, Aily, and AnyCross are valid migration choices for BotBuilder users, but they are not a good first implementation target inside `gitlink-cli`.

Use them as documentation references only.

## Recommended Product Flow

The practical GitLink-to-Feishu workflow should be:

```text
1. gitlink-cli workflow +repo-report --format json > report.json
2. gitlink-cli feishu +weekly-report --from-workflow-json report.json --format markdown
3. gitlink-cli feishu +doc-export --from-workflow-json report.json --folder-token <folder_token> --send
4. gitlink-cli feishu +notify --from-workflow-json report.json --doc-url <doc_url> --send
5. gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
6. Later: gitlink-cli feishu +bitable-sync --from-workflow-json report.json --send
```

Key point:

```text
Card = notification.
Doc = collaboration artifact.
Bitable = structured tracking data.
```

The earlier design covered card and Bitable dry-run, but missed the document artifact.

## Doc Export Requirements

Add a later `feishu +doc-export` command.

Inputs:

```text
--from-workflow-json report.json
--folder-token <folder_token>
--document-id <document_id> optional later
--wiki-url <wiki_url> optional later
--wiki-node-token <node_token> optional later
--title <title>
--send
```

Environment:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
```

Behavior:

```text
Default preview only.
--send creates or updates a Feishu DocX document.
Create document first.
Then create blocks under the document root block.
Return document_id and URL.
No document operation without --send.
```

Permission notes:

```text
The app must have required DocX/Drive application scopes.
The target folder or document must grant the app document permission.
folder_token/document_id/app_token must be read from URL or OpenAPI.
```

## Knowledge Base / Wiki Fit

Knowledge Base pages are useful for project showcase and reference material.

The supplied project page shape:

```text
https://<tenant>.feishu.cn/wiki/<node_token>
```

Official API flow:

```text
1. Get tenant_access_token with app_id/app_secret.
2. Resolve wiki node token with Wiki API.
3. If obj_type is docx, use obj_token as the DocX document target.
4. Export or append report blocks with DocX block APIs.
5. Send a Feishu bot card with the wiki/doc URL as the collaboration entry.
```

Design impact:

```text
Add wiki-url/wiki-node-token support to doc-export.
Add --doc-url to notify/weekly-report card commands.
Keep Wiki operations behind --send.
Do not edit knowledge base permissions automatically.
```

This makes the project output more suitable for display:

```text
Knowledge Base page = project homepage / reference index.
DocX report blocks = generated workflow report.
Bot card = notification and entry link.
Bitable records = structured data for later dashboards.
```

## Bitable Real Write Requirements

Keep current `+bitable-schema` and `+bitable-records` as dry-run commands.

Only add real write after the app auth layer exists.

Required inputs:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_REPORT_TABLE_ID
FEISHU_ISSUE_TABLE_ID
FEISHU_PR_TABLE_ID
FEISHU_CONTRIBUTOR_TABLE_ID optional
```

Required behavior:

```text
Fetch tenant_access_token.
Validate table IDs.
Create records or batch create records.
For update/upsert, search existing records first.
Do not call Bitable OpenAPI unless --send is explicit.
```

## Design Verdict

Current design is reasonable as a first safe MVP:

```text
custom bot send
workflow JSON local input
weekly report markdown
Bitable schema/records dry-run
mock tests
```

But it is incomplete for a "Feishu collaboration export" feature because it does not create or update Feishu Docs.

Required design adjustment:

```text
Add doc-export with DocX/Wiki support as the first Open Platform custom-app integration.
Keep real Bitable writes after doc-export.
Keep custom bot as the low-friction smoke test path.
```
