# milestone +report shortcut

## Overview

This change adds a new `gitlink-cli milestone +report` shortcut for repository
maintainers and AI agents who need a quick milestone health summary before a
release or iteration close-out.

The command accepts either `--id` or `--name`, resolves the target milestone,
collects all linked issues across pages, and outputs a structured report with:

- milestone metadata and completion percentage
- total/open/closed issue counts
- close-readiness blockers and warnings
- open issue breakdown by status, priority, assignee, and tag
- sample open issues for recent activity, unassigned work, and commented threads

## Why it matters

The existing milestone shortcuts cover CRUD and status changes, but they do not
help a maintainer answer practical questions such as:

- Is this milestone ready to close?
- How many open issues are still unassigned?
- Which priorities or tags dominate the remaining work?
- Which open issues should I inspect first?

`milestone +report` turns those checks into one command and keeps the output
machine-friendly for scripts and AI agents.

## Pagination safeguard

The report implementation now fetches milestone issues until the API-reported
total is fully collected, instead of assuming the server always honors the
requested `limit`.

This avoids undercounting when the service caps each response page below the
requested size. For filtered milestone issue views, the implementation uses the
matching filtered totals (`opened_issues_count` / `closed_issues_count`) so it
does not over-fetch extra pages.

## Example commands

```bash
gitlink-cli milestone +report --owner Gitlink --repo forgeplus --id 2438 --sample-limit 3
gitlink-cli milestone +report --owner Gitlink --repo forgeplus --name "v1.0"
```

## Files changed

- `shortcuts/milestone/milestone.go`
- `shortcuts/milestone/report.go`
- `shortcuts/milestone/report_test.go`
- `README.md`

## Validation

```bash
go test ./shortcuts/milestone/...
go test ./shortcuts/...
go test ./...
go build ./...
go run . milestone +report --owner Gitlink --repo forgeplus --id 2438 --sample-limit 3 --format json
```
