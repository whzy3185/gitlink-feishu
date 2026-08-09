# PR readiness shortcuts

## Background

Pull request workflows already support list, create, view, merge, review,
comments, changed files, and patchset/version inspection. Two helper endpoints
were still documented as Raw API calls:

- `GET /:owner/:repo/pulls/get_branches`
- `POST /:owner/:repo/pulls/check_can_merge`

This change adds first-class shortcuts for those PR preparation workflows.

## New shortcuts

- `pr +branches` lists PR source/target branch candidates.
- `pr +check-can-merge` checks whether a source branch can merge into a target
  branch.

## Safety model

`pr +branches` is read-only and can run directly:

```bash
gitlink-cli pr +branches --owner Gitlink --repo forgeplus
```

`pr +check-can-merge` uses a remote POST endpoint, so it supports dry-run and
requires explicit confirmation for real execution:

```bash
gitlink-cli pr +check-can-merge --owner Gitlink --repo forgeplus \
  --head feature/search --base master --dry-run
gitlink-cli pr +check-can-merge --owner Gitlink --repo forgeplus \
  --head feature/search --base master --yes
```

## Documentation updates

- README and README.zh-CN include branch helper and merge readiness examples.
- `skills/gitlink-pr` now prefers `pr +branches` and `pr +check-can-merge`
  over Raw API calls.
- `skills/gitlink-gatekeeper` includes merge readiness as a PR preflight signal.

## Tests

Unit tests cover:

- endpoint method/path mapping for `pr +branches`;
- `pr +check-can-merge` dry-run behavior;
- `--yes` confirmation guard;
- request payload and default `--base master`;
- HTTP error propagation for both shortcuts.

Suggested verification:

```bash
go test ./shortcuts/pr ./shortcuts
```

Full project verification:

```bash
go test ./...
```
