---
name: gitlink-wiki
version: 1.0.0
description: "Wiki management: list, view, create, update, and delete repository wiki pages."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
  shortcuts:
    - "wiki +list"
    - "wiki +view"
    - "wiki +create"
    - "wiki +update"
    - "wiki +delete"
---

# gitlink-wiki

## Shortcuts

| Shortcut | Description |
|----------|-------------|
| `wiki +list` | List repository wiki pages |
| `wiki +view` | View a wiki page |
| `wiki +create` | Create a wiki page |
| `wiki +update` | Update a wiki page |
| `wiki +delete` | Delete a wiki page |

## Examples

```bash
gitlink-cli wiki +list --owner Gitlink --repo forgeplus
gitlink-cli wiki +view --owner Gitlink --repo forgeplus --page Home
gitlink-cli wiki +create --owner Gitlink --repo forgeplus \
  --page Home --title Home --content "Welcome to the project wiki"
gitlink-cli wiki +update --owner Gitlink --repo forgeplus \
  --page Home --file docs/wiki-home.md --message "Update Home"
gitlink-cli wiki +delete --owner Gitlink --repo forgeplus --page Home
```

## Notes

`--project-id` is optional. When omitted, the CLI reads repository metadata and uses the returned `project_id`.
