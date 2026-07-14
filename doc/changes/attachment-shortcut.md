# Attachment Shortcut

## Summary

This change adds a new `attachment` shortcut group to `gitlink-cli` so users can upload and delete standalone attachments without dropping down to raw API calls.

## Commands

```bash
gitlink-cli attachment +upload -f ./build.log -d "CI build log"
gitlink-cli attachment +upload -f ./release-notes.md --container-id 42 --container-type VersionRelease
gitlink-cli attachment +delete -i 791eccbf-2e35-4301-ad95-8c937a117f40
```

## API Coverage

- `POST /api/attachments.json`
- `DELETE /api/attachments/{uuid}.json`

## Notes

- Upload uses multipart form data and works with the same `GITLINK_TOKEN` access token flow already used by the CLI.
- Delete accepts the attachment UUID returned by the upload API.
