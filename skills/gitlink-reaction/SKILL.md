---
name: gitlink-reaction
version: 1.0.0
description: "Repository reactions: list watchers/stargazers, follow or unfollow repositories, and like or unlike repositories."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli reaction --help"
---

# gitlink-reaction

Use this skill when an agent needs to inspect or change repository social interaction state in GitLink.

## Shortcuts

| Shortcut | Purpose |
|----------|---------|
| `reaction +watchers` | List repository watchers |
| `reaction +stargazers` | List repository stargazers |
| `reaction +follow` | Follow a repository |
| `reaction +unfollow` | Unfollow a repository |
| `reaction +like` | Like a repository |
| `reaction +unlike` | Unlike a repository |

## Examples

```bash
gitlink-cli reaction +watchers --owner Gitlink --repo forgeplus
gitlink-cli reaction +watchers --owner Gitlink --repo forgeplus --start-at 1700000000 --end-at 1700003600
gitlink-cli reaction +stargazers --owner Gitlink --repo forgeplus
gitlink-cli reaction +follow --owner Gitlink --repo forgeplus
gitlink-cli reaction +unfollow --owner Gitlink --repo forgeplus
gitlink-cli reaction +like --owner Gitlink --repo forgeplus
gitlink-cli reaction +unlike --owner Gitlink --repo forgeplus
```

## Safety Notes

- Confirm the target repository before follow, unfollow, like, or unlike actions.
- Use `--project-id` when the caller already knows the numeric project ID to avoid the extra repository lookup.
- `--start-at` and `--end-at` must be Unix timestamps.
