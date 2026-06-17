# Branch Lifecycle Shortcuts

## Background

GitLink OpenAPI exposes branch lifecycle capabilities that were not fully reachable from `gitlink-cli`: keyword/state branch listing, no-pagination listing, default branch switching, and deleted branch restoration.

## What Changed

Extended the `branch` shortcut group with documented OpenAPI coverage:

- `branch +list --keyword --state` maps to `GET /api/v1/{owner}/{repo}/branches.json` query parameters.
- `branch +all` maps to `GET /api/v1/{owner}/{repo}/branches/all.json`.
- `branch +set-default --name` maps to `PATCH /api/v1/{owner}/{repo}/branches/update_default_branch.json?name=...`.
- `branch +restore --id --name` maps to `POST /api/v1/{owner}/{repo}/branches/restore.json` with `branch_id` and `branch_name`.

Write operations support `--dry-run` so users and Agents can inspect the exact request before changing branch state.

## Validation

```bash
git diff --check
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/branch ./shortcuts
go vet ./shortcuts/branch ./shortcuts
go run . branch +set-default --help
go run . branch +restore --help
GOPROXY=https://goproxy.cn,direct go test ./...
go vet ./...
```
