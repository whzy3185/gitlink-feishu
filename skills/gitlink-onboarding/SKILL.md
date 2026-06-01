---
name: gitlink-onboarding
version: 1.0.0
description: "新人入门引导：帮助新贡献者发现适合入门的 Issue、了解项目贡献流程。当用户想参与项目贡献但不知从何入手、寻找入门任务、或询问如何开始贡献代码时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-onboarding（新人引导）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读操作，不会修改仓库任何内容。无需用户额外确认即可执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

帮助新贡献者快速了解项目并找到适合入门的任务：

1. **项目总览** — 获取仓库基本信息（语言、分支、贡献者规模）
2. **入门 Issue 发现** — 从开放 Issue 中筛选适合新手的任务
3. **贡献指南** — 生成 Fork → Branch → PR 的完整操作步骤

---

## 工作流：新人任务发现与引导

### Step 1：获取项目概览

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

从返回数据中提取：

| 字段 | 用途 |
|------|------|
| `full_name` | 确认仓库正确 |
| `default_branch` | 后续分支操作的目标分支（GitLink 通常是 `master`） |
| `contributor_users_count` | 判断社区活跃度 |
| `issues_count` | 了解任务池大小 |
| `size` | 判断项目规模 |
| `description` | 了解项目用途 |

### Step 2：扫描开放 Issue

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
```

返回的 Issue 数组中，关注以下字段：
- `subject` — Issue 标题
- `project_issues_index` — Issue 编号（用于 `+view` 的 `--number` 参数）
- `status_id` — 状态（1=新增, 2=正在解决, 3=已解决, 5=关闭）
- `tags` — 标签数组，每个元素含 `name` 字段
- `assigners` — 已分配人（空数组 = 无人认领）
- `priority` — 优先级（null 或 `{"name": "正常"/"紧急"/...}`）
- `created_at` / `updated_at` — 时间信息

> ⚠️ **已知问题**：`--state open` 过滤不准确，返回列表可能包含已关闭的 Issue。需要在客户端按 `status_id` 过滤：仅保留 `status_id` 为 1（新增）或 2（正在解决）。

### Step 3：预过滤 + 筛选入门级 Issue

**第一步：客户端状态过滤**

忽略 `--state` 参数的实际效果，从返回结果中手动过滤：
- 保留：`status_id` = 1（新增）或 2（正在解决）
- 排除：`status_id` = 3（已解决）、5（关闭）
- `status_id` = 0（未知状态）：可纳入候选但需特别标注"状态未知，建议先评论确认"

**第二步：根据标签/标题筛选入门级 Issue**

标签匹配（优先级从高到低）：
1. 标签名含 `good first issue`、`good-first-issue` → 官方标记的入门任务
2. 标签名含 `help wanted`、`help-wanted` → 维护者明确求帮助
3. 标签名含 `easy`、`beginner`、`新手`、`入门`、`低难度` → 社区约定的简单任务
4. 标签名含 `bug`、`fix` 且标题含 `修复`、`fix` → 修复类任务通常范围明确
5. 标签名含 `documentation`、`docs`、`文档` → 文档类任务对新手友好

辅助判断（无标签时）：
- `assigners` 为空 → 无人认领
- `priority` 为 null 或 `name` = "正常" → 不紧急
- 标题含 `优化`、`改进`、`添加`、`新增` → 可能是功能增强，范围弹性大

过滤规则：
- 已分配（`assigners` 非空）→ 排除（除非标签明确是 `help wanted`）
- 标题含 `紧急`、`hotfix`、`安全` → 排除（不适合新手）

### Step 4：深入查看候选 Issue

对筛选出的每个候选 Issue（建议 3~5 个），获取详情：

```bash
gitlink-cli issue +view --owner <owner> --repo <repo> --number <project_issues_index> --format json
```

从返回数据中确认：
- `description` — 任务描述是否清晰、有可执行的步骤
- `comment_journals_count` — 是否有讨论历史（有讨论 = 需求更明确）
- `start_date` / `due_date` — 是否有时间限制

### Step 5：生成新人引导报告

将所有信息组织为以下格式输出。

---

## 输出模板

```markdown
# 🚀 {{仓库名}} 新人引导报告

## 项目概览

