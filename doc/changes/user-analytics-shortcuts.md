# User Analytics Shortcuts

This change expands the `user` shortcut group beyond basic profile lookup and
adds a read-only analytics toolkit for contribution and activity inspection.

New commands:

- `user +headmap`
- `user +activity`
- `user +develop`
- `user +roles`
- `user +majors`
- `user +trends`

Highlights:

- Adds input validation for year, page, limit, and Unix timestamp ranges.
- Normalizes analytics responses into stable, script-friendly structures.
- Summarizes activity totals, peak days, language distribution, and primary roles.
- Supports cross-page trend filtering by type, project owner, project, and keyword.

Validation:

- `go test ./shortcuts/user/...`
- `go test ./shortcuts/...`
- `go test ./...`
- `go build ./...`
- `go run . user +headmap --login Mengz --format json`
- `go run . user +trends --login Mengz --trend-type PullRequest --limit 2 --format json`
