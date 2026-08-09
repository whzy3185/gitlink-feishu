---
name: gitlink-commit-check
version: 1.0.0
description: "提交规范检查：检查 commit message 是否符合 Conventional Commits（feat/fix/docs 等），识别不规范提交并给修复建议。当团队需要规范提交流程、审查提交质量时触发。任务二创新增强 Skill。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-commit-check（提交规范检查 · 创新增强 Skill）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读检查 + 建议，不改写历史。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务二**创新增强** Skill。规范的 commit message 是协作基础（影响 Release Notes 自动生成、PR 审查）。本 Skill 检查 commit 是否符合 **Conventional Commits**（feat/fix/docs/refactor/test/chore），识别不规范提交，给修复建议。配合平台的 `gitlink-commit-quality` Skill，强化提交质量。

---

## 解决的痛点
- commit message 不规范（如"测试流水线""1""修改"）→ Release Notes 难生成、历史难读
- 团队无统一规范 → 协作混乱

## 规范模型（Conventional Commits）

```
<type>(<scope>): <subject>

类型 type（必须合法）：
feat / fix / docs / style / refactor / perf / test / chore / ci
```

| 检查项 | 标准 | 不规范示例 |
|--------|------|----------|
| 格式 | `type: subject` | "测试流水线"、"1"、"修改" |
| type 合法 | feat/fix/docs/... | "新增 xxx"（缺 type）|
| subject 清晰 | 描述具体改动 | "改了下"、"update" |

## 工作流

### Step 1：采集 commit
```bash
git log --format="%h %s" -<N>                 # 近 N 条 commit message
# 或检查某范围：git log <from>..<to> --format="%h %s"
```

### Step 2：AI 逐条检查
对每条 commit，检查：
- 格式是否符合 `type(scope): subject`
- type 是否合法
- subject 是否清晰

### Step 3：输出检查报告 + 修复建议
```markdown
## 📝 提交规范检查 — 近 N 条

### ✅ 规范（X 条）
- abc1234 feat: add wiki +list shortcut
- def5678 fix: correct label API path

### ⚠️ 不规范（Y 条）
| commit | message | 问题 | 建议 |
|--------|---------|------|------|
| xyz | 测试流水线 | 缺 type | → chore: 测试流水线触发 |
| xyz | 1 | 无意义 | → 补充描述 |
| xyz | 增加label标签管理 | 缺 type 前缀 | → feat: 增加 label 标签管理 |

### 规范率：X/(X+Y) = N%
### 建议：<针对性改进>
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| merge commit 干扰 | 过滤掉 `Merge` 开头的 |
| 中文 commit | type 用英文（feat/fix），subject 可中文 |
| 不改写历史 | 只检查+建议，不用 rebase 改历史（风险）|

---

## 实测落地参考
**gitlink/gitlink-cli commit 检查**（部分历史）：
- ✅ 规范：`feat: add wiki +list shortcut` / `fix: correct label API path` / `feat(skills): 新增...`
- ⚠️ 不规范：`测试流水线`（缺type）→ 建议 `chore: 测试流水线` / `1`（无意义）→ 补充 / `增加label标签管理`（缺type）→ `feat: 增加 label 标签管理`

**规范率**：约 85%（少数测试提交不规范，可改进）。

详见 verification.md。
