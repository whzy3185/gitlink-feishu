# Member Application Shortcuts

## Background

GitLink OpenAPI documents two repository membership lifecycle endpoints that were not exposed as high-level shortcuts:

- `POST /api/applied_projects.json` for applying to join a project with an invite code.
- `POST /api/{owner}/{repo}/quit.json` for leaving a repository.

These operations are useful for community onboarding/offboarding flows and Agent-assisted repository membership workflows.

## What Changed

Added two `member` shortcuts:

- `member +apply --code --role [--dry-run]`
  - Builds the documented body shape: `{"applied_project":{"code":"...","role":"..."}}`.
  - Validates role as `manager`, `developer`, or `reporter`.
- `member +quit --owner --repo [--dry-run|--yes]`
  - Previews the quit request with `--dry-run`.
  - Requires explicit `--yes` before leaving the repository.

## OpenAPI Coverage

| Command | Method | Endpoint |
|---|---|---|
| `member +apply` | `POST` | `/api/applied_projects.json` |
| `member +quit` | `POST` | `/api/{owner}/{repo}/quit.json` |

## Validation

```bash
git diff --check
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/member ./shortcuts
go vet ./shortcuts/member ./shortcuts
go run . member +apply --help
go run . member +quit --help
GOPROXY=https://goproxy.cn,direct go test ./...
go vet ./...
```
