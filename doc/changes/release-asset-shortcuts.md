# Release Asset Shortcuts

Submitter: Mengz

This change fills the missing release asset workflow in `gitlink-cli`, so users can upload files, attach existing assets, inspect attached assets, and detach assets without dropping the rest of the release metadata.

## Commands

- Add `release +assets` to inspect the assets currently attached to a release.
- Add `release +attach` to merge existing attachment IDs into a release.
- Add `release +detach` to remove attachment IDs from a release.
- Add `release +upload` to upload a local file and attach it to a release in one command.

## Behavior

- Add multipart upload support to `internal/client` and expose it through `shortcuts/common.RuntimeContext`.
- Keep the `RuntimeContext` encapsulation instead of calling the low-level client directly from shortcuts.
- Preserve existing release fields when attaching, detaching, or uploading assets, so these commands do not accidentally overwrite `name`, `tag_name`, `body`, `target_commitish`, `draft`, or `prerelease`.
- Accept release tags for asset operations by resolving them to the internal release version ID before write operations.
- Clean up the uploaded attachment automatically if the follow-up release update fails, avoiding orphaned assets.

## Verification

- Add multipart client tests covering field values, uploaded filename, content type, and file body.
- Add release shortcut tests covering asset listing, attach/detach dry-run behavior, attachment merging, upload-and-attach flow, and cleanup after failed release update.
