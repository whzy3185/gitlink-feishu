# Message Settings Shortcut

Added a new `message-settings` shortcut group for managing notification delivery preferences from the CLI.

Included commands:

- `message-settings +catalog`
- `message-settings +view`
- `message-settings +update`
- `message-settings +preset`

This change normalizes the setting catalog into stable `Group::Key` identifiers, enriches user settings with group and display names, supports safe partial updates with `--dry-run`, and preserves unspecified settings while updating only the selected keys or groups.
