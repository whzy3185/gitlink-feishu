# Pull request review comment management shortcuts

P0 exposure status:

- `pr +review-comments` is registered as a read-only command.
- The create, update, and delete implementations remain intentionally
  unregistered until production API, permission, line-position, and rollback
  behavior are verified.

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

The source implementation validates numeric IDs, boolean filters, comment type,
state, and diff JSON before sending API requests. Source presence does not mean
the write commands are currently exposed.

Verification:

- `go test ./shortcuts/pr`
- `go test ./shortcuts`
- `go test ./...`
- `go build ./...`
- `git diff --check`
