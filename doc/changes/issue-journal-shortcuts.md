# Issue journal shortcuts

## Background

Issue comments and journal events are useful for stale issue detection, audit
trails, and triage workflows. Those reads were still documented as Raw API calls
against `/v1/:owner/:repo/issues/:number/journals`.

This change adds first-class read-only shortcuts for issue journals.

## New shortcuts

- `issue +journals` lists raw issue journal records.
- `issue +activity` uses the same journal endpoint as an activity-oriented
  alias.

Both commands preserve the existing issue number convention:

- `--number` / `-n` is the preferred web-visible issue number.
- `--id` / `-i` remains a compatibility alias for the same web-visible number,
  not the database ID.

## Options

- `--category` filters journal category, for example `comment`.
- `--page` defaults to `1`.
- `--limit` defaults to `50`.

## Examples

```bash
gitlink-cli issue +journals --owner Gitlink --repo forgeplus --number 123 --page 1 --limit 50
gitlink-cli issue +journals --owner Gitlink --repo forgeplus --number 123 --category comment
gitlink-cli issue +activity --owner Gitlink --repo forgeplus --number 123 --category comment
```

## Documentation updates

- README and README.zh-CN include journal and activity examples.
- `skills/gitlink-issue` documents the new read-only shortcuts.
- `skills/gitlink-stale-issue-manager` now uses `issue +journals` instead of
  Raw API for comment history lookup.

## Tests

Unit tests cover:

- endpoint path and query mapping;
- `--number` and `--id` alias behavior;
- missing issue number validation;
- HTTP error propagation.

Suggested verification:

```bash
go test ./shortcuts/issue ./shortcuts
```

Full project verification:

```bash
go test ./...
```
