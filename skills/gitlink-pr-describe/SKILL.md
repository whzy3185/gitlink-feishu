---
name: gitlink-pr-describe
version: 1.0.0
description: "PR 自动描述生成：获取 PR 的 diff/变更文件，AI 按规范结构（背景/改动/测试/影响）自动生成 PR 描述，可写入 PR body。当用户提 PR 不知怎么写描述、或想规范化 PR 时触发。任务二创新增强 Skill。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-pr-describe（PR 自动描述生成 · 创新增强 Skill）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — 写入 PR 描述前确认用户意图。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务二**创新增强** Skill。写 PR 描述是开发者的日常痛点（不知写啥/不规范）。本 Skill 获取 PR diff，AI 按规范结构**自动生成 PR 描述**，提升 PR 质量和评审效率。

---

## 解决的痛点
- 开发者提 PR 时描述写不全/不规范 → 评审难理解
- 手写描述费时 → 效率低

## 工作流

### Step 1：获取 PR 变更
```bash
gitlink-cli pr +view --owner <o> --repo <r> --id <pr_id> --format json   # PR 基本信息
gitlink-cli pr +files --owner <o> --repo <r> --id <pr_id> --format json  # 变更文件
gitlink-cli pr +diff --owner <o> --repo <r> --id <pr_id> --format json   # diff 内容
```

### Step 2：AI 分析 diff + 生成描述
AI 从 diff 提炼，按规范结构生成：
- **背景**：为什么改（从 commit message/改动推断）
- **改动**：改了什么（按文件/功能分组）
- **测试**：怎么验证（从测试文件改动推断）
- **影响**：影响范围（哪些功能/模块）
- **类型**：feat/fix/docs/refactor（Conventional Commits）

### Step 3：输出/写入 PR 描述
```markdown
## PR 描述（AI 生成）— #<id> <title>

### 背景
<为什么改>

### 改动
- <文件/功能1>：<具体改动>
- <文件/功能2>：<具体改动>

### 测试
- <如何验证>

### 影响
- 影响范围：<模块>
- 类型：feat/fix/...

### 关联 Issue
fixes #<n>
```

可选：通过 Raw API 写入 PR body（需确认）。

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| diff 太大 | 按文件分段处理，取关键改动 |
| 自动描述需人工校对 | 标注"AI 生成，请校对" |
| 写入 PR body 需 PATCH | `api PATCH /v1/<o>/<r>/pulls/<id>` + body |
| commit message 是好的素材 | 结合 commit message + diff 双源生成 |

---

## 实测落地参考
**gitlink/gitlink-cli 某 PR**（如 wiki +list shortcut）：
- diff：新增 shortcuts/wiki/wiki.go + wiki_test.go + register.go 注册
- AI 生成描述：
  - 背景：补全 wiki 知识库管理命令（PDF 任务一要求）
  - 改动：新增 wiki +list/+view/+create/+update/+delete 5 命令 + 测试 + 注册
  - 测试：`go test ./shortcuts/wiki/`
  - 影响：新增 wiki 模块，不影响现有
  - 类型：feat

详见 verification.md。
