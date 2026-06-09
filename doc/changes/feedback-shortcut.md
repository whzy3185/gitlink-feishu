# Feedback shortcut

This change adds a dedicated `feedback` shortcut group for submitting GitLink platform feedback from the CLI.

New command:

- `feedback +create`

The command wraps `POST /api/v1/{owner}/feedbacks.json` and improves the CLI experience around the narrow API payload:

- Resolves the current authenticated user with `GET /users/me` when `--user` is omitted.
- Accepts feedback text from `--content`, `--from`, and `--stdin`, combining multiple sources with blank lines.
- Adds optional metadata lines for `--category`, `--contact`, and `--repo-ref` before the body.
- Supports `--dry-run` to preview method, path, payload, and content length without submitting.
- Rejects empty feedback before making any API request.

Documentation was added to README, README.zh-CN, and `skills/gitlink-feedback/SKILL.md`.

Verification:

- `go test ./shortcuts/feedback`
- `go test ./shortcuts`
- `go test ./...`
- `go build ./...`
- `git diff --check`
- UTF-8 mojibake scan on touched files
