# User Statistics Shortcuts

## Summary

Adds user-centered statistics shortcuts for contributor analysis workflows:

```bash
gitlink-cli user +heatmap --user zhangsan --year 2026
gitlink-cli user +statistics --user zhangsan
gitlink-cli user +stats --user zhangsan
gitlink-cli user +project-trends --user zhangsan
gitlink-cli user +trends --user zhangsan
```

## Behavior

- `user +heatmap` calls `GET /users/{user}/headmaps`.
- `user +statistics` and alias `user +stats` call `GET /users/{user}/statistics`.
- `user +project-trends` and alias `user +trends` call `GET /users/{user}/project_trends`.
- `--user` is optional. When omitted, the shortcut resolves the current authenticated user via `/users/me`.
- `--year` is supported by `user +heatmap`.
- `--start-time` and `--end-time` are supported by statistics and project trend commands.

## Why

The `gitlink-user` Skill previously documented heatmaps, statistics, and project trends as Raw API calls. Contributor insight workflows also marked `user +heatmap`, `user +stats`, and `user +trends` as unavailable, forcing agents to approximate data from PR lists. These shortcuts expose the read-only user statistics endpoints directly and make contributor analysis more accurate.

## Documentation

- Updates README examples in English and Chinese.
- Updates `skills/README.md`.
- Updates `skills/gitlink-user/SKILL.md` to prefer shortcuts over Raw API.
- Adds dedicated `gitlink-user` reference pages for heatmaps, statistics, and project trends.
- Updates contributor insight guidance to use the new shortcuts when available.

## Verification

```bash
go test ./shortcuts/user ./shortcuts
go test ./...
```
