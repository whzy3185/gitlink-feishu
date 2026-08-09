# Issue Reopen

## Summary

`issue +reopen` reopens a closed issue, mirroring `issue +close`. It brings the
issue shortcut group to parity with `milestone +reopen` and `pr +reopen`, which
already had the counterpart to their close command.

Like `issue +close`, the v1 PATCH is read-modify-write, so the current issue is
fetched first and its metadata (priority, tags, assigners, linked branch, dates)
is replayed alongside the new status so unrelated fields are not reset. Only
`status_id` is flipped: `1` (open) for reopen, `5` (closed) for close. Both
commands share the same helper, so `+reopen` preserves exactly the fields
`+close` already does.

## Examples

```bash
gitlink-cli issue +reopen --owner Gitlink --repo forgeplus --number 123
gitlink-cli issue +reopen --owner Gitlink --repo forgeplus -i 123
```

`--number` / `-n` is the project-level issue number from the web URL; `--id` /
`-i` is accepted as a compatibility alias.
