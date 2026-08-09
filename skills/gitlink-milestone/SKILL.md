---
name: gitlink-milestone
version: 1.0.0
description: "里程碑管理：创建、查看、关闭、删除里程碑。当用户需要管理 GitLink 项目里程碑时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli milestone --help"
---

# gitlink-milestone（里程碑操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `milestone +list` | 列出里程碑 |
| `milestone +create` | 创建里程碑 |
| `milestone +view` | 查看里程碑详情（含关联 Issue） |
| `milestone +close` | 关闭里程碑 |
| `milestone +delete` | 删除里程碑 |

## 使用示例

```bash
# 列出里程碑
gitlink-cli milestone +list --owner Gitlink --repo forgeplus

# 按状态筛选
gitlink-cli milestone +list --owner Gitlink --repo forgeplus --status open

# 创建里程碑
gitlink-cli milestone +create --owner Gitlink --repo forgeplus --name "v2.0" --description "Second major release" --due-date 2026-09-01

# 查看里程碑详情
gitlink-cli milestone +view --owner Gitlink --repo forgeplus --id 5

# 关闭里程碑
gitlink-cli milestone +close --owner Gitlink --repo forgeplus --id 5

# 删除里程碑
gitlink-cli milestone +delete --owner Gitlink --repo forgeplus --id 5
```

## API 注意事项

- 里程碑使用 v1 API：`/v1/{owner}/{repo}/milestones`
- `milestone +view` 支持通过 `--category` 参数筛选关联 Issue（all/opened/closed）
- `milestone +close` 调用更新状态接口，status 设为 "closed"
