---
name: gitlink-notification
version: 1.0.0
description: "User messages: list GitLink messages, mark messages as read, and delete messages."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli notification --help"
---

# gitlink-notification

Use this skill when an agent needs to inspect or update GitLink user messages.

## Shortcuts

| Shortcut | Purpose |
|----------|---------|
| `notification +list` | List user messages |
| `notification +read` | Mark messages as read |
| `notification +delete` | Delete messages |

## Examples

```bash
gitlink-cli notification +list --type notification --status unread
gitlink-cli notification +list --user Mengz --type atme
gitlink-cli notification +read --type atme --ids 101,102
gitlink-cli notification +read --type notification --ids -1
gitlink-cli notification +delete --type notification --ids 101,102
```

## Safety Notes

- Confirm the target user before using `--user`.
- `notification +list --type all` queries all message types; when `type=all`, avoid assuming `--status` is applied to each backend category in the same way.
- `notification +read --ids -1` marks all unread messages of the selected type as read.
- `notification +delete` requires explicit message IDs and does not accept `-1`.
