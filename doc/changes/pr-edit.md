# PR Edit Shortcut

This change adds a `pr +edit` verb, bringing pull requests in line with `issue`, `label`, `release`, `milestone`, and `webhook`, which already expose an edit/update command.

## Commands

- Add `pr +edit` for `PUT /api/{owner}/{repo}/pulls/{index}.json`.

## Behavior

- `pr +edit` first GETs the current PR (the same `/pulls/{index}` path used by `pr +view`), then merges the requested changes onto its current values before the PUT. The update endpoint requires `title`, `body`, `head`, `base`, `issue_tag_ids`, and `receivers_login` together, so unspecified flags fall back to the existing values to avoid clobbering them.
- Flags: `--title`, `--body`, `--base`, `--head`, and `--tag-ids` (comma-separated). At least one is required.
- `--tag-ids` replaces the tag set when given; otherwise the PR's current tag IDs are preserved.
- PR detail fields are read defensively across the top-level, `pull_request`, and `issue` scopes, matching how the health checks parse the same endpoint.

## Verification

- Unit tests cover field preservation, tag-id override, the nested `pull_request` response shape, the missing-field guard, and the HTTP error path.
