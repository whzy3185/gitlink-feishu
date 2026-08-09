# Issue batch operations enhancement

## Summary

Add new Issue batch operation shortcuts to enhance issue management capabilities:

- `issue +batch-reopen` — Batch reopen closed issues by web URL issue numbers.
- `issue +batch-label` — Batch add/remove labels from issues by API issue IDs.
- `issue +batch-assign` — Batch assign/unassign users from issues by API issue IDs.
- `issue +batch-comment` — Batch add comments to issues by web URL issue numbers.
- `issue +batch-export` — Export issues to CSV or JSON format with optional filters.
- `issue +batch-import` — Create issues from CSV file.

These commands complement the existing `issue +batch-close`, `issue +batch-update`, and `issue +batch-delete` commands.

## OpenAPI coverage

| Command | Method | Endpoint |
|---|---|---|
| `issue +batch-reopen` | PATCH | `/api/v1/{owner}/{repo}/issues/{id}.json` |
| `issue +batch-label` | GET + PATCH | `/api/v1/{owner}/{repo}/issues/{id}.json` + `/api/v1/{owner}/{repo}/issues/batch_update.json` |
| `issue +batch-assign` | GET + PATCH | `/api/v1/{owner}/{repo}/issues/{id}.json` + `/api/v1/{owner}/{repo}/issues/batch_update.json` |
| `issue +batch-comment` | POST | `/api/v1/{owner}/{repo}/issues/{number}/journals.json` |
| `issue +batch-export` | GET | `/api/v1/{owner}/{repo}/issues.json` |
| `issue +batch-import` | POST | `/api/v1/{owner}/{repo}/issues.json` |

## ID semantics

- `issue +batch-reopen --numbers` uses web URL Issue numbers (`project_issues_index`).
- `issue +batch-comment --numbers` uses web URL Issue numbers (`project_issues_index`).
- `issue +batch-label --ids` uses API Issue IDs returned by Issue APIs.
- `issue +batch-assign --ids` uses API Issue IDs returned by Issue APIs.

The docs and help text explicitly call this out to avoid mixing the two ID types.

## Safety and usability

- All commands support `--dry-run` for preview.
- `issue +batch-label` and `issue +batch-assign` preserve existing labels/assigners and only add/remove specified ones.
- `issue +batch-export` supports filtering by status, assigner, milestone, keyword, and more.
- `issue +batch-import` requires a CSV file with `subject` column (required) and optional columns (`description`, `priority_id`, etc.).
- ID lists are validated as positive integers and de-duplicated.

## Examples

### Batch reopen issues

```bash
gitlink-cli issue +batch-reopen \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 42,43,44 \
  --dry-run

gitlink-cli issue +batch-reopen \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 42,43,44
```

### Batch add/remove labels

```bash
# Add labels to issues
gitlink-cli issue +batch-label \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102,103 \
  --add 1,2 \
  --dry-run

# Remove labels from issues
gitlink-cli issue +batch-label \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102,103 \
  --remove 3,4

# Add and remove labels in one command
gitlink-cli issue +batch-label \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102,103 \
  --add 1,2 \
  --remove 3,4
```

### Batch assign/unassign users

```bash
# Assign users to issues
gitlink-cli issue +batch-assign \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102,103 \
  --add 5,6 \
  --dry-run

# Unassign users from issues
gitlink-cli issue +batch-assign \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102,103 \
  --remove 5,6
```

### Batch add comments

```bash
gitlink-cli issue +batch-comment \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 42,43,44 \
  --message "This issue has been resolved in v2.0.0" \
  --dry-run

gitlink-cli issue +batch-comment \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 42,43,44 \
  --message "Closing as duplicate of #100"
```

### Export issues

```bash
# Export to CSV (default)
gitlink-cli issue +batch-export \
  --owner Gitlink \
  --repo forgeplus \
  --output issues.csv

# Export to JSON
gitlink-cli issue +batch-export \
  --owner Gitlink \
  --repo forgeplus \
  --format json \
  --output issues.json

# Export with filters
gitlink-cli issue +batch-export \
  --owner Gitlink \
  --repo forgeplus \
  --status-id 5 \
  --assigner-id 10 \
  --keyword "bug" \
  --output closed_bugs.csv
```

### Import issues from CSV

```bash
# Create issues from CSV file
gitlink-cli issue +batch-import \
  --owner Gitlink \
  --repo forgeplus \
  --file issues.csv \
  --dry-run

gitlink-cli issue +batch-import \
  --owner Gitlink \
  --repo forgeplus \
  --file issues.csv
```

CSV file format:

```csv
subject,description,priority_id
"Fix login bug","Users cannot login with special characters",1
"Add dark mode","Implement dark mode for the UI",2
"Update documentation","Add API reference for new endpoints",3
```

## Tests

```bash
GOPROXY=https://goproxy.cn,direct go test -v -run "TestBatch" ./shortcuts/issue/...
go vet ./...
go run . issue +batch-reopen --help
go run . issue +batch-label --help
go run . issue +batch-assign --help
go run . issue +batch-comment --help
go run . issue +batch-export --help
go run . issue +batch-import --help
```
