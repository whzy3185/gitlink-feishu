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
- Added experimental `feishu +doc-export` for Feishu DocX / Wiki export.
- Added self-built app tenant token acquisition.
- Added Wiki node resolution.
- Added DocX block creation client.
- Added document export preview and explicit `--send` behavior.
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

The first real DocX block write attempt reached the DocX block endpoint but Feishu rejected the write:

```text
HTTP status: 403
Feishu code: 1770032
Message: forBidden
```

Interpretation:

```text
The app credentials are valid and the Wiki node is readable, but the app does not currently have write permission on the target Wiki-backed DocX page or lacks the required document scope approval.
```

The command now reports a permission hint for this case.

## Knowledge Base Design Update

Added official-docs alignment notes:

```text
feishu-export-design/OFFICIAL_DOCS_ALIGNMENT.md
```

Design now treats Feishu Knowledge Base / Wiki pages as a project showcase and reference target:

```text
workflow JSON -> DocX/Wiki report -> bot card with doc URL -> Bitable dry-run records
```

After scope review, DocX / Wiki export is explicitly experimental and not part of the stable clean workflow.

Stable path:

```text
workflow JSON -> bot card / weekly report / Bitable dry-run records
```

Experimental path:

```text
workflow JSON -> DocX/Wiki export through self-built app OpenAPI
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

Note: experimental `doc-export` can attempt DocX block writes when explicitly invoked with `--send`, but it remains outside the stable clean export path.

## Next Engineering Step

For stable delivery, continue validating custom bot delivery, weekly reports, and Bitable dry-run output.

For experimental DocX/Wiki export, complete the Feishu document permission setup and rerun:

```text
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url <wiki_url> --send --format table
```
