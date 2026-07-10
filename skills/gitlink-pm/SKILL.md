---
name: gitlink-pm
version: 1.1.0
description: "项目管理（PM）：Sprint、看板、周报等项目管理功能。当用户需要使用 GitLink PM 功能时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pm --help"
---

# gitlink-pm（项目管理）

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

GitLink PM 模块提供敏捷项目管理能力，包括看板、Sprint、周报等功能。

## Shortcuts

| 命令 | 说明 | 认证 |
|------|------|------|
| `pm +boards` | 查看看板 | Token |
| `pm +sprints` | Sprint Issue 列表 | Token |
| `pm +weekly` | 周报 | Token |
| `pm +tags` | PM Issue 标签 | Token |
| `pm +pipelines` | PM 流水线 | Token |
| `pm +actions` | Action 运行记录 | Token |

## 使用示例

```bash
# 查看项目看板
gitlink-cli pm +boards

# 查看 Sprint Issue 列表
gitlink-cli pm +sprints

# 查看周报
gitlink-cli pm +weekly

# 查看 PM Issue 标签
gitlink-cli pm +tags

# 查看 PM 流水线
gitlink-cli pm +pipelines

# 查看 Action 运行记录（分页）
gitlink-cli pm +actions --page 2 --limit 10
```

## Raw API

如需更灵活的访问，可直接调用 Raw API：

```bash
# 看板
gitlink-cli api GET /pm/dashboards --query 'project_id=123'

# Sprint Issue 列表
gitlink-cli api GET /pm/sprint_issues --query 'project_id=123'

# 周报
gitlink-cli api GET /pm/weekly_issues --query 'project_id=123'

# Issue 标签
gitlink-cli api GET /pm/issue_tags --query 'project_id=123'

# 流水线
gitlink-cli api GET /pm/pipelines --query 'project_id=123'

# Action 运行记录
gitlink-cli api GET /pm/action_runs --query 'project_id=123'
```

## 注意事项

- PM 功能需要项目开启 PM 模块
- Shortcut 命令会自动从 git remote 解析 owner/repo 并获取 project_id
- 所有 PM 端点均为只读 GET 请求
- 如遇到 404 错误，请确认项目已启用 PM 模块
