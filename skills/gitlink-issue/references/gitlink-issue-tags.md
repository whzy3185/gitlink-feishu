# Issue Tags

Use `issue +tags` to list the Issue tags available in a repository.
This helps users find tag IDs before creating or updating Issues.

## Usage

```bash
gitlink-cli issue +tags --owner Gitlink --repo forgeplus
gitlink-cli issue +tags --owner Gitlink --repo forgeplus --only-name --keyword bug
gitlink-cli issue +tags --owner Gitlink --repo forgeplus --order-by issues_count --order-direction desc
```

## Options

| Option | Description |
|--------|-------------|
| `--owner` | Repository owner |
| `--repo` | Repository name |
| `--keyword`, `-k` | Optional search keyword |
| `--only-name` | Return only tag names and IDs |
| `--order-by` | Sort field: `updated_on`, `created_on`, or `issues_count` |
| `--order-direction` | Sort direction: `asc` or `desc` |

## API

```http
GET /api/v1/{owner}/{repo}/issue_tags.json
```

Query parameters are mapped as `keyword`, `only_name`, `order_by`, and `order_direction`.
