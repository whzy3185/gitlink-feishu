---
name: gitlink-webhook
version: 1.0.0
description: "Webhook 配置与测试：列出、创建、查看、更新、删除并触发 webhook 测试。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli webhook --help"
---

# gitlink-webhook（Webhook 管理）

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `webhook +list` | 列出仓库 webhook |
| `webhook +create` | 创建 webhook |
| `webhook +view` | 查看 webhook 详情 |
| `webhook +update` | 更新 webhook |
| `webhook +delete` | 删除 webhook |
| `webhook +test` | 触发 webhook 测试投递 |

## 使用示例

```bash
gitlink-cli webhook +list --owner Gitlink --repo forgeplus
gitlink-cli webhook +create --owner Gitlink --repo forgeplus \
  --url https://example.com/hook --events push,create
gitlink-cli webhook +test --owner Gitlink --repo forgeplus --id 68
```
