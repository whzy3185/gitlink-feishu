---
name: gitlink-tag
version: 1.0.0
description: "Git tag management: list tags, search names, view tag details, and delete tags."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli tag --help"
---

# gitlink-tag

Use this skill when an agent needs to inspect or manage Git tags in a GitLink repository.

## Shortcuts

| Shortcut | Purpose |
|----------|---------|
| `tag +list` | List tags with commit details |
| `tag +names` | List tag names without pagination |
| `tag +view` | View a tag by name |
| `tag +delete` | Delete a tag by name |

## Examples

```bash
gitlink-cli tag +list --owner Gitlink --repo forgeplus --page 1 --limit 20
gitlink-cli tag +names --owner Gitlink --repo forgeplus -k v1
gitlink-cli tag +view --owner Gitlink --repo forgeplus --name v1.0.0
gitlink-cli tag +delete --owner Gitlink --repo forgeplus --name release/v1.0.0
```

## Safety Notes

- Confirm the target repository before deleting a tag.
- Use `tag +names` or `tag +list` first when the exact tag name is unknown.
- Tags may contain slashes; pass the full tag name to `--name`.
