# Feishu Smoke Evidence

Date: 2026-06-27

Branch:

```text
feat/feishu-export-clean
```

Head commit:

```text
138d886 feat(feishu): add full PR inventory and review attribution
```

## Evidence Policy

No screenshot or other binary evidence is committed in this branch.

Visual validation can be attached directly to the PR description after manual
redaction when needed. It must not be stored under repository paths such as
`assets/validation-screenshots/`, `reports/images/`, or `docs/images/`.

## Text Evidence

```text
reports/FEISHU_SMOKE_20260626.md
reports/FEISHU_SMOKE_20260627.md
reports/FEISHU_PERMISSION_MATRIX.md
reports/FEISHU_API_COLLECTION_CHECKLIST_20260626.md
docs/FEISHU_OPENAPI_INVENTORY.md
docs/FEISHU_PR_ACTIVITY_STRATEGY.md
```

## PR Description Evidence

The following text is safe to paste into the PR description instead of adding
image files:

```text
Validation evidence is text-only in the repository. No screenshots are committed.

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

Tests passed:
- go test ./shortcuts/feishu
- go test ./shortcuts/workflow
- go test ./shortcuts
- go test ./...
- go build .
- go vet ./...

Known separate issue:
- go run ./internal/i18n/cmd/check is blocked by an existing locale formatting
  issue handled in a separate branch/PR.
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
[x] No screenshot committed
```

## Capture Rule

Screenshots may be used only outside the repository, for example pasted into the
PR description after redaction. If a local screenshot is temporarily captured,
keep it outside the worktree and delete it after the PR evidence is prepared.
