# Issue Priorities

Use `issue +priorities` to list the available Issue priority values for a repository.
This is useful before creating or updating an Issue that needs a specific `priority_id`.

## Usage

```bash
gitlink-cli issue +priorities --owner Gitlink --repo forgeplus
gitlink-cli issue +priorities --owner Gitlink --repo forgeplus --keyword normal
```

## Options

| Option | Description |
|--------|-------------|
| `--owner` | Repository owner |
| `--repo` | Repository name |
| `--keyword`, `-k` | Optional search keyword |

## API

```http
GET /api/v1/{owner}/{repo}/issue_priorities.json
```
