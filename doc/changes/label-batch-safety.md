# Label batch safety shortcuts

## Background

The label shortcut group already supported listing, creating, updating, and
deleting issue labels. Deletion executed immediately, and larger taxonomy setup
or cleanup workflows still required repeated manual commands or Raw API calls.

This change adds safer destructive operations and first-class batch helpers.

## New and changed shortcuts

- `label +delete` now supports `--dry-run` and requires `--yes` for real
  deletion.
- `label +batch-create` creates multiple labels from a semicolon-separated
  `name:color:description` list.
- `label +batch-delete` deletes multiple labels from a comma-separated ID list.

## Safety model

Destructive or multi-write commands should be previewed first:

```bash
gitlink-cli label +delete --owner Gitlink --repo forgeplus -i 42 --dry-run
gitlink-cli label +batch-create --owner Gitlink --repo forgeplus \
  --labels 'bug:#ee0701:Bug fixes;feature:#0075ca:New features' --dry-run
gitlink-cli label +batch-delete --owner Gitlink --repo forgeplus --ids 3,5,8 --dry-run
```

After confirmation, pass `--yes`:

```bash
gitlink-cli label +delete --owner Gitlink --repo forgeplus -i 42 --yes
gitlink-cli label +batch-create --owner Gitlink --repo forgeplus \
  --labels 'bug:#ee0701:Bug fixes;feature:#0075ca:New features' --yes
gitlink-cli label +batch-delete --owner Gitlink --repo forgeplus --ids 3,5,8 --yes
```

## Parsing rules

- `label +batch-create --labels` uses semicolons between labels and colons
  inside each label spec: `name:color:description`.
- Missing colors default to `#1E90FF`.
- Colors are validated as `#RGB` or `#RRGGBB` before any API call.
- `label +batch-delete --ids` accepts comma-separated positive integer IDs and
  removes duplicates before making requests.

## Documentation updates

- README and README.zh-CN include safe delete and batch examples.
- `skills/gitlink-label` documents the new shortcuts and safety model.
- `skills/gitlink-issue-tag` now recommends label shortcuts instead of Raw API
  calls for common label workflows.
- `skills/gitlink-stale-issue-manager` uses `label +batch-create` for stale
  label bootstrap steps.

## Tests

Unit tests cover:

- single delete dry-run and `--yes` confirmation;
- batch-create dry-run/default preview and real API calls;
- batch-delete dry-run/default preview, de-duplication, and real API calls;
- parser validation for label specs, colors, and ID lists;
- partial batch failure reporting.

Suggested verification:

```bash
go test ./shortcuts/label ./shortcuts
```

Full project verification:

```bash
go test ./...
```
