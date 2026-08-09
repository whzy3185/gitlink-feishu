---
name: gitlink-quality-gate
version: 3.0.0
description: "质量看门编排：依次调度 gitlink-pr → gitlink-code-review → gitlink-ci → gitlink-pr 四个子 Skill。当用户需要对 PR 做质量门禁检查时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-quality-gate（质量看门 · 编排 Skill）

> **你是编排者（Orchestrator），不是执行者。每一步都通过 Skill 工具调用对应的子 Skill 来完成。不要自己直接跑命令。**

---

## 编排架构

```
gitlink-quality-gate
  ├── Step 1 → Skill("gitlink-pr")           列出 PR，确定审查目标
  ├── Step 2 → Skill("gitlink-code-review")  代码审查，生成分级报告
  ├── Step 3 → Skill("gitlink-ci")           检查 CI 构建状态
  └── Step 4 → Skill("gitlink-pr")           汇总判定：合并 or 驳回
```

## 子 Skill 依赖

| 顺序 | 子 Skill | 用途 | 写入 |
|:----:|----------|------|:----:|
| 1 | gitlink-pr | 列出开放 PR，获取 PR 详情 | 否 |
| 2 | gitlink-code-review | 获取变更 → 逐文件分析 → 分级报告 → 提交 Review | 是 |
| 3 | gitlink-ci | 查看构建列表和日志，判定 CI 状态 | 否 |
| 4 | gitlink-pr | 门禁判定：达标合并，不达标评论驳回 | 是 |

---

## 前置：收集参数

| 参数 | 说明 | 示例 |
|------|------|------|
| owner | 仓库所有者 | `ylly` |
| repo | 仓库名称 | `gitlink-cli` |
| PR 编号（可选） | 指定审查哪个 PR | 如果不提供，自动列出开放 PR |

---

## 工作流

### Step 1：找 PR

**→ 调用 Skill 工具：`Skill("gitlink-pr", args="列出 <owner>/<repo> 的所有开放 PR（state=open），确认待审查的 PR 编号。如果没有开放 PR，按 Fork 流程自己提一个测试 PR。")`**

调用后记录 PR 编号、标题、分支信息，传递给 Step 2。

### Step 2：代码审查

**→ 调用 Skill 工具：`Skill("gitlink-code-review", args="对 <owner>/<repo> 的 PR #<id> 执行完整代码审查：获取变更文件 → 逐文件分析（按语言检查清单）→ 按 Critical/Warning/Suggestion/Positive 分级 → 生成审查报告。")`**

调用后记录审查结果（Critical/Warning/Suggestion 数量），传递给 Step 4。

**⚠️ 审查报告的提交方式（关键）**：

审查结束后，必须用以下方式**各提交一次**，确保报告同时出现在审查记录和 PR 讨论流中：

**提交 1：审查记录（必须用 common，禁止用 approved）**
```bash
gitlink-cli pr +review --owner <owner> --repo <repo> --id <pr_id> \
  --status common \
  --content "<审查报告 Markdown>"
```
> `--status common` = API 的 `event: "COMMENT"`，报告会出现在 PR 的 Review 记录中。
> **绝对不要用 `--status approved`**，那只是点了个「通过」按钮，审查报告正文不显眼。

**提交 2：PR 评论（可选，让报告更显眼）**
```bash
gitlink-cli pr +comment --owner <owner> --repo <repo> --id <pr_id> \
  --body "<审查报告 Markdown>"
```

**⚠️ Windows 避免 emoji 乱码**：审查报告中的 🔴🟡🔵✅⚠️ 等 emoji 在 Windows Git Bash 下会变成 `?`。审查报告中使用纯文本标记替代：
- `[Critical]` 替代 🔴
- `[Warning]` 替代 🟡
- `[Suggestion]` 替代 🔵
- `[Positive]` 替代 ✅
- `[Skip]` 替代 ⚠️

### Step 3：CI 检查

**→ 调用 Skill 工具：`Skill("gitlink-ci", args="查看 <owner>/<repo> 的 CI 构建列表和日志，判定构建状态（通过/失败/运行中/无配置）。")`**

调用后记录 CI 状态（通过/失败/无配置），传递给 Step 4。

### Step 4：门禁判定

**→ 调用 Skill 工具：`Skill("gitlink-pr", args="汇总 PR #<id> 的审查结果（Critical=N, Warning=N）和 CI 状态（<状态>）。判定规则：Critical=0 且 CI 通过 → 合并（pr +merge）；否则 → 评论驳回（pr +comment）。合并前确认用户意图。")`**

---

## 门禁判定矩阵

| Critical | CI 状态 | 判定 | 动作 |
|:--------:|:------:|:----:|------|
| 0 | ✅ 通过 | ✅ 合并 | `pr +merge` |
| 0 | ⚠️ 无 CI | ✅ 合并（弱） | 标注"无 CI"后合并 |
| > 0 | 任意 | ❌ 驳回 | 评论修改建议 |
| 任意 | ❌ 失败 | ❌ 驳回 | 评论 + CI 日志 |

---

## 最终输出

四个子 Skill 执行完毕后，汇总输出：

```markdown
## 📋 质量门禁报告 — PR #<id>
| 门禁项 | 状态 | 详情 |
|--------|:----:|------|
| 🔍 代码审查 | ✅/❌ | Critical: N, Warning: N |
| 🔧 CI 构建 | ✅/❌/⚠️ | <摘要> |
| 📋 最终判定 | 通过/驳回 | <理由> |
```
