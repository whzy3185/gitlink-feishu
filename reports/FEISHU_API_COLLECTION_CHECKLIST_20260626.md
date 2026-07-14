# Feishu API Collection Checklist

Date: 2026-06-26

Branch:

```text
feat/feishu-export-clean
```

Current commit at collection time:

```text
73da46c143b37cb2b26e9e624b8c39963ad52d77
```

## Current Local State

```text
Feishu desktop/web state: test account is logged in.
Local env file: .local/feishu-gitlink.env.ps1 is configured and ignored.
Stable previews: available from .local/report.json and .local/report.zh-CN.json.
Real Feishu sends: passed through custom bot webhook.
Real DocX append: passed through self-built app OpenAPI.
Real Bitable sync: passed after target table fields were created.
Split Bitable sync: passed with five separate tables.
Real Task create: passed; project/section placement remains unmapped.
Read/check diagnostics: passed for app, DocX target, Bitable tables, and Task credentials.
GitLink write operations: not implemented and not tested.
Test enterprise permissions: intentionally broad for validation; production should use minimum scopes.
```

## API Collection Status

| Item | Status | Evidence | Next action |
| --- | --- | --- | --- |
| Custom bot webhook | Complete and real-tested | `shortcuts/feishu/client.go`, `sign.go`, `card.go` | Image evidence deferred |
| Custom bot signing | Complete and real-tested | `SignCustomBotRequest` unit test plus signed bot smoke | Keep secrets redacted |
| tenant_access_token | Complete and real-tested | `OpenAPIClient.TenantAccessToken`, `+app-check --remote` | Add richer scope hints later |
| Wiki node resolution | Complete | `OpenAPIClient.GetWikiNode` | Still depends on target Wiki node permission |
| DocX create | Complete | `OpenAPIClient.CreateDocument` | Requires folder permission when creating new docs |
| DocX block append | Complete and real-tested | `OpenAPIClient.CreateBlocks` | App must have target DocX edit permission |
| Bitable search | Complete and real-tested | `SearchBitableRecord` | Requires `unique_key` field |
| Bitable create | Complete and real-tested | `CreateBitableRecord` | Requires existing table and compatible fields; split-table write passed |
| Bitable update | Complete and real-tested | `UpdateBitableRecord` | Never deletes records |
| Task create | Complete at minimal level and real-tested | `CreateTask` sends summary and description | Confirm project/section request fields |
| IM app bot message | Planned | Official API collected | Not needed for stable webhook path |
| Card callbacks | Future | Official capability identified | Requires server, signature validation, identity mapping |
| User identity mapping | Future | Required for personalized contributor routing | Not implemented in this branch |
| GitLink write actions | Explicitly out of scope | Capability boundary docs | Do not implement in this branch |

## Command Checklist

| Command | Layer | Current status | Real side effect? | Needs user setup? |
| --- | --- | --- | --- | --- |
| `feishu +bot-test` | Stable | Implemented | Only with `--send` | `FEISHU_WEBHOOK_URL` |
| `feishu +notify` | Stable | Implemented | Only with `--send` | `FEISHU_WEBHOOK_URL` |
| `feishu +weekly-report` | Stable | Implemented | Only with `--send` | `FEISHU_WEBHOOK_URL` |
| `feishu +owner-digest` | Stable | Implemented | Only with `--send` | `FEISHU_WEBHOOK_URL` |
| `feishu +contributor-digest` | Stable | Implemented | Only with `--send` | `FEISHU_WEBHOOK_URL` |
| `feishu +app-check` | Stable diagnostics | Implemented | No writes; optional read/check with `--remote` | App credentials for remote token check |
| `feishu +doc-check` | Stable diagnostics | Implemented | No writes; optional Wiki read with `--remote` | App credentials and DocX/Wiki target |
| `feishu +bitable-check` | Stable diagnostics | Implemented | No writes; optional Bitable search with `--remote` | Base app token, table IDs, `unique_key` |
| `feishu +task-check` | Stable diagnostics | Implemented | No writes; optional token check with `--remote` | App credentials |
| `feishu +bitable-schema` | Stable dry-run | Implemented | No | No |
| `feishu +bitable-records` | Stable dry-run | Implemented | No | No |
| `feishu +task-preview` | Stable dry-run | Implemented | No | No |
| `feishu +doc-export` | Experimental | Implemented and real-tested | Only with `--send` | App scopes and document/folder permission |
| `feishu +bitable-sync` | Experimental | Implemented and real-tested | Only with `--send` | Base app token, table IDs, fields, scopes |
| `feishu +task-create` | Experimental | Implemented and real-tested at minimal level | Only with `--send` | Task scopes; project/section placement pending |

## What Is Complete

```text
1. Feishu API families are mapped to current commands.
2. Current code endpoints are inventoried.
3. Stable custom bot boundary is clear.
4. Experimental Open Platform boundary is clear.
5. GitLink write action boundary is clear.
6. User-required environment variables are documented.
7. Resource-level permission requirements are documented.
8. Task project/section limitation is explicitly called out.
9. Feishu configuration diagnostics are available before running --send writes.
```

## What Still Needs User Action

These remain manual or owner-side tasks and should not be committed to the
repository.

