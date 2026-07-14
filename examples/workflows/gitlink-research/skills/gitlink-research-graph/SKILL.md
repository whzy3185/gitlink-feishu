---
name: gitlink-research-graph
version: 1.0.0
description: "科研协作知识图谱：采集 GitLink 仓库的贡献者-PR-Issue 关系，构建协作网络图谱，分析核心贡献者/协作社区/知识流动/团队结构。当科研工作者需要了解项目协作生态、识别核心人物、研究开源团队协作模式时触发。覆盖任务四「知识图谱构建」场景。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-research-graph（科研协作知识图谱 · 科研辅助 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)（认证、权限、API 注意事项）。**
**CRITICAL — 本 Skill 为只读采集 + 分析，不写入任何仓库。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 gh（GitHub CLI）操作 GitLink 资源。**

> **定位**：任务四科研辅助 Skill（第 2 个）。构建 GitLink 仓库的**协作知识图谱**——贡献者、PR、Issue 作为节点，"提交/评论/关联"作为边，形成协作网络。AI 分析**核心贡献者、协作社区、知识流动、团队结构**，帮科研工作者洞察开源团队的协作模式。覆盖 PDF 任务四「科研热点追踪与知识图谱构建」场景。

---

## 图谱模型

### 节点（3 类）
| 节点 | 来源 | 科研含义 |
|------|------|---------|
| 👤 贡献者 | PR/Issue 作者、commit 作者 | 协作主体（人）|
| 🔀 PR | `pr +list` | 协作贡献（代码改动）|
| 🐛 Issue | `issue +list` | 协作需求（问题/讨论）|

### 边（4 类关系）
| 边 | 含义 | 来源 |
|----|------|------|
| 贡献者 →提交→ PR | 谁提的 PR | PR.author |
| 贡献者 →创建/评论→ Issue | 谁提/讨论的 Issue | Issue.author / journals |
| PR →关联→ Issue | PR 解决了哪个 Issue（fix #N）| PR.body 含 issue 引用 |
| 贡献者 →review→ PR | 谁审查的 PR | pr +reviews |

---

## 分析维度（4 维）

| 维度 | 分析方法 | 科研问题 |
|------|---------|---------|
| 🎯 核心贡献者 | 度中心性（谁的 PR/Issue 最多）| 谁是项目核心？团队依赖谁？|
| 🤝 协作社区 | 聚类（共同 PR/Issue 的人）| 团队分成几个协作小组？|
| 🔄 知识流动 | PR↔Issue 关联路径 | 问题如何被解决？需求如何落地？|
| 🏗 团队结构 | 贡献者角色分布（提交者/审查者/提问者）| 团队分工健康吗？|

---

## 工作流

### Step 1：采集贡献者（节点）

```bash
# PR 作者（主要贡献者）
gitlink-cli pr +list --owner <owner> --repo <repo> --format json
# 提取每条 PR 的 author.login

# Issue 作者（需求方）
gitlink-cli issue +list --owner <owner> --repo <repo> --format json
# 提取每条 Issue 的 author.login

# 贡献者全量（contributors endpoint 可能返回 HTML，降级方案）
MSYS_NO_PATHCONV=1 gitlink-cli api GET /<owner>/<repo>/contributors.json --format json
# 若返回 HTML：git log --format="%an" | sort | uniq -c | sort -rn
```

### Step 2：采集关系（边）

```bash
# PR-贡献者关系（谁提了哪些 PR）
# 从 Step1 的 pr +list 提取 author + number

# PR-Issue 关联（PR 解决了哪个 Issue）
# 从 PR 的 body/description 提取 "fix #N" / "close #N" 引用

# PR 审查关系（谁 review 了谁）
gitlink-cli pr +reviews --owner <owner> --repo <repo> --id <pr_id> --format json
```

### Step 3：AI 构建图谱 + 分析

综合节点和边，AI 分析四维：

| 分析 | 方法 |
|------|------|
| 🎯 核心贡献者 | 按 PR/Issue 数排序，Top N 为核心；识别"枢纽人物"（review 多的）|
| 🤝 协作社区 | 找共同出现在多个 PR/Issue 的贡献者群（协作紧密的小组）|
| 🔄 知识流动 | 追踪 Issue → PR → merge 路径（需求如何变成代码）|
| 🏗 团队结构 | 角色分布：纯提交者 / 纯审查者 / 提问者 / 全能型 |

### Step 4：输出协作知识图谱分析报告

```markdown
## 🕸️ 科研协作知识图谱 — <owner>/<repo>

### 📊 图谱规模
- 节点：N 贡献者 + M PR + K Issue
- 边：N 条协作关系

### 🎯 核心贡献者（度中心性 Top 5）
| 排名 | 贡献者 | PR数 | Issue数 | 角色 |
|:----:|--------|:----:|:-------:|------|
| 1 | xxx | 42 | 5 | 核心维护者 |
| ... |

### 🤝 协作社区
- 社区A：[人物] 围绕 [模块] 协作
- 社区B：...

### 🔄 知识流动模式
- 典型路径：Issue(提问) → PR(实现) → review(审查) → merge(落地)
- 平均闭环时间：...

### 🏗 团队结构洞察
- 核心层 / 活跃层 / 边缘层 贡献者分布
- 分工健康度评估
```

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| `contributors` API 返回 HTML 非 JSON | 降级用 `git log --format="%an" \| sort \| uniq -c` 聚合 |
| `pr +list` 不返回 author 字段 | 用 `pr +view --id <n>` 逐个取 author（量大时抽样）|
| `pr +reviews` 需逐个 PR 查 | 抽样 Top N PR 分析（避免 API 频率限制）|
| PR-Issue 关联（fix #N）需解析 body | AI 从 PR.description 正则提取 issue 编号 |
| 中文仓库名 URL 编码 | 优先英文 repo 名 |
| `api GET` Git Bash 路径转换 | 加 `MSYS_NO_PATHCONV=1` |

---

## 实测落地参考

**验证仓库**：`Gitlink/gitlink-cli`（29 贡献者 / 322 PR / 19 Issue，协作数据丰富）

| 分析 | 结果 |
|------|------|
| 🎯 核心贡献者 | wbtiger（PR 最多的核心维护者）、wangyue111、puygob236 等 |
| 🤝 协作社区 | 按 shortcut 模块分工（wiki/label/notification 等各有人负责）|
| 🔄 知识流动 | Issue 提需求 → fork 分支 → PR → review → merge（典型开源协作流）|
| 🏗 团队结构 | 多小组并行（各做不同 Skill/命令），wbtiger 统筹合并 |

**科研价值**：gitlink-cli 的协作图谱是研究"开源 AI 工具多团队协作模式"的典型样本——展现了模块化分工 + 集中审核的协作结构。

> 说明：本 Skill 输出**图谱的文字分析**（节点/边/社区/中心性）。若需可视化图谱（力导向图），可将数据导出给前端（ECharts/D3）渲染——是 Dashboard 网页的素材来源。
