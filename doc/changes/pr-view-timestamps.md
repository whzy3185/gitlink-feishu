# PR View Timestamp Normalization

## Summary

`gitlink-cli pr +view` now normalizes pull request lifecycle timestamps so merged and closed pull requests expose stable top-level time fields in JSON output.

## Command

| Command | Purpose |
|---------|---------|
| `gitlink-cli pr +view` | Return PR detail and normalize `created_at`, `merged_at`, `closed_at`, and `closed_on` when GitLink provides them directly or journals can infer them. |

## Behavior

- Promote `created_at` and `merged_at` from nested response objects to the top-level payload.
- Backfill `closed_at` and `closed_on` for merged pull requests when GitLink omits an explicit close timestamp.
- Read issue journals only when a merged or closed pull request is still missing lifecycle timestamps.
- Recognize merge and close journal operations after stripping HTML tags and whitespace.
- Support both `pull_request_status` and `pull_request_staus` status shapes returned by GitLink APIs.

## Tests

- `go test ./shortcuts/pr/...`
- `go build ./...`
- `go test ./...`
- `go run . pr +view --owner Gitlink --repo gitlink-cli -i 15 --format json`