| 项目 | 信息 |
|------|------|
| 仓库 | {{full_name}} |
| 描述 | {{description}} |
| 主分支 | {{default_branch}} |
| 贡献者数 | {{contributor_users_count}} |
| 开放 Issue | {{issues_count 或实际列表长度}} |
| 项目规模 | {{size}} |

---

## 📋 推荐的入门 Issue

> 以下 Issue 适合新贡献者，按推荐度排序。

### ⭐ 最推荐（官方标记/明确入门级）

| # | 标题 | 标签 | 推荐理由 |
|---|------|------|----------|
| {{number}} | {{subject}} | {{tags}} | good first issue 官方标记 |
| ... | ... | ... | ... |

### 👍 推荐（文档/简单修复）

| # | 标题 | 标签 | 推荐理由 |
|---|------|------|----------|
| {{number}} | {{subject}} | {{tags}} | 文档类任务，无需深入代码 |
| ... | ... | ... | ... |

### 🤔 可尝试（功能增强，范围需确认）

| # | 标题 | 标签 | 推荐理由 |
|---|------|------|----------|
| {{number}} | {{subject}} | {{tags}} | 功能明确，建议先评论确认范围 |
| ... | ... | ... | ... |

---

## 📖 贡献流程

### 第 1 步：Fork 仓库

在 GitLink 网页打开 https://www.gitlink.org.cn/{{owner}}/{{repo}} ，点击右上角 **Fork** 按钮。

### 第 2 步：Clone 到本地

```bash
git clone https://www.gitlink.org.cn/<你的用户名>/{{repo}}.git
cd {{repo}}
git remote add upstream https://www.gitlink.org.cn/{{owner}}/{{repo}}.git
```

### 第 3 步：创建分支

```bash
git checkout -b fix/issue-{{number}}-简要描述
```

### 第 4 步：修改 + 提交

```bash
git add -A
git commit -m "fix: 简要描述修改内容 (#{{number}})"
```

### 第 5 步：Push 并提 PR

```bash
git push origin fix/issue-{{number}}-简要描述
gitlink-cli pr +create \
  --owner {{owner}} \
  --repo {{repo}} \
  --head <你的用户名>:fix/issue-{{number}}-简要描述 \
  --base {{default_branch}} \
  --title "fix: 简要描述 (#{{number}})"
```

### 第 6 步：在 Issue 下留言

在 Issue 页面评论说明你正在处理，避免与他人重复劳动。如果 Issue 已有讨论，先阅读并确认无人认领。

---

## ⚠️ 注意事项

1. **先评论再动手** — 在 Issue 下留言「我来处理这个」，避免重复劳动
2. **保持 PR 小** — 一个 PR 只解决一个问题，方便维护者 Review
3. **阅读贡献指南** — 如果仓库有 `CONTRIBUTING.md`，先阅读
4. **不确定就问** — 对需求有疑问，在 Issue 下直接提问
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 无开放 Issue | 输出项目概览后，建议用户关注 `watch` 仓库等待新 Issue，或查看已有 PR 了解贡献模式 |
| 所有 Issue 已分配 | 列出已分配 Issue，建议用户在感兴趣的 Issue 下评论询问是否需要帮助 |
| 无入门级标签 | 列出所有未分配的开放 Issue，标注「无明确入门标记，建议根据兴趣自行选择」，优先推荐标题含 `fix`/`doc`/`优化` 的 |
| `repo +info` 返回空 | 检查 owner/repo 是否正确，提示用户确认仓库名 |

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **`issue +view` 使用 `--number`（网页编号）**，非数据库 ID
- ✅ **本 Skill 为纯只读**，不会修改仓库
- ✅ **Owner/repo 优先从 `git remote` 自动解析**，无 git 上下文时询问用户
- ⚠️ **Issue 列表可能分页**，如果总数 >20，需要追加 `--page 2` 等参数获取全部
- ⚠️ **`issue +list --state open` 过滤不准确**，返回列表可能含已关闭 Issue。必须客户端按 `status_id` 二次过滤（保留 1、2，排除 3、5、0）
- ⚠️ **此仓库未使用 Issue 标签系统**，筛选主要依赖标题关键词和 `assigners` 状态。如目标仓库有标签，优先使用标签匹配
