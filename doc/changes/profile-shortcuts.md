# Profile (User Statistics) Shortcuts

## Summary

Adds a new read-only `profile` shortcut group that wraps GitLink's user statistics
APIs (development ability, role positioning, major/discipline, recent activity, and
contribution heatmap). These endpoints previously had no shortcut coverage, forcing
agents to fall back to Raw API calls — the `gitlink-contributor-insight` skill even
documents `user +stats`/`user +heatmap` as unavailable. The `profile` group surfaces
the platform's native portrait data directly, powering "research subject portrait"
scenarios.

## Commands

| Command | Purpose | Endpoint |
|---------|---------|----------|
| `gitlink-cli profile +ability` | Development ability scores + language breakdown | `GET /users/{user}/statistics/develop` |
| `gitlink-cli profile +role` | Role positioning | `GET /users/{user}/statistics/role` |
| `gitlink-cli profile +major` | Major/discipline categories | `GET /users/{user}/statistics/major` |
| `gitlink-cli profile +activity` | Recent activity (issues/PRs/commits per day) | `GET /users/{user}/statistics/activity` |
| `gitlink-cli profile +contribution` | Contribution heatmap | `GET /users/{user}/headmaps` |

## Behaviour

- `--user`/`-u` selects the target user. When omitted, the user is resolved from the
  authenticated account via `/users/me`, so `gitlink-cli profile +ability` works with
  no arguments.
- `+ability`, `+role`, and `+major` accept optional `--start-time` / `--end-time`
  (Unix timestamps) that map to the `start_time` / `end_time` query parameters.
- `+contribution` accepts an optional `--year` query parameter.

## Tests

Unit tests cover endpoint paths for every subcommand, the `start_time`/`end_time` and
`year` query parameter mapping, current-user fallback via `/users/me`, the missing-login
error path, and HTTP error handling.

## 中文说明

### 变更内容

- 新增 `profile` 命令组，封装 GitLink 用户画像统计接口：
  - `profile +ability` 开发能力评分（影响力/贡献度/活跃度/项目经验/语言能力）及语言分布
  - `profile +role` 角色定位
  - `profile +major` 专业/学科定位（如深度学习、量子计算）
  - `profile +activity` 近期活动统计（每日疑修/合并请求/提交数量）
  - `profile +contribution` 贡献热力图
- `--user`/`-u` 指定目标用户；缺省时通过 `/users/me` 解析为当前认证用户。
- `+ability`/`+role`/`+major` 支持 `--start-time`/`--end-time`（Unix 时间戳）。
- `+contribution` 支持 `--year`。

### 价值

这些接口此前无任何 shortcut 封装，`gitlink-contributor-insight` Skill 甚至将
`user +stats`/`user +heatmap` 标注为"不可用"并改用 PR 时间戳手工推算。`profile`
命令组直接暴露平台原生画像数据，为"科研主体画像"等场景提供数据底座。
