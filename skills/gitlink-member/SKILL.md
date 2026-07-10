---
name: gitlink-member
version: 1.0.0
description: "项目成员管理：列出、添加、移除项目成员。当用户需要管理 GitLink 项目协作成员时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli member --help"
---

# gitlink-member（成员操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `member +list` | 列出项目成员 |
| `member +add` | 添加项目成员 |
| `member +remove` | 移除项目成员 |

## 使用示例

```bash
# 列出项目成员
gitlink-cli member +list --owner myuser --repo myrepo

# 添加成员（使用用户 ID）
gitlink-cli member +add --owner myuser --repo myrepo --user-id 42

# 移除成员
gitlink-cli member +remove --owner myuser --repo myrepo --user-id 42
```

## API 注意事项

- 成员 API 使用非 v1 路径：`/{owner}/{repo}/collaborators`
- 添加和移除成员需要**用户 ID**（数字），不是用户名。可通过 `user +info` 获取用户 ID
- 移除成员调用 DELETE `/{owner}/{repo}/collaborators/remove`
