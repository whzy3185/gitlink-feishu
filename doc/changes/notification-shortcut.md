# Notification shortcut

新增 `notification` Shortcut 组，封装 GitLink 通知相关 OpenAPI：

- `notification +list` — 列出通知（`--all` 含已读、`--participating` 仅参与的，支持分页）
- `notification +read` — 标记单条通知为已读（`-i/--id`）
- `notification +read-all` — 标记所有通知为已读
- `notification +watch` — 关注 / 取消关注仓库通知（`-o/--owner`、`-r/--repo`，`--unwatch` 取消）

实现要点：

- `+list` 为 GET `/notifications`，带 `page`/`limit`/`all`/`participating` 查询参数。
- `+read` 为 PUT `/notifications/{id}`；`+read-all` 为 PUT `/notifications`。
- `+watch` 为 POST `/watchers/{owner}/{repo}.json`，`--unwatch` 时改用 DELETE。
- 全部统一 `owner/repo` 自动解析与 `--format json|table|yaml` 输出。

背景：通知管理此前只能在 Web 端手工进行，无法脚本化或被 Agent 调用。`notification` 组补齐命令行入口，便于 CI/Agent 做通知聚合、定期已读、仓库关注等自动化。含单元测试覆盖各命令的 HTTP 方法、路径与查询参数。
