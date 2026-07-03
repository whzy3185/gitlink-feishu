# Organization Team Shortcuts

## Summary

Adds a read-only organization team shortcut so users and agents can inspect teams without dropping down to Raw API calls.

## Commands

```bash
gitlink-cli org +teams --id Gitlink
gitlink-cli org +teams --id Gitlink --page 1 --limit 50
gitlink-cli org +teams --id Gitlink --format json
```

## Behavior

- Calls `GET /organizations/{id}/teams`.
- Supports `--page` and `--limit`, matching `org +list` and `org +members` pagination behavior.
- Requires `--id` to avoid accidental ambiguous organization lookup.
- Leaves team creation and destructive team management in Raw API because only the read-only list endpoint is currently documented by the GitLink org Skill.

## Documentation

- Updates the `gitlink-org` Skill index to include `org +teams`.
- Adds a dedicated `gitlink-org-teams` reference page for Agent usage.
- Updates README feature wording to state that organization teams can be inspected through shortcuts.

## Verification

```bash
go test ./shortcuts/org ./shortcuts
go test ./...
```
