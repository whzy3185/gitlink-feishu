# PR View Merged Timestamp

## Summary

`pr +view` now surfaces the merge timestamp at the top level of its output.

The non-v1 detail endpoint `/{owner}/{repo}/pulls/{id}` nests the merge time under
`pull_request.merged_at` (an ISO-8601 string such as `2026-07-05T12:52:05+08:00`),
but the CLI previously only lifted `closed_at`. Merged PRs therefore showed no merge
time, mirroring upstream issue #14.

The `closed_at` enrichment is renamed to `enrichPullRequestTimestamps` and extended so
that, when `pull_request.merged_at` is present, it is copied to `merged_at` at the top
level (and the boolean `merged`, when present, is surfaced alongside it). This matches
`gh pr view`, which exposes `mergedAt`. The existing `closed_at` behavior is unchanged.

## Example

```bash
gitlink-cli pr +view --owner Gitlink --repo forgeplus --id 42
```

```json
{
  "merged_at": "2026-07-05T12:52:05+08:00",
  "merged": true
}
```
