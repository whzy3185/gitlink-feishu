# Issue comment management shortcuts

This change expands issue comment support from create-only to a full comment
management workflow.

- `issue +comment` now supports threaded replies through `--parent-id` and
  `--reply-id`, attachment IDs, and mentioned users.
- `issue +comments` lists comments and operation records with category,
  keyword, sorting, and pagination filters.
- `issue +comment-update` and `issue +comment-delete` edit or remove existing
  issue comments.
- `issue +comment-replies` lists child comments for threaded conversations.

The implementation keeps the existing `issue +comment -b` behavior compatible
and adds validation for numeric comment, parent, reply, and attachment IDs
before any API request is sent.

Verification:

- `go test ./shortcuts/issue`
- `go test ./shortcuts`
- `go build ./...`
- `git diff --check`
