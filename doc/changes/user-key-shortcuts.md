# User SSH Key Shortcuts

## Summary

Adds account-level SSH public key management shortcuts under the existing `user` command group.

## Commands

```bash
gitlink-cli user +keys
gitlink-cli user +add-key --title laptop --key "ssh-ed25519 AAAA..."
gitlink-cli user +add-key --title laptop --from ~/.ssh/id_ed25519.pub
gitlink-cli user +add-key --from ~/.ssh/id_rsa.pub
gitlink-cli user +delete-key --id 123
```

## Behavior

- `user +keys` calls `GET /public_keys` with `--page` and `--limit`.
- `user +add-key` calls `POST /public_keys` with `title` and `key`.
- `user +delete-key` calls `DELETE /public_keys/{id}`.
- `user +add-key` accepts either inline key content or a public key file path.
- `user +add-key --from` can infer the default title from the public key filename.
- Key content and key IDs are validated before a write/delete request is sent.

## Tests

- Unit tests cover list pagination, inline key creation, file-based key creation, invalid key sources, delete path construction, and invalid key IDs.
