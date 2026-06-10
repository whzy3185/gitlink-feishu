# Message Center Shortcuts

## Summary

This change adds a new `message` shortcut group for personal inbox management in GitLink CLI.
It covers message listing, unread counters, batch mark-as-read, and batch delete workflows without forcing users to drop down to raw API calls.

## Included Commands

- `message +list` filters inbox items by message type, read status, page, and limit.
- `message +stats` returns unread counters for notifications and `@me` messages.
- `message +read` marks selected message IDs, or all unread messages of a given type, as read.
- `message +delete` deletes selected message IDs, or all unread messages of a given type.

## Usability Details

- `--login` defaults to the authenticated user when omitted.
- `--dry-run` is supported for write operations so users can inspect destructive requests first.
- `content_text` is added to list output to expose HTML-free plain text that is easier to grep, diff, and script.

## Validation

- Added shortcut tests for list, stats, read, delete, ID parsing, and content normalization.
- Verified registration by wiring the `message` group into the global shortcut registry.
- Updated `README.md` with feature coverage and usage examples.
