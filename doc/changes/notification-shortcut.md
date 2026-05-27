# Notification Shortcut

## Summary

Adds a `notification` shortcut group for GitLink user messages. The group supports listing messages, marking messages as read, and deleting messages without requiring raw API calls.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli notification +list` | List messages for the current or specified user |
| `gitlink-cli notification +read` | Mark specific messages, or all unread messages, as read |
| `gitlink-cli notification +delete` | Delete specific messages |

## Behavior

- `+list` supports `--type notification|atme|all`, `--status unread|read|all`, and pagination.
- `+read` and `+delete` require `--type notification|atme`.
- `+read --ids -1` marks all unread messages of the selected type as read.
- `+delete` rejects `--ids -1` to avoid accidental bulk deletion.
- When `--user` is omitted, the shortcut resolves the current authenticated user via `/users/me`.

## Tests

The unit tests verify current-user resolution, explicit-user paths, query parameters, read/delete payloads, duplicate ID removal, all-unread handling, and validation failures.
