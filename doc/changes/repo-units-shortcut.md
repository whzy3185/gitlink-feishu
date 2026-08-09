# Repository Units Shortcut

## Summary

Adds repository navigation unit shortcuts for the GitLink project settings API.

## Commands

```bash
gitlink-cli repo +units --owner Gitlink --repo forgeplus
gitlink-cli repo +set-units --owner Gitlink --repo forgeplus --units code,issues,pulls,wiki
```

## Behavior

- `repo +units` calls `GET /{owner}/{repo}/project_units`.
- `repo +set-units` calls `POST /{owner}/{repo}/project_units` with `unit_types`.
- `--units` accepts a comma-separated list and validates values before making a request.
- Duplicate unit names are removed while preserving order.

## Validation

Allowed units are `code`, `issues`, `pulls`, `devops`, `versions`, `wiki`, `services`, and `resources`.

## Tests

- Unit tests cover the read endpoint, update request body, duplicate handling, and invalid input rejection.
