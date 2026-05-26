---
name: gitlink-growth
version: 1.0.0
description: "开源贡献者成长体系：追踪 PR/Issue 活动 → 生成贡献排行 → 自动颁发徽章。当用户提到贡献者排行、贡献成长、颁发徽章、开发者活跃度排名等意图时触发。"
metadata:
  requires:
    bins: ["gitlink-cli", "sqlite3"]
  cliHelp: "gitlink-cli health +fetch --help"
---

# gitlink-growth（开源贡献者成长体系）

**CRITICAL — 前置阅读：**
1. [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) — 认证、权限处理
2. [`../gitlink-health/references/queries.md`](../gitlink-health/references/queries.md) — 共享数据库表结构
3. [`../gitlink-health/SKILL.md`](../gitlink-health/SKILL.md) — 数据采集命令参考

## 与 gitlink-health 的关系

gitlink-growth **复用** gitlink-health 的 `health +fetch` 命令和同一 SQLite 数据库，但分析视角完全不同：

| 维度 | gitlink-health | gitlink-growth |
|------|---------------|----------------|
| **视角** | 项目整体健康度 | 个人贡献者成长 |
| **输出** | 项目的 Issue/PR/贡献者指标 | 贡献者排行 + 徽章 |
| **关心** | 合并率、解决时长、活跃度 | 谁贡献最多、什么类型、持续度 |

## 目录结构

```
skills/gitlink-growth/
├── SKILL.md                         # 本文件
├── references/
│   └── queries.md                   # 排名 & 徽章 SQL 查询
└── asset/
    └── growth_report_template.md    # 报告模板
```

> **数据库**直接读取 gitlink-health 的 SQLite 文件，不维护独立数据目录。

## 工作流程

### 1. 询问分析偏好

向用户确认以下参数，如果用户未明确指定全部参数，必须逐一询问：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--owner` | 仓库所有者 | 自动从 git remote 推断 |
| `--repo` | 仓库名称 | 自动从 git remote 推断 |
| **时间周期** | 分析的时间范围 | **全量**（从仓库存在至今） |

时间周期选项：
- **本周** — 本周一 00:00:00 至现在
- **本月** — 本月 1 日 00:00:00 至现在
- **本年** — 今年 1 月 1 日 00:00:00 至现在
- **自定义范围** — 用户指定起止日期（如 `2026-01-01 ~ 2026-06-30`）
- **全量**（默认）— 不设时间过滤，查询全部数据

> **记住偏好**：将用户选择的时间周期存入记忆，后续分析直接使用。

### 2. 采集数据

```bash
gitlink-cli health +fetch --owner <owner> --repo <repo>
```

`health +fetch` 会将 PR、Issue、用户、标签数据全部拉取到本地 SQLite 数据库中。数据库默认路径为 `~/.agents/skills/gitlink-health/data/gitlink_health.db`。

> **注意：** 如果数据库已有该仓库的数据且用户只追加了少量新活动，再次 `+fetch` 会增量更新不会丢失历史。

### 3. 查询排名并生成报告

1. 打开 [`references/queries.md`](references/queries.md) 获取 SQL 查询参考
2. 确定 `repo_id`
3. 根据用户选择的时间周期替换 `<start_date>` 和 `<end_date>`
4. 按查询清单逐项执行
5. 将结果填入 [`asset/growth_report_template.md`](asset/growth_report_template.md)
6. 根据 [`references/queries.md#9-徽章生成`](references/queries.md#9-徽章生成) 中的流程分析数据并设计颁发徽章

### 4. 输出报告

将填充完毕的报告写入 markdown 文件并告知用户文件路径。

### 5. 提交报告并上传

询问用户当前报告是否合格，若通过，则通过`git add`添加报告文件，通过`git commit`提交修改，通过`git push`将报告推送至平台。

## 数据库路径

数据存储在 gitlink-health 的 SQLite 数据库中。默认路径：

- **Linux / macOS:** `~/.agents/skills/gitlink-health/data/gitlink_health.db`
- 如果 `health +fetch` 使用了 `--db` 指定了其他路径，则使用该路径

## 徽章规则

Agent 应根据仓库实际数据自行设计徽章，而非套用预定义列表。详见 [`references/queries.md#9-徽章生成`](references/queries.md#9-徽章生成)。

**原则：**
- 执行查询 1-8 和勘探查询获取完整数据画像，从数据中识别模式
- 徽章名称从数据特征派生（如标签名、行为类型），不套用固定模板
- 阈值参考数据分布（取前 N 名、高于均值、或连续 N 周活跃等）
- 达到阈值的贡献者都获得徽章，不设人数上限；贡献者平票时多人并列
- 每个徽章至少 1 人获得；无人达标则不发

> **关于"处理者"（processor）：** 表中的 `processor_id` 即**指派人**（assignee）。GitLink API 不提供 `merged_by`（合入者）信息，所以用 `processor_id` 来代表负责处理该 PR/Issue 的人。

## 边界情况处理

| 情况 | 处理方式 |
|------|---------|
| 数据库不存在 | 提示用户先执行 `health +fetch`，不要试图创建空报告 |
| 时段内无数据 | 排行榜显示「本期无贡献数据」，不颁发任何徽章 |
| 贡献者平票 | 同一名次允许多人并列，徽章颁发给所有平票者 |
| 无标签数据 | 跳过"细分领域排行"章节，不输出空表 |
| 仓库无 Issue/PR | 跳过对应的排行章节 |
| `processor_id` 均为 NULL | 跳过"处理排行"章节，并说明「暂无已指派的记录」 |