```text
1. Defer Feishu UI screenshots and image evidence for this upload.
2. Keep the split-table validation as text evidence.
3. Decide later whether the final demonstration should use split tables only or
   also keep the one-table/multiple-view proof as background evidence.
4. Decide whether `+bitable-sync` should stay experimental or be narrowed to
   dry-run-only for upstream review.
5. Confirm Feishu Task project/section request fields before placing tasks in
   a specific project or section.
6. Keep all real app credentials, webhook URLs, table IDs, and tokens in local
   env only.
```

## Commands To Run After User Setup

Preview first:

```powershell
.\scripts\feishu-gitlink-env-check.ps1 -Layer all

go run . feishu +notify --from-workflow-json .local\report.json --format table
go run . feishu +owner-digest --from-workflow-json .local\report.json --format table
go run . feishu +contributor-digest --from-workflow-json .local\report.json --format table
go run . feishu +bitable-records --from-workflow-json .local\report.json --format json
go run . feishu +bitable-sync --from-workflow-json .local\report.json --format table
go run . feishu +doc-export --from-workflow-json .local\report.json --format table
go run . feishu +task-preview --from-workflow-json .local\report.json --format markdown
```

Real sends/writes only after preview is correct:

```powershell
go run . feishu +notify --from-workflow-json .local\report.json --send --format table
go run . feishu +weekly-report --from-workflow-json .local\report.json --send --format table
go run . feishu +owner-digest --from-workflow-json .local\report.json --send --format table
go run . feishu +contributor-digest --from-workflow-json .local\report.json --send --format table
go run . feishu +doc-export --from-workflow-json .local\report.json --send --format table
go run . feishu +bitable-sync --from-workflow-json .local\report.json --send --format table
go run . feishu +task-create --from-workflow-json .local\report.json --send --format table
```

Test suite:

```powershell
gofmt -w shortcuts\feishu
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
go test ./...
```

## Current Blockers

```text
Task project/section placement needs official request-field confirmation.
Current Base output is summary-oriented and not yet row-level project cockpit data.
Image evidence is deferred and is not part of this upload.
```

## Verification Run

Executed on 2026-06-26 after the API inventory update:

| Check | Result | Notes |
| --- | --- | --- |
| Computer-use Feishu desktop read-only check | Pass | Feishu test account visible |
| `go run . feishu --help` | Pass | Expected stable and experimental commands are registered |
| Env check | Pass | Required stable and Open Platform variables present; task project/section optional |
| `+notify` preview | Pass | Local preview mode |
| `+owner-digest` preview | Pass | Repository `Gitlink/gitlink-cli`, risk `high`, score `49` |
| `+contributor-digest` preview | Pass | Role-oriented digest, not personalized routing |
| `+notify --send` | Pass | Custom bot delivered English/default and Chinese cards |
| `+weekly-report --send` | Pass | Custom bot delivered weekly report |
| `+owner-digest --send` | Pass | Custom bot delivered English/default and Chinese owner digest |
| `+contributor-digest --send` | Pass | Custom bot delivered English/default and Chinese contributor digest |
| `+app-check --remote` | Pass | Custom bot, app credentials, and tenant_access_token validated with redacted output |
| `+doc-check --remote` | Pass | App credentials and DocX/folder targets checked; write permission not probed without appending |
| `+bitable-check --remote` | Pass | Five split tables passed sentinel `unique_key` search without record writes |
| `+task-check --remote` | Pass with warnings | Tenant token passed; project/section/dedupe remain next-stage boundaries |
| `+bitable-sync` preview | Pass | 1 report, 5 issue, 2 PR, 1 contributor, 7 task records |
| `+bitable-sync --send` | Pass | Search/create/update real-tested after field creation |
| Split-table `+bitable-sync --send` | Pass | Created 1 report, 5 issue, 2 PR, 1 contributor, and 7 task records across five separate Bitable tables |
| `+doc-export` preview | Pass | 9 DocX-ready blocks |
| `+doc-export --send` | Pass | Appended English/default and Chinese DocX blocks |
| `+task-preview` preview | Pass | 7 task candidates |
| `+task-create --send` | Pass | 7 tasks created; reruns may duplicate tasks |
| Feishu command i18n | Pass | zh-CN cards, digest, DocX blocks, and task titles previewed/sent |
| `go run ./internal/i18n/cmd/check` | Expected fail | Existing `en-US.json` formatting issue outside Feishu module |
| `go test ./shortcuts/feishu` | Pass | Includes task preview count regression test |
| `go test ./shortcuts/workflow` | Pass | Workflow report source remains valid |
| `go test ./shortcuts` | Pass | Shortcut package regression passed |
| `go test ./...` | Pass | Full repository test suite passed |
| Raw secret scan | Pass | No raw secret values found in tracked/unignored candidate files |
| Image evidence | Deferred | No screenshots or image files are included in this upload |

## Do Not Commit

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET
FEISHU_APP_ID
FEISHU_APP_SECRET
tenant_access_token
user_access_token
FEISHU_BASE_APP_TOKEN
table IDs
Wiki node token
folder token
chat_id
open_id
union_id
GITLINK_TOKEN
personal account credentials
```
