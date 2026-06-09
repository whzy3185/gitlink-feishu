# Issue batch maintenance shortcuts

## Summary

Add OpenAPI-backed Issue batch maintenance shortcuts:

- `issue +batch-update` — batch update Issue status, priority, milestone, tags, and assigners by API issue IDs.
- `issue +batch-delete` — batch delete Issues by API issue IDs with explicit confirmation.

This complements the existing `issue +batch-close` command. `batch-close` uses web URL issue numbers, while the OpenAPI batch update/delete endpoints use API issue IDs.

## OpenAPI coverage

| Command | Method | Endpoint |
|---|---|---|
| `issue +batch-update` | PATCH | `/api/v1/{owner}/{repo}/issues/batch_update.json` |
| `issue +batch-delete` | DELETE | `/api/v1/{owner}/{repo}/issues/batch_destroy.json` |

## ID semantics

- `issue +batch-close --numbers` uses web URL Issue numbers (`project_issues_index`).
- `issue +batch-update --ids` and `issue +batch-delete --ids` use API Issue IDs returned by Issue APIs.

The docs and help text explicitly call this out to avoid mixing the two ID types.

## Safety and usability

- Both commands support `--dry-run`.
- `issue +batch-update` requires at least one update field.
- `issue +batch-delete` is destructive and requires `--yes` for real execution.
- ID lists are validated as positive integers and de-duplicated.

## Examples

```bash
gitlink-cli issue +batch-update \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102 \
  --status-id 3 \
  --priority-id 2 \
  --tag-ids 7,8 \
  --assigner-ids 11,12 \
  --dry-run

gitlink-cli issue +batch-delete \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102 \
  --dry-run

gitlink-cli issue +batch-delete \
  --owner Gitlink \
  --repo forgeplus \
  --ids 101,102 \
  --yes
```

## Tests

```bash
GOPROXY=https://goproxy.cn,direct go test ./...
go vet ./...
go run . issue +batch-update --help
go run . issue +batch-delete --help
go run . issue +batch-update --owner wangyue111 --repo gitlink-cli --ids 101,102 --status-id 3 --dry-run --format json
go run . issue +batch-delete --owner wangyue111 --repo gitlink-cli --ids 101,102 --dry-run --format json
```
