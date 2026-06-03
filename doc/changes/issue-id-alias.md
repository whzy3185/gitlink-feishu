# Issue ID Alias

## Summary

`issue +view`, `issue +close`, `issue +update`, and `issue +comment` now accept
`--id` / `-i` as a compatibility alias for `--number` / `-n`.

The alias uses the same project-level issue number shown in the web URL, for
example `issues/123`. It is not the global database ID.

`--number` remains the preferred flag and takes precedence when both flags are
provided.

## Examples

```bash
gitlink-cli issue +view --owner Gitlink --repo forgeplus --id 123
gitlink-cli issue +close --owner Gitlink --repo forgeplus -i 123
gitlink-cli issue +comment --owner Gitlink --repo forgeplus -i 123 --body "Fixed"
```

## Submitter

Wang Yue
