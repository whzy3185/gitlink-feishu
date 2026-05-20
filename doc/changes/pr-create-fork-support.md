# PR Create Fork Support

## Summary

This change makes `gitlink-cli pr +create` work with the fork syntax already documented in the README:

```bash
gitlink-cli pr +create --owner Gitlink --repo forgeplus \
  -t "feat: New feature" \
  --head your_username/forgeplus:feature/my-feature \
  --base master
```

## What changed

- Parse `owner/repo:branch` fork heads in `pr +create`
- Auto-resolve the fork repository metadata required by GitLink:
  - `merge_user_login`
  - `merge_project_identifier`
  - `fork_project_id`
- Auto-fill compare counts when available so the request matches GitLink's real PR create flow more closely
- Keep same-repo PR creation behavior unchanged

## Validation

- `go test ./...`
- Unit tests for:
  - same-repo PR creation payload
  - fork PR creation payload
  - invalid fork head syntax
