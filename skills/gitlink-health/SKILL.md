---
name: gitlink-health
version: 1.0.1
description: "项目健康度分析（专用工作流）：采集仓库 PR/Issue 数据到 SQLite 并计算聚合指标，生成健康度报告。当用户提到「项目怎么样」「项目健康度」「项目报告」「项目状况」「项目分析」「项目整体情况」等综合分析意图时，必须使用本 skill，不要拆分为 repo/issue/pr 单独操作。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli health +fetch --help"
---

# gitlink-health（开源项目健康度分析技能）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

## 何时使用 / 何时跳过

**必须使用本 skill（不要拆分为 repo/issue/pr）的场景：**
- 用户说"项目怎么样""健康度""项目报告""项目状况""项目整体情况"
- 用户要求综合分析一个项目的 PR、Issue 等多项指标
- 用户要生成任何形式的项目评估报告

**跳过本 skill（使用底层 skill）的场景：**
- 用户明确只操作仓库信息（查看分支、提交等）→ `gitlink-repo`
- 用户明确只操作 Issue（创建、查看、关闭等）→ `gitlink-issue`
- 用户明确只操作 PR（创建、合并、查看 diff 等）→ `gitlink-pr`

## 目录结构

```
- `SKILL.md`: 本文件
- `references/queries.md`: SQL 查询参考与执行指南
- `asset/health_report_template.md`: 报告模板
- `data/gitlink_health.db`: SQLite 数据库
```

## 前置条件

首先确保已完成认证：

```bash
gitlink-cli auth login        # 交互式登录（推荐）
# 或
export GITLINK_TOKEN="your-token"  # 非交互环境设置 Token
```

可通过 `gitlink-cli auth status` 验证登录状态。

## 工作流程

### 1. 采集仓库数据

运行以下命令采集目标仓库的数据：
```bash
gitlink-cli health +fetch --owner OWNER --repo REPO
```
不传 `--owner`/`--repo` 时会自动从当前目录的 git remote 推断。命令运行后会自动生成 `~/.agents/skills/gitlink-health/data/gitlink_health.db` 文件。

### 2. 查询指标

读取 [references/queries.md](references/queries.md)，按其中说明确定目标仓库的 `repo_id`，再逐条执行指标查询。

### 3. 生成报告

按照 [references/queries.md 底部的「报告组装清单」](references/queries.md#报告组装清单)逐项执行查询并填入 [asset/health_report_template.md](asset/health_report_template.md)。**必须逐项打勾核对，输出前确认报告包含全部 14 个模板字段，不允许省略任何一项。**

## 命令参考

### health +fetch

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--owner` | 仓库所有者 | 自动从 git remote 推断 |
| `--repo` | 仓库名称 | 自动从 git remote 推断 |
| `--db` / `-d` | SQLite 数据库路径 | `~/.agents/skills/gitlink-health/data/gitlink_health.db` |
| `--max-pages` / `-M` | 最大页数（不限为全量） | 不限 |


## 数据库表

| 表名 | 用途 | 支持的指标 |
|------|------|-----------|
| `users` | 用户信息（user_name） | - |
| `repos` | 仓库信息（repo_name, owner_id） | - |
| `issues` | Issue 数据 | Issue 解决时长、状态分布 |
| `pulls` | PR 数据 | PR 合并率、贡献者活跃度 |

详细表结构与 API 字段映射规则见 [references/queries.md 表结构章节](references/queries.md#表结构)。