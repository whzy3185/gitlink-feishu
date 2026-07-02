---
name: gitlink-pm
version: 2.0.0
description: "项目管理（PM）：Sprint、看板、周报等项目管理功能。当用户需要使用 GitLink PM 功能时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pm --help"
---

# gitlink-pm（项目管理）

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

GitLink PM 模块提供敏捷项目管理能力。优先使用 `pm` shortcut，不再直接拼接 `/pm/...` Raw API。

## 命令

```bash
# 看板
gitlink-cli pm +dashboards --project-id 123 --format json

# Sprint Issue 列表
gitlink-cli pm +sprint-issues --project-id 123 --page 1 --limit 20 --format json

# 周报 Issue
gitlink-cli pm +weekly-issues --project-id 123 --format json

# PM Issue 标签
gitlink-cli pm +issue-tags --project-id 123 --format json

# PM 流水线
gitlink-cli pm +pipelines --project-id 123 --format json

# Action 运行记录
gitlink-cli pm +action-runs --project-id 123 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--project-id, -P` | 是 | GitLink 数字项目 ID |
| `--page, -p` | 否 | 页码，默认 `1` |
| `--limit, -l` | 否 | 每页数量，默认 `20` |

## API 映射

| Shortcut | API |
|----------|-----|
| `pm +dashboards` | `GET /pm/dashboards` |
| `pm +sprint-issues` | `GET /pm/sprint_issues` |
| `pm +weekly-issues` | `GET /pm/weekly_issues` |
| `pm +issue-tags` | `GET /pm/issue_tags` |
| `pm +pipelines` | `GET /pm/pipelines` |
| `pm +action-runs` | `GET /pm/action_runs` |

## 注意事项

- PM 接口需要项目 ID（`project_id`），可通过 `repo +info` 获取。
- PM 功能需要项目开启 PM 模块。
- 这些命令均为只读查询，适合 Agent 生成 Sprint 报告、看板摘要和项目管理周报。
- 建议在 Agent 场景始终使用 `--format json`。
