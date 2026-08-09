# Meta and Attachment Shortcuts

## Summary

This change adds high-level shortcuts for GitLink OpenAPI endpoints that were previously only reachable through Raw API:

- `meta +licenses` → `GET /api/licenses.json`
- `meta +ignores` → `GET /api/ignores.json`
- `attachment +upload` → `POST /api/attachments.json`
- `attachment +delete` → `DELETE /api/attachments/{uuid}.json`

## User Value

- Maintainers can query license and `.gitignore` templates before creating repositories.
- Agents can upload files once, capture the returned attachment UUID/URL, and reuse it in Issue/PR/comment workflows.
- Destructive attachment deletion supports `--dry-run` to preview the request before remote mutation.

## Validation

- Unit tests cover query parameters, multipart upload fields, dry-run behavior, missing local files, and deletion.
- README and Skill docs include command examples and Agent safety guidance.
