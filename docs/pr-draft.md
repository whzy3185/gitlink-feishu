# feat(feishu): add layered Feishu collaboration exports for workflow reports

## Summary

- Add Feishu custom-bot cards for GitLink workflow reports.
- Add weekly, owner, and contributor digests.
- Add Bitable-ready schemas and records.
- Add experimental DocX, Bitable, and Task OpenAPI writes.
- Add read-only Open Platform readiness diagnostics.
- Add English and zh-CN Feishu output.
- Fix workflow Issue/PR list filters and paginate all open items by default.
- Add optional read-only PR review audit for formal reviews and comment actor attribution.
- Keep all GitLink write operations out of scope.

## Data Correctness

`workflow +repo-report` now uses the GitLink API parameters used by the native
Issue and PR commands:

```text
Issue open filter: category=opened
PR open filter: status=0
```

It paginates all open issues and pull requests by default. An explicit
`--issue-limit` or `--pr-limit` enables sampling.

Feishu output labels these values as analyzed counts and includes a scope note.
This avoids presenting a limited sample as a repository total.

## PR Review Attribution

`workflow +repo-report --include-pr-review-audit` performs a read-only audit of
the analyzed PRs:

```text
formal /pulls/{number}/reviews objects mark a PR as reviewed
submitter comments are counted separately and do not mark a PR as reviewed
participant comments are counted separately and do not mark a PR as reviewed
comments from actors with formal review identity are counted as reviewer feedback
maintainer role is not guessed when member lookup is unavailable
```

This keeps metadata risk, formal review status, and conversation attribution as
separate signals.

## Stable Surface

```text
feishu +bot-test
feishu +notify
feishu +weekly-report
feishu +owner-digest
feishu +contributor-digest
feishu +bitable-schema
feishu +bitable-records
feishu +task-preview
```

Stable commands preview locally by default. Custom-bot delivery requires
explicit `--send`.

## Readiness Diagnostics

```text
feishu +app-check
feishu +doc-check
feishu +bitable-check
feishu +task-check
```

Local mode checks configuration only. `--remote` performs read/check OpenAPI
calls and does not create or modify Feishu or GitLink resources.

## Experimental Surface

```text
feishu +doc-export
feishu +bitable-sync
feishu +task-create
```

These commands require a Feishu self-built app and explicit `--send`.

## Safety

- Preview/check by default.
- Real Feishu side effects require explicit `--send`.
- Remote readiness calls require explicit `--remote`.
- GitLink write operations are not implemented.
- Card buttons are navigation-only.
- Secrets and resource IDs come from ignored local env files.
- CLI and smoke output redact sensitive values.
- Bitable sync never deletes records.

## Real Validation

The local test enterprise validated:

```text
custom bot card delivery
English and zh-CN cards
DocX append
Bitable search/create/update
Feishu Task create
app/doc/bitable/task readiness diagnostics
```

The current real repository report validated complete default pagination:

```text
open issues analyzed: 9
open pull requests analyzed: 166
open/merged/closed PR lifecycle totals: 166 / 65 / 74
full review-audit result: 166 audited, 4 reviewed, 162 unreviewed
needs re-review after reviewer feedback: 0
```

Task creation was not repeated during the final smoke because Feishu-side
deduplication is not implemented.

## Validation Commands

```powershell
go run . workflow +repo-report --owner "$env:GITLINK_OWNER" --repo "$env:GITLINK_REPO" --format json > .local\report.json
go run . workflow +repo-report --owner "$env:GITLINK_OWNER" --repo "$env:GITLINK_REPO" --include-pr-review-audit --format json > .local\report.review-audit.full.json

go run . feishu +app-check --remote --format table
go run . feishu +doc-check --remote --format table
go run . feishu +bitable-check --tables reports,issues,prs,tasks --remote --format table
go run . feishu +task-check --remote --format table

go run . feishu +notify --from-workflow-json .local\report.json --send --format table
go run . feishu +owner-digest --from-workflow-json .local\report.review-audit.full.json --send --format table
go run . feishu +doc-export --from-workflow-json .local\report.json --send --format table
go run . feishu +bitable-sync --from-workflow-json .local\report.json --send --format table

go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./...
go vet ./...
```

## Review and Comment Attribution Boundary

Formal reviews and PR-associated Issue journals are consumed only by the
optional read-only audit path. Previous-snapshot comparison, member-role
enrichment, and review-content fingerprint persistence remain designed in:

```text
docs/FEISHU_PR_ACTIVITY_STRATEGY.md
```

## Evidence

Validation evidence is text-only in the repository. No screenshots or other
binary evidence are committed. If visual proof is requested, use redacted
screenshots pasted directly into the PR description, not repository files.

```text
reports/FEISHU_SMOKE_20260626.md
reports/FEISHU_SMOKE_20260627.md
reports/FEISHU_SMOKE_EVIDENCE_20260627.md
reports/FEISHU_PERMISSION_MATRIX.md
docs/FEISHU_OPENAPI_INVENTORY.md
```

Text-only validation summary:

```text
Real Feishu custom bot delivery passed:
- final English notify card: HTTP 200 / Feishu code 0
- final English owner digest card: HTTP 200 / Feishu code 0

Real GitLink repository data:
- repository: Gitlink/gitlink-cli
- open issues analyzed: 9
- open PRs analyzed: 166
- PR lifecycle totals: open 166, merged 65, closed/rejected 74

Full PR review audit:
- PRs audited: 166
- reviewed PRs: 4
- unreviewed PRs: 162
- needs re-review: 0
- formal reviews: 4
- reviewer comments: 6
- submitter comments: 0
- participant comments: 436
- system events: 0
- audit errors: 0

Risk source:
- high-risk PRs from metadata rule `security-sensitive keyword`: 13
```

## Out of Scope

- GitLink issue comment or close.
- GitLink PR review, approve, reject, or merge.
- GitLink member management.
- Feishu callback server.
- Feishu-to-GitLink identity mapping.
- Automatic Base/table/field/view creation.
- Task project/section/assignee placement.
- Feishu-side Task deduplication.
- PR activity snapshot persistence.
- Review-content fingerprint diffing.
- Maintainer-role enrichment without authenticated member data.

## Reviewer Questions

- Should webhook export remain the stable main path?
- Should DocX, Bitable, and Task writes remain experimental?
- Should full PR review activity be a separate workflow command?
- Should member-role enrichment require authenticated GitLink access?
- Should future Feishu callbacks live in gitlink-cli or a separate service?
