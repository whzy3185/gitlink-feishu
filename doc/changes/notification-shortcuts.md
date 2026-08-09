# Notification shortcuts

## Background

GitLink exposes user messages and notifications through the messages API. The
existing `gitlink-notification-digest` Skill had to call Raw API paths directly
to list notifications and mark messages as read. That made agent workflows more
fragile and forced users to remember GitLink's "messages" terminology.

This change adds a first-class `notification` shortcut group.

## New shortcuts

- `notification +list` lists messages for the authenticated user or a specified
  user, with type/status/page/limit filters.
- `notification +read` marks selected messages as read, or marks all unread
  messages of a selected type as read.
- `notification +delete` deletes selected messages.
- `notification +send-atme` sends @ mention messages for `Journal`, `Issue`, or
  `PullRequest` targets.

## Safety model

Read-only listing runs directly:

```bash
gitlink-cli notification +list --status unread --limit 20
```

Remote write operations require explicit confirmation and support dry-run
previews:

```bash
gitlink-cli notification +read --ids 740214,740213 --dry-run
gitlink-cli notification +read --ids 740214,740213 --yes
```

```bash
gitlink-cli notification +delete --ids 740214,740213 --dry-run
gitlink-cli notification +delete --ids 740214,740213 --yes
```

```bash
gitlink-cli notification +send-atme --receivers alice,bob \
  --atmeable-type Issue --atmeable-id 123 --dry-run
```

`notification +read --all-unread` maps to GitLink's `ids: [-1]` convention.
The delete command does not expose `--all-unread` to avoid accidental broad
deletion.

## Documentation updates

- README and README.zh-CN include notification usage examples.
- `skills/gitlink-notification/` documents the new shortcut group.
- `skills/gitlink-notification-digest` now prefers `notification +list` and
  `notification +read` instead of Raw API calls.
- The Skills overview lists the new notification Skill.

## Tests

Unit tests cover:

- list endpoint path, filters, pagination, and current-user fallback;
- dry-run behavior for all write operations;
- confirmation guard without `--yes`;
- read payload construction for selected IDs and all unread messages;
- delete payload construction;
- send-atme payload construction and validation;
- invalid argument handling before remote calls.

Suggested verification:

```bash
go test ./shortcuts/notification
```

Full project verification:

```bash
go test ./...
```
