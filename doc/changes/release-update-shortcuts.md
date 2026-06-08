# Release Update Shortcuts

Submitter: Wang Yue

This change completes the release shortcut coverage for the release edit/update OpenAPI endpoints and improves release write safety.

## Commands

- Add `release +edit` for `/api/{owner}/{repo}/releases/{id}/edit.json`.
- Add `release +update` for `PUT /api/{owner}/{repo}/releases/{id}.json`.
- Extend `release +create` with `--draft` and `--attachment-ids`.
- Extend `release +delete` with `--dry-run`.

## Behavior

- `release +update` fetches current edit data first, then preserves unspecified fields such as `name`, `tag_name`, `body`, `target_commitish`, `draft`, `prerelease`, and existing attachment IDs.
- `release +update` validates boolean flags before reading remote data.
- `release +update` and `release +delete` support `--dry-run` to preview write/delete requests.
- `release +create` validates boolean flags and de-duplicates comma-separated attachment IDs.

## Verification

- Unit tests cover create payloads, edit endpoint routing, update field preservation, attachment overrides, dry-run behavior, and invalid argument validation.
