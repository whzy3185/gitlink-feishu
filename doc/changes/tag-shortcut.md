# Tag Shortcut

## Summary

Adds a `tag` shortcut group for Git repository tags. This complements release management by exposing the underlying tag inspection and deletion APIs directly.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli tag +list` | List repository tags with commit details using the paginated v1 API |
| `gitlink-cli tag +names` | List tag names without pagination and optionally filter by keyword |
| `gitlink-cli tag +view` | View a single tag by name |
| `gitlink-cli tag +delete` | Delete a tag by name |

## Validation

- `--page` and `--limit` must be positive integers.
- `--name` is required and trimmed for view/delete operations.
- Tag names are URL path escaped before calling v1 item endpoints, so names such as `release/v1.0.0` do not break routing.

## Tests

The unit tests verify paginated listing, name-only lookup, view/delete endpoint paths, tag-name escaping, and invalid input handling.
