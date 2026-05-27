# Reaction Shortcut

## Summary

Adds a `reaction` shortcut group for repository social interactions. The group exposes watcher and stargazer lists plus follow/unfollow and like/unlike actions.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli reaction +watchers` | List repository watchers |
| `gitlink-cli reaction +stargazers` | List repository stargazers |
| `gitlink-cli reaction +follow` | Follow a repository |
| `gitlink-cli reaction +unfollow` | Unfollow a repository |
| `gitlink-cli reaction +like` | Like a repository |
| `gitlink-cli reaction +unlike` | Unlike a repository |

## Behavior

- `+watchers` and `+stargazers` support optional `--start-at` and `--end-at` Unix timestamp filters.
- Write actions accept optional `--project-id`; if omitted, the shortcut resolves the project ID from `--owner/--repo` automatically.
- Timestamp and project ID inputs are validated before API calls are sent.

## Tests

The unit tests verify list query parameters, project ID auto-resolution, explicit project ID handling, follow/unfollow endpoints, like/unlike endpoints, and validation failures.
