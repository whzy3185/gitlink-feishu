# Issue Assigners

Use `issue +assigners` to list users that can be assigned to Issues in a repository.
This helps users find the assignee ID before creating or updating an Issue.

## Usage

```bash
gitlink-cli issue +assigners --owner Gitlink --repo forgeplus
gitlink-cli issue +assigners --owner Gitlink --repo forgeplus --keyword alice
```

## Options

| Option | Description |
|--------|-------------|
| `--owner` | Repository owner |
| `--repo` | Repository name |
| `--keyword`, `-k` | Optional search keyword |

## API

```http
GET /api/v1/{owner}/{repo}/issue_assigners.json
```
