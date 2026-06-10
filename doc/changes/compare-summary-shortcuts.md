# Compare Summary Shortcuts

## Summary

Adds higher-level compare shortcuts so users and AI agents can inspect commit lists and summarize branch differences without manually stitching together raw compare responses.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli compare +commits` | List commits between two refs with optional author, keyword, limit, and reverse filters. |
| `gitlink-cli compare +summary` | Summarize compare metadata, commit sample, changed file totals, file status counts, top files, path groups, and extension groups. |

## Behavior

- Reuse the existing compare endpoint so branch, tag, and commit refs keep the same URL-safe encoding behavior.
- Normalize commit output into stable fields such as `subject`, `author_login`, and `committer_login`.
- Aggregate compare file data into top changed files, directory groups, extension groups, and created / modified / deleted / renamed counts.
- Mark truncated summaries when `--max-files` analyzes only part of a large compare result.
- Validate `compare +files`, `compare +commits`, and `compare +summary` numeric flags before sending API requests.

## Tests

- `go test ./shortcuts/compare/...`
- `go build ./...`
- `go test ./...`
- `go run . compare +summary --owner Gitlink --repo gitlink-cli --head Mengz:mengz/compare-summary-shortcuts --base master --format json`
