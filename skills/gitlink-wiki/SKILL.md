---
name: gitlink-wiki
version: 1.0.0
description: "Wiki management: list, view, create, update, and delete GitLink wiki pages."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
---

# gitlink-wiki

Read `../gitlink-shared/SKILL.md` first for authentication, global flags, and safety rules.

Write and delete operations must be confirmed by the user before execution. Use
`wiki +delete --dry-run` before deleting a page.

## Shortcuts

| Shortcut | Purpose | Auth |
|----------|---------|------|
| `wiki +pages` | List wiki pages | No for public repos |
| `wiki +view` | Show one wiki page | No for public repos |
| `wiki +create` | Create a wiki page | Yes |
| `wiki +update` | Update a wiki page | Yes |
| `wiki +delete` | Delete a wiki page | Yes |

All shortcuts require `--project-id` because the Wiki OpenAPI requires the
GitLink project ID in addition to owner and repo.

`wiki +create` and `wiki +update` accept either `--content`, which the CLI
base64-encodes, or `--content-base64` for callers that already have encoded
content.

## Examples

```bash
gitlink-cli wiki +pages --owner Gitlink --repo forgeplus --project-id 123
gitlink-cli wiki +view --owner Gitlink --repo forgeplus --project-id 123 --page Home

gitlink-cli wiki +create --owner Gitlink --repo forgeplus \
  --project-id 123 \
  --page Home \
  --title Home \
  --content "Welcome to Wiki"

gitlink-cli wiki +update --owner Gitlink --repo forgeplus \
  --project-id 123 \
  --page Home \
  --title Home \
  --content "Updated content"

gitlink-cli wiki +delete --owner Gitlink --repo forgeplus --project-id 123 --page Home --dry-run
```

## API Mapping

| Shortcut | API |
|----------|-----|
| `wiki +pages` | `GET /api/wiki/wikiPages` |
| `wiki +view` | `GET /api/wiki/getWiki` |
| `wiki +create` | `POST /api/wiki/createWiki` |
| `wiki +update` | `PUT /api/wiki/updateWiki` |
| `wiki +delete` | `DELETE /api/wiki/deleteWiki` |
