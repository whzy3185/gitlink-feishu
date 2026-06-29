---
name: gitlink-health
version: 1.1.0
description: "项目健康度分析（专用工作流）：提供两种模式 - (1) diagnose 快速诊断：综合评估文档、许可证、社区、成熟度、CI/CD 五大维度；(2) fetch 深度分析：采集 PR/Issue 数据到 SQLite 并计算聚合指标。当用户提到「项目怎么样」「项目健康度」「项目报告」「项目状况」「项目分析」「项目整体情况」等综合分析意图时，必须使用本 skill，不要拆分为 repo/issue/pr 单独操作。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli health --help"
---

# gitlink-health（开源项目健康度分析技能）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

## 何时使用 / 何时跳过

**必须使用本 skill（不要拆分为 repo/issue/pr）的场景：**
- 用户说"项目怎么样""健康度""项目报告""项目状况""项目整体情况"
- 用户要求综合分析一个项目的 PR、Issue 等多项指标
- 用户要生成任何形式的项目评估报告

**选择合适的命令：**
- **快速诊断**：使用 `health diagnose` - 综合评估文档、许可证、社区、成熟度、CI/CD 五大维度，生成健康度评分和改进建议
- **深度分析**：使用 `health +fetch` - 采集 PR/Issue 数据到 SQLite，支持自定义 SQL 查询和详细报告

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

## 功能概述

本 Skill 提供两种健康度分析模式：

### 模式一：快速诊断（health diagnose）

综合评估项目五大维度，生成健康度评分和改进建议：

1. **文档质量**（20分）— 检查 README 完整性、结构化程度
2. **许可证**（15分）— 验证开源许可证是否存在及合规性
3. **社区活跃度**（25分）— 分析贡献者数量、活跃度分布
4. **项目成熟度**（20分）— 评估项目版本、星标、Fork 等指标
5. **CI/CD**（20分）— 检查 CI 激活状态和构建成功率

**适用场景**：快速了解项目整体健康状况，获取改进建议。

### 模式二：深度分析（health +fetch）

采集 PR/Issue 数据到 SQLite，支持自定义查询和详细报告：

1. **数据采集** — 批量获取 PR 和 Issue 数据
2. **指标计算** — 计算合并率、解决时长、活跃度等聚合指标
3. **自定义查询** — 支持 SQL 查询，灵活分析
4. **详细报告** — 生成包含 14 个维度的完整报告

**适用场景**：需要深入分析项目协作数据，生成详细报告。

---

## 工作流程

### 快速诊断模式（推荐首选）

#### 1. 运行诊断命令

```bash
# 诊断当前项目
gitlink-cli health diagnose

# 诊断指定项目
gitlink-cli health diagnose --owner <owner> --repo <repo>

# 指定输出格式
gitlink-cli health diagnose --format json
gitlink-cli health diagnose --format markdown

# 显示详细信息
gitlink-cli health diagnose --verbose
```

#### 2. 查看诊断结果

诊断报告包含：
- 总体健康度评分（满分 100）
- 五大维度评分和状态
- 详细指标信息
- 改进建议

#### 3. 根据建议改进项目

根据报告中的改进建议，针对性地优化项目。

---

### 深度分析模式

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

### health diagnose（快速诊断）

综合评估项目健康度，生成评分和改进建议。

```bash
gitlink-cli health diagnose [flags]
```

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--owner` | `-o` | 项目所有者 | 自动从 git remote 推断 |
| `--repo` | `-r` | 项目名称 | 自动从 git remote 推断 |
| `--format` | `-f` | 输出格式 (text/json/markdown) | text |
| `--verbose` | `-v` | 显示详细输出 | false |
| `--db` | `-d` | 数据库路径（可选） | - |

**输出格式示例：**

```
╔══════════════════════════════════════════════════════════════════╗
║                    🏥 项目健康度诊断报告                          ║
╠══════════════════════════════════════════════════════════════════╣
║  项目：owner/repo                                                ║
║  诊断时间：2024-01-15 10:30:00                                   ║
║  总体状态：🟢 Good (75/100)                                       ║
╚══════════════════════════════════════════════════════════════════╝

📊 维度评分：
  ├─ 📚 文档质量：18/20  [good]
  ├─ ⚖️  许可证：15/15  [good]
  ├─ 👥 社区活跃度：20/25  [warning]
  ├─ 🌟 项目成熟度：12/20  [warning]
  └─ 🔧 CI/CD：10/20  [critical]

💡 改进建议：
  1. 增加核心贡献者数量，提升社区活跃度
  2. 激活 CI/CD 服务，提升代码质量保障
  3. 增加项目星标和 Fork 数量
```

---

### health +fetch（深度分析）

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