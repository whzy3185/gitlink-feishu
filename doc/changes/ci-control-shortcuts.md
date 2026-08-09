# CI control shortcuts

## Background

The CI shortcut group already supported build listing, log inspection, restart,
and stop operations. Repository-level CI activation, deactivation, and
authorization checks were still documented as Raw API calls in `gitlink-ci`.

This change adds first-class CI control shortcuts.

## New shortcuts

- `ci +activate` activates CI for a repository.
- `ci +deactivate` deactivates CI for a repository.
- `ci +authorize` shows CI authorization state for a repository.

## Safety model

`ci +authorize` is read-only and can run directly:

```bash
gitlink-cli ci +authorize --owner Gitlink --repo forgeplus
```

`ci +activate` and `ci +deactivate` change repository CI state, so they require
an explicit confirmation flag and support dry-run previews:

```bash
gitlink-cli ci +activate --owner Gitlink --repo forgeplus --dry-run
gitlink-cli ci +activate --owner Gitlink --repo forgeplus --yes
```

```bash
gitlink-cli ci +deactivate --owner Gitlink --repo forgeplus --dry-run
gitlink-cli ci +deactivate --owner Gitlink --repo forgeplus --yes
```

## Documentation updates

- README and README.zh-CN include CI control examples.
- `skills/gitlink-ci` now prefers `ci +activate`, `ci +deactivate`, and
  `ci +authorize` instead of Raw API calls.

## Tests

Unit tests cover:

- endpoint method/path mapping for activate, deactivate, and authorize;
- dry-run behavior for state-changing commands;
- `--yes` confirmation guards;
- HTTP error propagation.

Suggested verification:

```bash
go test ./shortcuts/ci ./shortcuts
```

Full project verification:

```bash
go test ./...
```
