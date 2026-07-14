---
name: gitlink-pr-guard
version: 1.0.0
description: "代码质量看门人：PR 提交后自动跑完「采集→AI Review→CI 检查→汇总评论→质量判定/合并」端到端流水线。当用户需要 PR 质量门禁、自动化代码审查流水线、PR 自动合并看门人、code quality gate 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-pr-guard（代码质量看门人）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 合并 PR（`pr +merge`）是写操作，执行前必须确认用户意图；默认只「建议合并」，由人确认后再执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`。**

> **前置条件：** 先读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)（认证与全局参数）、[`../gitlink-code-review/SKILL.md`](../gitlink-code-review/SKILL.md)（Review 分析逻辑）。

## 定位：与 gitlink-code-review 的分工

| Skill | 职责边界 |
|-------|---------|
| `gitlink-code-review` | **只做 Review**：分析 PR diff，输出结构化审查意见（Critical/Warning/Suggestion） |
| `gitlink-pr-guard`（本 Skill） | **完整看门人**：Review + CI 检查 + 汇总评论 + **质量判定 + 合并决策**，一条流水线 |

> 本 Skill 是**端到端工作流**（子任务三），串联 `pr` + `code-review` + `ci` + `merge` 多个能力，交付完整质量门禁方案。

---

## 工作流概览（5 步流水线）

| 阶段 | 操作 | 命令 | AI Agent 角色 |
|------|------|------|--------------|
| ① 采集变更 | 拉 PR 详情、文件、Diff | `pr +view` / `+files` / `+diff` | 数据采集 |
| ② AI Review | 分析 diff，分级找问题 | （复用 code-review 逻辑） | 逐文件审查 |
| ③ CI 检查 | 查最新构建状态 | `ci +builds` / `+log` | 状态判定 |
| ④ 汇总评论 | 发布结构化审查报告 | `api POST .../reviews` | 生成报告并发布 |
| ⑤ 质量判定 | 按门禁规则判通过/拒绝 | `pr +merge`（达标且确认后） | 决策 + 执行 |

---

## 详细工作流

### Step 1：采集 PR 变更

```bash
gitlink-cli pr +view --id <pr_id> --format json
gitlink-cli pr +files --id <pr_id> --format json
gitlink-cli pr +diff --id <pr_id> --format json          # 完整 diff
gitlink-cli pr +diff --id <pr_id> --stat                  # 仅统计摘要
```

### Step 2：AI Review（复用 code-review 分析逻辑）

对每个变更文件按 [`../gitlink-code-review/SKILL.md`](../gitlink-code-review/SKILL.md) 的检查项审查，输出分级意见：
- 🔴 **Critical**：硬编码密钥、SQL/命令注入、路径遍历等安全红线
- 🟡 **Warning**：错误处理缺失、边界条件、密码明文等
- 🔵 **Suggestion**：命名、性能、可配置化等优化
- ✅ **Positive**：值得肯定的设计

### Step 3：CI 检查

```bash
# 查最新构建
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
# 从返回找 PR 对应分支的最新构建，取 status（success/failure/pending）
# 失败时取日志定位原因
gitlink-cli ci +log --build <build_number>
```

### Step 4：发布汇总评论（审查报告）

```bash
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{
  "body": "<质量看门人报告，见输出模板>",
  "event": "COMMENT"
}'
```

### Step 5：质量判定（门禁核心）

按下方「质量门禁规则」判定，达标且**用户确认后**执行合并：

```bash
# 达标 + 用户确认 → 合并（默认建议，不自动执行）
gitlink-cli pr +merge --id <pr_id> --method squash
```

---

## 质量门禁规则（决策表）

本 Skill 的灵魂：把「能否合并」从主观判断变成可量化门禁。

| 条件 | 判定 | 动作 |
|------|:----:|------|
| 0 Critical **且** CI = success | ✅ **通过** | 建议合并（确认后 `pr +merge`） |
| 有 Critical（任一） | 🔴 **拒绝** | 请求修改，逐条列 Critical + 文件:行号 |
| CI = failure | 🔴 **拒绝** | 请求修改，附 CI 失败日志摘要 |
| CI = pending | ⏳ **等待** | 等构建完成再判定 |
| 0 Critical + CI success + 有 Warning | 🟡 **通过(带建议)** | 合并 + 评论里列出 Warning 供后续优化 |
| 仅 Suggestion | ✅ **通过** | 合并 + 评论附优化建议 |

**安全红线**（必须 Critical，不得合并）：
- 硬编码 Token / 密钥 / 密码
- SQL / NoSQL / 命令注入
- 路径遍历、不安全反序列化、XSS

---

## 输出模板：质量看门人报告

```markdown
## 🚪 质量看门人报告 — PR #<id> <title>

### 📊 质量判定：<✅ 通过 / 🔴 拒绝 / ⏳ 等待 CI>

| 维度 | 结果 |
|------|------|
| 变更规模 | <n> 文件，+<add> / -<del> |
| 🔴 Critical | <n> |
| 🟡 Warning | <n> |
| 🔵 Suggestion | <n> |
| CI 构建 | <success/failure/pending>（build #<n>） |

### 🔴 Critical（必须修改）
- <问题> — `<file>:<line>`
  > <修改建议>

### 🟡 Warning（建议修改）
- <问题> — `<file>:<line>`

### ✅ CI 状态
- 构建 #<n>：<success/failure>，耗时 <duration>
- 失败原因（如有）：<日志摘要>

### 📋 处置建议
<根据门禁规则：合并 / 请求修改 / 等 CI>

---
*由 gitlink-pr-guard 质量看门人自动生成*
```

---

## 工作流决策规则

| 场景 | 处理 |
|------|------|
| 用户说"帮我把关这个 PR" | 跑完整 5 步，输出报告 + 判定 |
| 用户只要 Review 不要合并 | 跑 Step 1-4，跳过 Step 5（指向 code-review） |
| PR 是 draft 草稿 | 提示先标记 Ready for Review |
| CI 还在跑 | Step 3 返回 pending，报告标 ⏳，建议稍后再判 |
| Critical 数 ≥1 | 报告标 🔴 拒绝，**不合并**，列清 Critical |
| 用户要求自动合并 | 警告风险，仅在 0 Critical + CI success 时执行，且二次确认 |

---

## 注意事项

- **合并是写操作**：默认只「建议合并」，`pr +merge` 必须用户确认后执行
- **CI 对应分支**：`ci +builds` 返回多条，需按 PR 的 source_branch 匹配最新构建
- **Review 评论量控制**：最多列 3-5 个最严重问题，避免信息过载
- **草稿 PR**：先提示标记 Ready for Review
- **与 code-review 复用**：Step 2 的审查逻辑直接引用 code-review SKILL.md，不重复定义
- **可复现**：配套 `pr-guard-workflow.sh` 脚本可端到端复现整个流水线

## References

- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证与全局参数
- [gitlink-code-review](../gitlink-code-review/SKILL.md) — Review 分析逻辑（Step 2 复用）
- [gitlink-pr](../gitlink-pr/SKILL.md) — PR 命令
- [gitlink-ci](../gitlink-ci/SKILL.md) — CI 命令
