# Wiki OpenAPI Shortcuts

## Summary

This change adds a dedicated `wiki` shortcut group for GitLink Wiki OpenAPI
coverage. It includes read, create, update, and delete workflows.

## Added shortcuts

| Shortcut | API |
|----------|-----|
| `wiki +pages` | `GET /api/wiki/wikiPages` |
| `wiki +view` | `GET /api/wiki/getWiki` |
| `wiki +create` | `POST /api/wiki/createWiki` |
| `wiki +update` | `PUT /api/wiki/updateWiki` |
| `wiki +delete` | `DELETE /api/wiki/deleteWiki` |

`wiki +create` and `wiki +update` accept either raw `--content`, which the CLI
base64-encodes, or pre-encoded `--content-base64`. `wiki +delete` supports
`--dry-run` for safe previews.

## Submitter

Wang Yue
