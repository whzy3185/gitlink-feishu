# Wiki Shortcut

Added a new `wiki` shortcut group for repository wiki management.

Included commands:

- `wiki +list`
- `wiki +view`
- `wiki +create`
- `wiki +update`
- `wiki +delete`

The shortcut auto-resolves GitLink `projectId` from repository metadata when `--project-id` is omitted, supports both inline content and file-backed Markdown content, base64-encodes wiki content for the API, and validates conflicting or incomplete input before sending requests.
