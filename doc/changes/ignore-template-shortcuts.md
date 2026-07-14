# Ignore Template Shortcuts

## Summary

Added `gitlink-cli ignore +list` to expose GitLink's `.gitignore` template catalog from the CLI. This fills the OpenAPI wrapper gap for `GET /ignores` and helps repository bootstrap workflows pick a valid ignore template before creating a project.

## Commands

| Command | Method / Endpoint | Purpose |
| --- | --- | --- |
| `ignore +list` | `GET /ignores` | List all built-in `.gitignore` templates |
| `ignore +list --name Go` | `GET /ignores?name=Go` | Filter templates by name |

## Validation

```bash
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/ignore ./shortcuts
go vet ./shortcuts/ignore ./shortcuts
go run . ignore --help
go run . ignore +list --help
```
