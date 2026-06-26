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
gitlink-cli feishu +owner-digest
gitlink-cli feishu +contributor-digest
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
gitlink-cli feishu +bitable-sync
gitlink-cli feishu +doc-export
gitlink-cli feishu +task-preview
gitlink-cli feishu +task-create
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
- Added role-aware owner digest rendering and optional custom bot sending.
- Added role-oriented contributor digest rendering and optional custom bot sending.
- Added `--doc-url` support for notification cards.
- Added experimental `feishu +doc-export` for Feishu DocX / Wiki export.
- Added self-built app tenant token acquisition.
- Added Wiki node resolution.
- Added DocX block creation client.
- Added document export preview and explicit `--send` behavior.
- Added Bitable dry-run schema generation.
- Added Bitable-ready dry-run records.
- Expanded Bitable records to `reports`, `issues`, `prs`, `contributors`, and `tasks`.
- Added experimental `feishu +bitable-sync` with `unique_key` search-before-update, create fallback, and no-delete behavior.
- Added local task candidate generation through `feishu +task-preview`.
- Added experimental `feishu +task-create` using Feishu Open Platform Task API.
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

The 2026-06-26 implementation smoke in the current shell did not have Feishu environment variables available, so real `--send` calls were not repeated in that shell. Public GitLink read smoke and local preview commands passed; see `reports/FEISHU_SMOKE_20260626.md`.

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

Additional experimental Open Platform paths added after the original smoke:

```text
bitable-sync: mock HTTP tested for tenant token, search, and create.
task-create: mock HTTP tested for tenant token and task create.
```

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
workflow JSON -> bot card / weekly report / owner digest / contributor digest / Bitable dry-run records / task preview
```

Experimental path:

```text
workflow JSON -> DocX/Wiki export / Bitable sync / Task create through self-built app OpenAPI
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
Feishu approval creation
callback server
button callbacks
GitLink remote writes
GitLink comments
Issue closure
code merge actions
direct GitLink webhook creation
Feishu Base/table/view creation
document permission modification
```

Note: experimental `doc-export`, `bitable-sync`, and `task-create` can attempt Open Platform writes when explicitly invoked with `--send`, but they remain outside the stable custom-bot export path.

## Next Engineering Step

For stable delivery, continue validating custom bot delivery, weekly reports, and Bitable dry-run output.

For experimental DocX/Wiki export, complete the Feishu document permission setup and rerun:

```text
gitlink-cli feishu +doc-export --from-workflow-json report.json --wiki-url <wiki_url> --send --format table
```
