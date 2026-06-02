# Issue Metadata Fields

## Summary

`issue +create` and `issue +update` now support common GitLink Issue metadata fields.
When updating or closing an Issue, the shortcut also carries the current metadata
back to the API so unrelated fields are not reset by partial updates.

## Added flags

| Flag | API field |
|------|-----------|
| `--priority-id` | `priority_id` |
| `--tag-ids` | `issue_tag_ids` |
| `--assigner-ids` | `assigner_ids` |
| `--branch` | `branch_name` |
| `--start-date` | `start_date` |
| `--due-date` | `due_date` |

`issue +create --label` is also mapped as a single tag ID for backward compatibility.

## Examples

```bash
gitlink-cli issue +create --owner Gitlink --repo forgeplus \
  --title "Bug: login failed" \
  --priority-id 3 \
  --tag-ids 4,5 \
  --assigner-ids 7

gitlink-cli issue +update --owner Gitlink --repo forgeplus \
  --number 123 \
  --priority-id 4 \
  --branch bugfix/login \
  --due-date 2026-06-15
```
