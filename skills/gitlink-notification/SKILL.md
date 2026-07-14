---
name: gitlink-notification
version: 1.0.0
description: "通知消息操作：查看消息、标记已读、删除消息、创建 @我通知、查看和更新消息设置。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli notification --help"
---

# gitlink-notification（通知消息管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — `notification +mark-read`、`notification +delete`、`notification +create-atme`、`notification +settings-update` 都会修改远端数据，执行前必须先向用户展示 `--dry-run` 结果并获得确认。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **适用场景：** Agent 帮用户整理 GitLink 站内消息、批量标记已读、清理指定消息、在已有 Issue/PR/Journal 上补发 @我通知、检查或调整消息/邮件通知配置。

## Shortcuts

| Shortcut | 说明 | 读/写 |
|----------|------|-------|
| `notification +list` | 查询用户消息列表，支持 `notification` / `atme` 与已读状态过滤 | 读 |
| `notification +mark-read` | 按消息 ID 标记已读，或使用 `--all-unread` 标记全部未读 | 写 |
| `notification +delete` | 按消息 ID 删除消息 | 写 |
| `notification +create-atme` | 基于 Issue、PullRequest 或 Journal 创建 @我通知 | 写 |
| `notification +platform-settings` | 查看平台消息模板配置 | 读 |
| `notification +settings` | 查看指定用户消息设置 | 读 |
| `notification +settings-update` | 更新用户消息/邮件设置，未指定 key 会保留原值 | 写 |

## 参数约定

- `--user`：OpenAPI 路径中的用户标识，如 `wangyue111`。
- `--type`：消息类型，取值：`notification`（系统消息）或 `atme`（@我消息）。不传表示不限定类型。
- `--status`：列表过滤：`unread`/`1` 表示未读，`read`/`2` 表示已读。
- `--ids`：消息 ID 列表，例如 `101,102,103`。
- `--all-unread`：仅用于 `notification +mark-read`，对应 OpenAPI 的 `ids: [-1]`。
- `--atmeable-type`：@我消息来源对象，取值：`Journal`、`Issue`、`PullRequest`。
- `--notification` / `--email`：消息设置键值对，格式 `Key=true,OtherKey=false`，例如 `Normal::Project=true`。

## 安全工作流

写操作必须遵循：

1. 先读取用户输入并确认目标用户、消息 ID / 设置 key。
2. 先执行带 `--dry-run` 的命令，展示将要请求的 method/path/body。
3. 用户确认后再去掉 `--dry-run` 执行真实写操作。
4. 执行后用 `--format json` 保留结构化结果，便于答辩或审计复现。

## 使用示例

```bash
# 查看未读 @我消息
gitlink-cli notification +list --user wangyue111 --type atme --status unread --format json

# 预览标记指定消息为已读
gitlink-cli notification +mark-read --user wangyue111 --ids 101,102 --dry-run --format json

# 确认后真实标记已读
gitlink-cli notification +mark-read --user wangyue111 --ids 101,102 --format json

# 预览标记全部未读为已读
gitlink-cli notification +mark-read --user wangyue111 --all-unread --type notification --dry-run --format json

# 预览删除指定消息
gitlink-cli notification +delete --user wangyue111 --ids 201,202 --type notification --dry-run --format json

# 创建绑定到 Issue 的 @我通知（先 dry-run）
gitlink-cli notification +create-atme \
  --user wangyue111 \
  --receivers reviewer1,reviewer2 \
  --atmeable-type Issue \
  --atmeable-id 99 \
  --dry-run \
  --format json

# 查看平台消息模板和用户当前设置
gitlink-cli notification +platform-settings --format json
gitlink-cli notification +settings --user wangyue111 --format json

# 预览更新消息设置：只修改指定 key，其余配置保留
gitlink-cli notification +settings-update \
  --user wangyue111 \
  --notification Normal::Project=true,ManageProject::Issue=false \
  --email Normal::Project=false \
  --dry-run \
  --format json
```

## OpenAPI 映射

| Shortcut | Method | Path |
|----------|--------|------|
| `notification +list` | `GET` | `/api/users/{owner}/messages.json` |
| `notification +mark-read` | `POST` | `/api/users/{owner}/messages/read.json` |
| `notification +delete` | `DELETE` | `/api/users/{owner}/messages.json` |
| `notification +create-atme` | `POST` | `/api/users/{owner}/messages.json` |
| `notification +platform-settings` | `GET` | `/api/template_message_settings.json` |
| `notification +settings` | `GET` | `/api/users/{owner}/template_message_settings.json` |
| `notification +settings-update` | `POST` | `/api/users/{owner}/template_message_settings/update_setting.json` |

## Agent 提示模板

当用户要求“帮我清理/整理 GitLink 通知”时：

1. 先问清楚目标用户和范围（只看 @我、只看未读、还是全部消息）。
2. 使用 `notification +list` 获取候选消息。
3. 对标记已读、删除、更新设置等写操作，先执行 `--dry-run`。
4. 把 dry-run 中的 `method`、`path`、`body` 展示给用户确认。
5. 用户确认后执行真实命令，并总结成功/失败结果。
