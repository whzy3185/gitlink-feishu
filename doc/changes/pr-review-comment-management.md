# Pull request review comment management shortcuts

This change adds first-class shortcuts for GitLink pull request review comment
threads. It complements `pr +review`, which creates an overall review decision,
with commands for line-level and threaded discussion records.

- `pr +review-comments` lists review comments with keyword, review ID,
  need-response, state, parent, path, full-thread, and sorting filters.
- `pr +review-comment` creates review comments or replies and supports
  `comment`/`problem` types, review IDs, line codes, commit IDs, paths,
  parent IDs, raw diff JSON, and dry-run previews.
- `pr +review-comment-update` edits the note, commit, or state (`opened`,
  `resolved`, `disabled`) with dry-run support.
- `pr +review-comment-delete` deletes a review comment by ID.

The implementation validates numeric IDs, boolean filters, comment type, state,
and diff JSON before sending API requests.

Verification:

- `go test ./shortcuts/pr`
- `go test ./shortcuts`
- `go test ./...`
- `go build ./...`
- `git diff --check`
