# Issue Statuses

Use `issue +statuses` to list available Issue status values for a repository.
This helps users find `status_id` values before creating reports or updating Issues.

## Usage

```bash
gitlink-cli issue +statuses --owner Gitlink --repo forgeplus
gitlink-cli issue +statuses --owner Gitlink --repo forgeplus --page 1 --limit 50
```

## Options

| Option | Description |
|--------|-------------|
| `--owner` | Repository owner |
| `--repo` | Repository name |
| `--page`, `-p` | Page number |
| `--limit`, `-l` | Items per page |

## API

```http
GET /api/v1/{owner}/{repo}/issue_statues.json
```

The API path and response field use `statues` as documented by GitLink OpenAPI.
