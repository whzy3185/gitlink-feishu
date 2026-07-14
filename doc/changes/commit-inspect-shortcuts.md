# Commit Inspect Shortcuts

Added `gitlink-cli commit` for read-only commit inspection.

| Command | Endpoint |
| --- | --- |
| `commit +list` | `GET /v1/{owner}/{repo}/commits` |
| `commit +files` | `GET /v1/{owner}/{repo}/commits/{sha}/files` |
| `commit +diff` | `GET /v1/{owner}/{repo}/commits/{sha}/diff` |
| `commit +blame` | `GET /v1/{owner}/{repo}/blame` |

Validation:

```bash
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/commit ./shortcuts
go vet ./shortcuts/commit ./shortcuts
go run . commit --help
```
