# Repo Mirror Sync Shortcut

Added `gitlink-cli repo-mirror +sync` for `POST /repositories/{id}/sync_mirror`.

Safety: supports `--dry-run`; validates `--id` as integer.

Validation:

```bash
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/repomirror ./shortcuts
go vet ./shortcuts/repomirror ./shortcuts
go run . repo-mirror --help
```
