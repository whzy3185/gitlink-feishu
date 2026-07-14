# Member application shortcuts

This change extends `gitlink-cli member` from direct collaborator and invite-link operations to the project membership application workflow.

New shortcuts:

- `member +applications` lists project membership applications for a user inbox with `--user`, `--page`, and `--per-page`.
- `member +accept-application` accepts an application by `applied_projects[].id` and supports `--dry-run`.
- `member +refuse-application` refuses an application by `applied_projects[].id` and supports `--dry-run`.
- `member +apply` applies to join a project with an application code and requested role, also supporting `--dry-run`.

The implementation follows the documented GitLink OpenAPI endpoints:

- `GET /api/users/{owner}/applied_projects.json`
- `POST /api/users/{owner}/applied_projects/{id}/accept.json`
- `POST /api/users/{owner}/applied_projects/{id}/refuse.json`
- `POST /api/applied_projects.json`

Safety details:

- Application decisions validate positive integer IDs before calling the API.
- Application role values are normalized to `manager`, `developer`, or `reporter`.
- Dry-run output includes the method, path, and request body where applicable.
- When `--user` is omitted, the shortcut uses `--owner` first and falls back to `GET /users/me`.

Verification:

- `go test ./shortcuts/member`
- `go test ./shortcuts`
- `go test ./...`
- `go build ./...`
- `go run ./internal/i18n/cmd/check --scan-code`
- `git diff --check`
