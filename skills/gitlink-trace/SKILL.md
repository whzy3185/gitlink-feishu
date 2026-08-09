---
name: gitlink-trace
version: 1.0.0
description: "代码溯源分析：初始化账号、发起分支扫描、查看分析结果、重新扫描并获取报告。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli trace --help"
---

# gitlink-trace（代码溯源分析）

Read [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) first for authentication, global flags, owner/repo resolution, and output formatting.

**CRITICAL**: `trace +start` and `trace +rescan` trigger platform-side analysis jobs. Use `--dry-run` first when the user is not explicitly asking to start or rerun a scan.

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `trace +init` | 初始化当前账号的代码溯源分析能力 |
| `trace +results` | 查看仓库代码溯源分析结果，支持 `--page` 和 `--limit` |
| `trace +start` | 对指定分支发起代码溯源分析 |
| `trace +rescan` | 按 `project_id` 重新扫描已有代码溯源结果 |
| `trace +report` | 按 `task_id` 获取代码溯源分析报告 |

## Usage

```bash
# 初始化当前账号
gitlink-cli trace +init

# 查看最近的分析结果
gitlink-cli trace +results --owner Gitlink --repo forgeplus --page 1 --limit 20 --format json

# 先预览，再对 master 分支发起扫描
gitlink-cli trace +start --owner Gitlink --repo forgeplus --branch master --dry-run
gitlink-cli trace +start --owner Gitlink --repo forgeplus --branch master

# 重新扫描已有结果
gitlink-cli trace +rescan --owner Gitlink --repo forgeplus --project-id 67890 --dry-run
gitlink-cli trace +rescan --owner Gitlink --repo forgeplus --project-id 67890

# 获取分析报告
gitlink-cli trace +report --owner Gitlink --repo forgeplus --task-id 12345 --format json
```

## Agent Notes

- Prefer `--format json` when consuming result IDs for a later `trace +report` or `trace +rescan` command.
- Treat `task_id` and `project_id` as platform-provided identifiers from `trace +results`; do not invent them.
- If owner/repo are omitted, the CLI can resolve them from the current Git remote, but explicit flags are safer in automation.
