# Branch Lifecycle Shortcuts

This change expands branch management coverage so repository maintainers can complete more of the branch lifecycle from `gitlink-cli` without falling back to manual API calls.

## Commands

- `branch +all`
- `branch +set-default`
- `branch +restore`

## Improvements

- `branch +list` now supports `--state` so users can inspect visible branches, deleted branches, or all branch records.
- `branch +list` now supports `--keyword` to filter branches by name on the server side.
- The README examples document the deleted-branch recovery flow so users can retrieve `branch_id` and restore the branch in one CLI workflow.

## API Mapping

| Shortcut | Method | API path |
|----------|--------|----------|
| `branch +all` | GET | `/api/v1/{owner}/{repo}/branches/all.json` |
| `branch +set-default` | PATCH | `/api/v1/{owner}/{repo}/branches/update_default_branch.json?name=...` |
| `branch +restore` | POST | `/api/v1/{owner}/{repo}/branches/restore.json` |

## Verification

- Unit tests cover request methods, paths, query parameters, restore payloads, invalid `branch-id` validation, and HTTP error handling.
- Documentation now includes branch filtering, default-branch switching, and deleted-branch restore examples.
