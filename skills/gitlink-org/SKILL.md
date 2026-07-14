---
name: gitlink-org
version: 1.0.0
description: "组织管理：查看组织列表、详情、成员，创建组织。当用户需要操作 GitLink 组织时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli org --help"
---

# gitlink-org（组织操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `org +list` | 组织列表 | 否 |
| `org +info` | 组织详情 | 否 |
| `org +members` | 成员列表 | 否 |
| `org +create` | 创建组织 | 是 |
| `org +teams` | 团队列表 | 否 |
| `org +create-team` | 创建团队 | 是 |
| `org +remove-member` | 移除成员 | 是 |

## 使用示例

```bash
gitlink-cli org +list
gitlink-cli org +info --id Gitlink
gitlink-cli org +members --id Gitlink
gitlink-cli org +create --name my-org --description "我的组织"

# 管理团队
gitlink-cli org +teams --id 12345
gitlink-cli org +create-team --id 12345 --name dev-team
gitlink-cli org +remove-member --id 12345 --uid 67890
```

## 注意事项

- `org +create` 创建的组织默认为公开
- 团队管理和成员移除需组织 owner 权限
