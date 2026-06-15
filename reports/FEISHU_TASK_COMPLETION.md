# Feishu Export Task Completion

## Branch

```text
feat/feishu-export-clean
```

## Implemented Commands

```text
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
```

## Implemented Behavior

- Added a `feishu` shortcut group.
- Added local preview by default.
- Added explicit `--send` for real Feishu custom bot delivery.
- Added `--send` and `--dry-run` conflict validation.
- Added webhook URL redaction in command output.
- Added Feishu custom bot interactive card builders.
- Added Feishu custom bot signing helper.
- Added injectable HTTP webhook client.
- Added workflow JSON input support for `RepoReportInput`, `RepoReportResult`, and envelope-like `data`.
- Added project activity card generation from workflow JSON.
- Added weekly report rendering from workflow JSON.
- Added `--doc-url` support for notification cards.
- Added Bitable dry-run schema generation.
- Added Bitable-ready dry-run records.
- Registered the new shortcut group in `shortcuts/register.go`.
- Updated shortcut registration tests.

## Feishu Smoke Checks

Custom bot send was tested against a real Feishu custom bot webhook.

Result:

```text
HTTP status: 200
Feishu code: 0
Message: success
```

Webhook output was redacted.

A second notification card was sent with a Feishu Wiki URL as the report entry link.

## Open Platform Checks

Self-built app authentication was checked with the Feishu Open Platform tenant token endpoint.

Result:

```text
tenant_access_token: acquired
expire: 7199 seconds
```

The supplied Feishu Wiki URL was resolved through Wiki OpenAPI.

Result:

```text
wiki node: resolved
object type: docx
object token: present
```

No document content was modified in this check.

## Knowledge Base Design Update

Added official-docs alignment notes:

```text
feishu-export-design/OFFICIAL_DOCS_ALIGNMENT.md
```

Design now treats Feishu Knowledge Base / Wiki pages as a project showcase and reference target:

```text
workflow JSON -> DocX/Wiki report -> bot card with doc URL -> Bitable dry-run records
```

## Tests

Commands run:

```bash
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

Result:

```text
passed
```

## Explicitly Not Implemented

```text
BotBuilder integration
Feishu Robot Assistant workflows
Feishu task creation
Feishu approval creation
callback server
button callbacks
GitLink remote writes
GitLink comments
Issue closure
code merge actions
direct GitLink webhook creation
real Bitable OpenAPI writes
Bitable create/update/upsert
Feishu Base/table/view creation
document permission modification
DocX content write
```

## Next Engineering Step

Add `feishu +doc-export` as the first self-built app integration:

```text
app_id/app_secret -> tenant_access_token -> resolve Wiki node or create DocX -> write report blocks -> return doc URL
```

