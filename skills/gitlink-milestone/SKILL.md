---
name: gitlink-milestone
version: 1.0.0
description: "Milestone management: list, create, view, update, delete, close, and reopen GitLink project milestones."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli milestone --help"
---

# gitlink-milestone

**CRITICAL**: Read [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) before starting. It covers authentication, permissions, global flags, and GitLink API behavior.
**CRITICAL**: Confirm user intent before running write or destructive operations such as `+create`, `+update`, `+delete`, `+close`, or `+reopen`.
**CRITICAL**: Use `gitlink-cli` for GitLink resources. Do not use GitHub-only tools such as `gh`.

## Shortcuts

| Shortcut | Description | Operation |
|----------|-------------|-----------|
| `milestone +list` | List repository milestones | Read |
| `milestone +create` | Create a milestone | Write |
| `milestone +view` | View milestone details and linked issues | Read |
| `milestone +update` | Update milestone fields | Write |
| `milestone +delete` | Delete a milestone | Destructive |
| `milestone +close` | Close a milestone | Write |
| `milestone +reopen` | Reopen a closed milestone | Write |

## Examples

```bash
# List open milestones
gitlink-cli milestone +list --owner Gitlink --repo forgeplus --category opening

# Create a milestone
gitlink-cli milestone +create --owner Gitlink --repo forgeplus \
  --name v1.0 --description "First stable release" --due-date 2026-07-01

# View milestone details and linked opened issues
gitlink-cli milestone +view --owner Gitlink --repo forgeplus --id 7 --category opened

# Update the due date
gitlink-cli milestone +update --owner Gitlink --repo forgeplus --id 7 --due-date 2026-08-01

# Close and reopen
gitlink-cli milestone +close --owner Gitlink --repo forgeplus --id 7
gitlink-cli milestone +reopen --owner Gitlink --repo forgeplus --id 7
```

## Parameters

| Command | Key parameters |
|---------|----------------|
| `+list` | `--keyword`, `--category opening,closed`, `--only-name`, `--sort-by`, `--sort-direction`, `--page`, `--limit` |
| `+create` | `--name`, `--description`, `--due-date` |
| `+view` | `--id`, `--category all,opened,closed`, `--author-id`, `--assigner-id`, `--issue-tag-ids`, `--page`, `--limit` |
| `+update` | `--id` plus at least one of `--name`, `--description`, `--due-date` |
| `+delete` | `--id` |
| `+close` / `+reopen` | `--id` |

## API Notes

- Milestone list/create/view/update/delete use `/api/v1/{owner}/{repo}/milestones`.
- Status updates use `/api/{owner}/{repo}/milestones/{id}/update_status`.
- `--due-date` maps to the GitLink API field `effective_date`.
- `--issue-tag-ids` accepts comma-separated IDs and normalizes whitespace before calling the API.
