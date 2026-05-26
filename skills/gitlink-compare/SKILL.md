---
name: gitlink-compare
version: 1.0.0
description: "Compare GitLink branches, tags, or commits and inspect changed files."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli compare --help"
---

# gitlink-compare

Read [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) first for authentication, global flags, and API behavior.

## Shortcuts

| Shortcut | Description |
|----------|-------------|
| `compare +view` | Compare two branches, tags, or commits |
| `compare +files` | List changed files between two refs |

## Examples

```bash
# Compare two refs and include commit/diff summary
gitlink-cli compare +view --owner Gitlink --repo forgeplus --head feature/api --base master

# List changed files
gitlink-cli compare +files --owner Gitlink --repo forgeplus --head feature/api --base master

# Filter a single file in the file diff endpoint
gitlink-cli compare +files --owner Gitlink --repo forgeplus \
  --head feature/api --base master --file cmd/api/api.go
```

## Notes

- Pass normal branch, tag, or commit names. The CLI base64-url encodes refs before calling GitLink compare endpoints.
- `compare +view` calls `/api/{owner}/{repo}/compare/{head}...{base}`.
- `compare +files` calls `/api/v1/{owner}/{repo}/compare/{head}...{base}/files`.
