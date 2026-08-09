# User Account And Stats Shortcuts

Submitter: Wang Yue

This change expands the `user` shortcut group with account metadata, Public Key management, and user statistics OpenAPI coverage.

## Commands

- `user +current`
- `user +keys`
- `user +key-create`
- `user +key-delete`
- `user +activity`
- `user +headmap`
- `user +develop`
- `user +role`
- `user +major`

## API Mapping

| Shortcut | Method | API path |
|----------|--------|----------|
| `user +current` | GET | `/api/users/get_user_info.json` |
| `user +keys` | GET | `/api/public_keys.json` |
| `user +key-create` | POST | `/api/public_keys.json` |
| `user +key-delete` | DELETE | `/api/public_keys/{id}.json` |
| `user +activity` | GET | `/api/users/{owner}/statistics/activity.json` |
| `user +headmap` | GET | `/api/users/{owner}/headmaps.json` |
| `user +develop` | GET | `/api/users/{owner}/statistics/develop.json` |
| `user +role` | GET | `/api/users/{owner}/statistics/role.json` |
| `user +major` | GET | `/api/users/{owner}/statistics/major.json` |

## Verification

- Unit tests cover public key list/create/delete requests, dry-run behavior, current profile lookup, user statistics endpoints, query parameters, and argument validation.
- `user +key-create` supports either inline `--key` content or `--key-file`.
