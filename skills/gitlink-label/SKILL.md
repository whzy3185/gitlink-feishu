---
name: gitlink-label
version: 1.0.0
description: "标签管理：列出、创建、删除 Issue 标签。当用户需要管理 GitLink 项目标签时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli label --help"
---

# gitlink-label（标签操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `label +list` | 列出标签 |
| `label +create` | 创建标签 |
| `label +delete` | 删除标签 |

## 使用示例

```bash
# 列出标签
gitlink-cli label +list --owner Gitlink --repo forgeplus

# 创建标签
gitlink-cli label +create --owner Gitlink --repo forgeplus --name bug --color "#FF0000" --description "Bug report"

# 删除标签（使用 label ID）
gitlink-cli label +delete --owner Gitlink --repo forgeplus --id 3
```

## API 注意事项

- 标签使用 v1 API：`/v1/{owner}/{repo}/issue_tags`
- 创建标签时 color 为可选参数，格式为十六进制颜色值（如 `#FF0000`）
- 删除标签需要标签 ID，可通过 `label +list` 获取
- `label +list` 支持排序：`--order-by`（updated_on/created_on/issues_count）和 `--order-direction`（asc/desc）
