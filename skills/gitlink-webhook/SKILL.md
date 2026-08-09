---
name: gitlink-webhook
version: 1.1.0
description: "Webhook 管理：列出、查看、创建、更新、删除、测试 Webhook。当用户需要管理 GitLink 项目 Webhook 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli webhook --help"
---

# gitlink-webhook（Webhook 操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `webhook +list` | 列出 Webhook |
| `webhook +view` | 查看 Webhook 详情 |
| `webhook +create` | 创建 Webhook |
| `webhook +update` | 更新 Webhook 配置 |
| `webhook +delete` | 删除 Webhook |
| `webhook +history` | 查看 Webhook 推送历史 |
| `webhook +test` | 测试 Webhook 推送 |

## 使用示例

```bash
# 列出 Webhook
gitlink-cli webhook +list --owner myuser --repo myrepo

# 查看 Webhook 详情
gitlink-cli webhook +view --owner myuser --repo myrepo --id 1

# 创建 Webhook
gitlink-cli webhook +create --owner myuser --repo myrepo \
  --url https://example.com/webhook \
  --events push,issues \
  --secret my-secret-key

# 更新 Webhook URL 和事件
gitlink-cli webhook +update --owner myuser --repo myrepo \
  --id 1 \
  --url https://example.com/new-hook \
  --events push,issues,pull_request

# 查看推送历史
gitlink-cli webhook +history --owner myuser --repo myrepo --id 1

# 测试推送
gitlink-cli webhook +test --owner myuser --repo myrepo --id 1

# 删除 Webhook
gitlink-cli webhook +delete --owner myuser --repo myrepo --id 1
```

## API 注意事项

- Webhook 使用 v1 API：`/v1/{owner}/{repo}/webhooks`
- 创建 Webhook 时 `--events` 为逗号分隔的事件列表，支持：push, issues, pull_request, create, delete 等
- 不指定 `--events` 时默认监听 push 事件
- `--content-type` 默认为 json，可选 form
- `+test` 命令会实际触发一次 Webhook 推送，请谨慎使用
- `+history` 返回 Webhook 的推送记录，包含每次推送的状态和响应
