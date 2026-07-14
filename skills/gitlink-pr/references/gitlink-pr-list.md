# pr +list

> Read [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) first for
> authentication, global flags, and safety rules.

List pull requests for a repository. The command supports state filtering,
keyword search, pagination, and direct lookup by PR number.

## Examples

```bash
# List open pull requests for the current repository
gitlink-cli pr +list

# List merged pull requests for a specific repository
gitlink-cli pr +list --owner Gitlink --repo forgeplus --state merged

# Search by keyword
gitlink-cli pr +list --keyword release --sort-by updated_at --sort-direction desc

# Search by PR number from the web URL
gitlink-cli pr +list --number 42
gitlink-cli pr +list --id 42

# Paginate results
gitlink-cli pr +list --page 2 --limit 10
```

## Flags

| Flag | Required | Description |
|---|---|---|
| `--state`, `-s` | No | Filter by `open`, `merged`, `closed`, or `all`. Default: `open`. |
| `--keyword`, `-k` | No | Search PRs by keyword. |
| `--number`, `-n` | No | Fetch one PR by the PR number shown in the web URL. |
| `--id`, `-i` | No | Compatibility alias for `--number`. |
| `--priority-id` | No | Filter by priority ID. |
| `--tag-id` | No | Filter by issue tag ID. |
| `--milestone-id` | No | Filter by milestone/version ID. |
| `--reviewer-id` | No | Filter by reviewer ID. |
| `--assignee-id` | No | Filter by assignee ID. |
| `--sort-by` | No | Sort field, such as `updated_at` or `created_at`. |
| `--sort-direction` | No | Sort direction: `asc` or `desc`. |
| `--page`, `-p` | No | Page number. Default: `1`. |
| `--limit`, `-l` | No | Page size. Default: `20`. |

## Behavior Notes

- Regular list mode uses `GET /api/v1/{owner}/{repo}/pulls.json`.
- `--number` / `--id` uses `GET /api/v1/{owner}/{repo}/pulls/{index}.json` and
  returns the result in list form so scripts can keep using `pr +list`.
- Each returned PR includes a user-facing `number` field that matches the web UI
  URL `/pulls/N`.
- The PR number is the project-level sequence number, not the global database
  primary key.

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md)
- [gitlink-pr SKILL.md](../SKILL.md)
