---
name: gitlink-commit
version: 1.0.0
description: "GitLink 提交审查：提交列表、单个提交文件、diff 与 blame 查询。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli commit --help"
---

# gitlink-commit

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**

## Shortcuts

| Shortcut | 说明 |
| --- | --- |
| `commit +list` | 查询提交列表 |
| `commit +files` | 查询单个提交变更文件 |
| `commit +diff` | 查询单个提交 diff |
| `commit +blame` | 查询文件 blame |

## 示例

```bash
gitlink-cli commit +list --owner Gitlink --repo forgeplus --sha master --page 1 --limit 20
gitlink-cli commit +files --owner Gitlink --repo forgeplus --sha <sha> --filepath README.md
gitlink-cli commit +diff --owner Gitlink --repo forgeplus --sha <sha>
gitlink-cli commit +blame --owner Gitlink --repo forgeplus --sha master --filepath README.md
```

全部命令均为只读，适合代码审查、变更追踪、科研仓库复现和 Agent 预检。
