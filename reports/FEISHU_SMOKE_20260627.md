# Feishu Smoke Report

Date: 2026-06-27

## Branch and Base Commit

```text
branch: feat/feishu-export-clean
base commit: d7812df1af49519f9eb84def218bd3d5a9fdf02f
```

This smoke run included uncommitted data-correctness fixes that are documented
below and will receive a new commit after final validation.

## Environment

```text
Real Feishu test enterprise: used
Real GitLink repository: used
Custom bot: used
Self-built Feishu app: used
DocX target: used
Five split Bitable tables: used
GitLink write operations: not used
```

Real credentials and resource IDs remained in the ignored file:

```text
.local/feishu-gitlink.env.ps1
```

## Readiness Diagnostics

| Command | Result | Side effect |
| --- | --- | --- |
| `feishu +app-check` | pass | none |
| `feishu +doc-check` | pass | none |
| `feishu +bitable-check --tables reports,issues,prs,tasks` | pass | none |
| `feishu +task-check` | pass with expected project/section/dedupe warnings | none |
| `feishu +app-check --remote` | pass | tenant token check only |
| `feishu +doc-check --remote` | pass with write-permission checks skipped | Wiki/DocX read/check only |
| `feishu +bitable-check --remote` | pass for four tables | sentinel search only |
| `feishu +task-check --remote` | pass with expected warnings | tenant token check only |

## Data-Correctness Finding

The first report generated:

```text
issues analyzed: 19
pull requests analyzed: 10
```

The GitLink UI showed:

```text
open issues: 9
open pull requests: 166
```

The values were real API-derived values, but the workflow request semantics
were wrong:

1. Issue workflow fetch sent `state=open`. GitLink Issue list requires
   `category=opened`, so the API ignored the filter and returned 9 open plus 10
   closed issues.
2. PR workflow fetch sent `state=open`. GitLink PR list requires `status=0`, so
   the API ignored the filter and returned all states.
3. The old repo-report defaults analyzed only 20 issues and 10 PRs.
4. The API can cap a requested page at 50 records, so stopping only because a
   page is shorter than the requested limit can truncate a report.

## Data-Correctness Fix

The branch now:

```text
uses category=opened for open Issue queries
uses status=0 for open PR queries
paginates until the API total_count is reached
deduplicates list items by stable identifiers
recognizes GitLink PR index as the user-facing PR number
analyzes all open issues and PRs by default
uses --issue-limit/--pr-limit only as explicit sampling controls
labels Feishu counts as analyzed counts
adds a scope note that sampled values are not repository totals
```

Real post-fix result:

```text
open issues analyzed: 9
open pull requests analyzed: 166
open PR lifecycle total: 166
merged PR lifecycle total: 65
closed/rejected PR lifecycle total: 74
```

These values match the GitLink web UI badges used during the smoke run.

## PR Risk Source

The bulk repo report uses PR list metadata, not changed files, commits, reviews,
or journal comments. The 13 critical metadata classifications came from the
existing `security-sensitive keyword` rule:

| PR | Metadata hit |
| --- | --- |
| 76 | token |
| 77 | token |
| 115 | token |
| 146 | token |
| 167 | secret |
| 171 | token, secret, credential |
| 173 | secret |
| 183 | secret |
| 189 | token |
| 225 | token |
| 254 | token |
| 280 | token |
| 293 | token |

This is a metadata risk hint, not a formal reviewer conclusion. Detailed code
risk requires files and commits. Formal review status must be reported
separately.

## Review and Journal API Validation

Three historical PRs were used only as local read-only samples:

| Sample | Formal review | Journal result |
| --- | --- | --- |
| PR 95 | one approved review with reviewer identity and content | review comments and merged event readable |
| PR 29 | one approved review with reviewer identity and content | review comment and merged event readable |
| PR 75 | no formal review object | review-like comments and rejected/closed event readable |

The repository member list returned 401 without GitLink authentication.
Therefore a generic implementation must not guess maintainer identity. It can
still reliably distinguish the submitter, formal reviewer, participant, and
system event. The cross-repository strategy is documented in:

```text
docs/FEISHU_PR_ACTIVITY_STRATEGY.md
```

## Full PR Review Audit

After the strategy was implemented as an explicit read-only audit path, the
full open-PR inventory was audited with:

```text
workflow +repo-report --include-pr-review-audit
```

Result:

```text
PRs analyzed: 166
PRs review-audited: 166
PRs with formal review evidence: 4
PRs without formal review evidence: 162
PRs needing re-review after reviewer feedback: 0
formal reviews: 4
reviewer comments: 6
submitter comments: 0
participant comments: 436
system events: 0
audit errors: 0
```

Review judgment is conservative:

```text
formal /pulls/{number}/reviews objects mark a PR as reviewed
submitter comments do not mark a PR as reviewed
participant comments do not mark a PR as reviewed
comments by actors with formal review identity are counted as reviewer feedback
reviewed PRs are marked needs_re_review when later submitter comments, later commits, or later PR updates appear after the last reviewer feedback
maintainer identity is not guessed without member-role data
```

## Real Feishu Writes

| Command group | Result |
| --- | --- |
| Eight webhook test/report/digest sends | HTTP 200, Feishu code 0 |
| Corrected full-analysis notify card | HTTP 200, Feishu code 0 |
| Corrected full-analysis owner digest | HTTP 200, Feishu code 0 |
| Full review-audit notify card | HTTP 200, Feishu code 0 |
| Full review-audit owner digest | HTTP 200, Feishu code 0 |
| Final English smoke notify card | HTTP 200, Feishu code 0 |
| Final English smoke owner digest | HTTP 200, Feishu code 0 |
| English and Chinese DocX append | 9 blocks each |
| Corrected Chinese DocX append | 11 blocks |
| Bitable full-analysis upsert | reports 1 updated; issues 5 updated; PRs 5 created/3 updated; contributors 1 updated; tasks 2 created/7 updated |
| Task create | intentionally skipped in this run to avoid duplicates |

## Current Boundaries

```text
No GitLink write operation.
No callback server.
No automatic Base/table/view creation.
No Feishu-side Task dedupe.
No PR activity snapshot persistence.
No previous-snapshot review diff.
No review-content fingerprint persistence.
No maintainer-role guess when member lookup is unavailable.
```

## Test Results

| Check | Result |
| --- | --- |
| `go test ./shortcuts/feishu` | pass |
| `go test ./shortcuts/workflow` | pass |
| `go test ./shortcuts` | pass |
| `go test ./...` | pass |
| `go build .` | pass |
| `go vet ./...` | pass |
| `go run ./internal/i18n/cmd/check` | blocked by existing Windows locale line-ending issue |
| `go run ./internal/i18n/cmd/check --scan-code` | blocked by the same formatting check |

The i18n line-ending and missing-key fix remains in its independent branch/PR
and is intentionally not duplicated into this Feishu change.

## Screenshot Status

The requested Windows computer-use connection failed twice during plugin
initialization:

```text
failed to write kernel assets: path not found
```

No screenshot was fabricated or committed. Text evidence and API-derived
results remain the evidence for this run. See:

```text
reports/FEISHU_SMOKE_EVIDENCE_20260627.md
```
