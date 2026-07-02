---
name: gitlink-notification
version: 1.0.0
description: "通知与消息管理：查看 GitLink 通知、标记已读、删除消息、发送 @ 提及消息。当用户需要查看或管理站内消息/通知时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
---

# gitlink-notification（通知与消息管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、全局参数和安全规则。**
**CRITICAL — 标记已读、删除消息和发送 @ 消息都是写操作，执行前必须先 dry-run 并确认用户意图。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 功能概述

GitLink 平台的通知在 API 中称为 messages。本 Skill 使用 `notification` shortcut 管理用户消息：

| 命令 | 用途 | 是否写操作 |
|------|------|------------|
| `notification +list` | 列出用户消息和通知 | 否 |
| `notification +read` | 将消息标记为已读 | 是 |
| `notification +delete` | 删除消息 | 是 |
| `notification +send-atme` | 发送 @ 提及消息 | 是 |

## 常用命令

```bash
# 查看当前认证用户的未读通知
gitlink-cli notification +list --status unread --limit 20 --format json

# 查看 @ 我消息
gitlink-cli notification +list --type atme --status unread --format json

# 查看指定用户消息
gitlink-cli notification +list --user zhangsan --type notification --status read --page 1 --limit 20 --format json

# 预览标记指定消息为已读
gitlink-cli notification +read --ids 740214,740213 --dry-run --format json

# 确认标记指定消息为已读
gitlink-cli notification +read --ids 740214,740213 --yes --format json

# 预览将全部未读系统通知标记为已读
gitlink-cli notification +read --type notification --all-unread --dry-run --format json

# 预览删除指定消息
gitlink-cli notification +delete --ids 740214,740213 --dry-run --format json

# 发送 @ 提及消息，先 dry-run
gitlink-cli notification +send-atme --receivers alice,bob \
  --atmeable-type Issue --atmeable-id 123 --dry-run --format json
```

## 参数

### `notification +list`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--user, -u` | 否 | 目标用户登录名，默认使用当前认证用户 |
| `--type, -t` | 否 | 消息类型：`notification` 或 `atme` |
| `--status, -s` | 否 | 状态：`unread`/`1` 或 `read`/`2` |
| `--page, -p` | 否 | 页码，默认 `1` |
| `--limit, -l` | 否 | 每页数量，默认 `20` |

### `notification +read`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--ids, -i` | 条件必填 | 消息 ID，多个用英文逗号分隔 |
| `--all-unread` | 条件必填 | 将所选类型全部未读消息标记为已读 |
| `--type, -t` | 否 | 消息类型，默认 `notification` |
| `--dry-run` | 否 | 预览请求，不修改远端 |
| `--yes` | 否 | 确认执行远端写入 |

### `notification +delete`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--ids, -i` | 是 | 要删除的消息 ID，多个用英文逗号分隔 |
| `--type, -t` | 否 | 消息类型，默认 `notification` |
| `--dry-run` | 否 | 预览删除请求 |
| `--yes` | 否 | 确认执行删除 |

### `notification +send-atme`

| 参数 | 必填 | 说明 |
|------|------|------|
| `--receivers, -r` | 是 | 接收者登录名，多个用英文逗号分隔 |
| `--atmeable-type` | 是 | @ 消息目标类型：`Journal`、`Issue` 或 `PullRequest` |
| `--atmeable-id` | 是 | @ 消息目标对象 ID |
| `--dry-run` | 否 | 预览发送请求 |
| `--yes` | 否 | 确认发送 |

## 安全规则

- `notification +read`、`notification +delete` 和 `notification +send-atme` 默认不会修改远端状态。
- 真实执行前必须先使用 `--dry-run` 查看 `payload`。
- 用户明确确认后，才可以加 `--yes` 执行。
- `notification +read --all-unread` 会向 API 发送 `ids: [-1]`，表示所选类型的全部未读消息。
- `notification +delete` 不支持 `--all-unread`，避免误删大量消息。

## 参考

- [gitlink-shared](../gitlink-shared/SKILL.md)
- [gitlink-notification-digest](../gitlink-notification-digest/SKILL.md)
