# Repository Settings Shortcuts

Submitter: Wang Yue

This change expands repository shortcut coverage for repository metadata, settings, project topics, navigation units, and transfer OpenAPI endpoints.

## Commands

- `repo +detail`
- `repo +simple`
- `repo +settings`
- `repo +units`
- `repo +units-update`
- `repo +topics`
- `repo +topic-add`
- `repo +topic-delete`
- `repo +transfer-orgs`
- `repo +transfer`
- `repo +transfer-cancel`

## API Mapping

| Shortcut | Method | API path |
|----------|--------|----------|
| `repo +detail` | GET | `/api/{owner}/{repo}/detail.json` |
| `repo +simple` | GET | `/api/{owner}/{repo}/simple.json` |
| `repo +settings` | GET | `/api/{owner}/{repo}/edit.json` |
| `repo +units` | GET | `/api/{owner}/{repo}/project_units.json` |
| `repo +units-update` | POST | `/api/{owner}/{repo}/project_units.json` |
| `repo +topics` | GET | `/api/v1/project_topics.json` |
| `repo +topic-add` | POST | `/api/v1/project_topics.json` |
| `repo +topic-delete` | DELETE | `/api/v1/project_topics/{id}.json` |
| `repo +transfer-orgs` | GET | `/api/{owner}/{repo}/applied_transfer_projects/organizations.json` |
| `repo +transfer` | POST | `/api/{owner}/{repo}/applied_transfer_projects.json` |
| `repo +transfer-cancel` | POST | `/api/{owner}/{repo}/applied_transfer_projects/cancel.json` |

## Verification

- Unit tests cover request methods, paths, query parameters, JSON payloads, dry-run behavior, CSV de-duplication, and invalid project ID validation.
- Write and state-changing commands support `--dry-run`.
