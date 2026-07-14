---
name: gitlink-onboarding
version: 1.0.0
description: "新人引导：自动识别适合新贡献者的 Issue，打 good-first-issue 标签、生成个性化引导评论和项目入门指南，降低参与门槛。当用户需要管理 good-first-issue、帮助新人上手项目或提升社区友好度时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-onboarding（新人引导）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前（打标签、写评论），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 工作流概览

| 工作流 | 操作 | AI Agent 角色 | 写入 |
|--------|------|--------------|:----:|
| 工作流 1：good-first-issue 自动标记 | 扫描开放 Issue → 识别适合新人的 → 打标签 | 判断复杂度 + 创建标签 | 是 |
| 工作流 2：引导评论生成 | 对新人 Issue 写个性化引导评论 | 生成评论内容 | 是 |
| 工作流 3：项目入门指南 | 生成项目贡献入门文档 | 分析仓库 + 生成指南 | 否 |

---

## 新人友好 Issue 识别标准

AI 判断一个 Issue 是否适合新人时，参考以下信号（满足越多越适合）：

| 信号 | 说明 | 权重 |
|------|------|:----:|
| 标题关键词 | typo / 文档 / 简单 / 拼写 / 入门 / good first issue / help wanted | 高 |
| 改动范围 | 单文件、文案/文档类、配置类 | 高 |
| 描述清晰度 | 有明确预期结果和复现步骤 | 中 |
| 不涉及核心 | 不触碰核心架构、复杂业务逻辑、并发/安全 | 高 |
| 已有友好标签 | 已标记 question / 文档 / 协助 | 中 |

**排除标准（不建议标记为新人 Issue）：**
- 性能优化、安全漏洞、架构重构
- 描述模糊、无法复现、信息严重不足
- 涉及 CI/CD、部署、数据库迁移

---

## 工作流 1：good-first-issue 自动标记

**触发场景：** "帮我找出适合新人的 Issue 并打上标签"

### Step 1：获取开放 Issue

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --format json
```

### Step 2：AI 判断哪些 Issue 适合新人

遍历每个 Issue，按"新人友好识别标准"评估，筛选出适合新人的 Issue 列表。

### Step 3：查找或创建 good-first-issue 标签

```bash
# 先查现有标签，看是否已有 good first 标签
gitlink-cli label +list --owner <owner> --repo <repo> --format json

# 若不存在，创建（颜色遵循社区惯例 #7057ff）
# 注意：GitLink 标签名限 15 字符，"good first issue"(16字符) 会被截断，用 "good first"
gitlink-cli label +create --owner <owner> --repo <repo> \
  --name "good first" --color "#7057ff"
```

### Step 4：自动打标签

```bash
# 注意：--label 是"设置"语义（覆盖），需保留原标签时一并传入
gitlink-cli issue +update --owner <owner> --repo <repo> \
  --number <n> --label <good_first_issue_id>,<原标签id>
```

### Step 5：输出标记报告

```markdown
## 🌱 新人友好 Issue 标记报告 — <owner>/<repo>

| Issue | 标题 | 适合新人理由 | 已打标签 |
|-------|------|------------|:--------:|
| #5 | 修复 README 拼写错误 | 单文件文案修改，范围明确 | ✅ |
| #3 | 补充 API 文档示例 | 文档类，预期清晰 | ✅ |

共标记 2 个 good-first-issue。
```

---

## 工作流 2：引导评论生成

**触发场景：** "帮我给适合新人的 Issue 写引导评论"

### Step 1：获取 Issue 详情

```bash
gitlink-cli issue +view --owner <owner> --repo <repo> --number <n> --format json
```

### Step 2：AI 生成个性化引导评论

根据 Issue 内容，生成包含以下要素的引导评论（**不要用固定模板**，要结合具体 Issue）：

- **任务说明**：用一句话概括要做什么
- **相关文件**：定位到具体文件/目录（结合仓库结构推断）
- **本地准备**：克隆仓库、切换分支、运行测试的命令
- **提交指引**：提交 PR 的步骤和规范

### Step 3：发布评论

```bash
gitlink-cli issue +comment --owner <owner> --repo <repo> \
  --number <n> --body "欢迎贡献！这是一个适合新人的任务..."
```

### 引导评论输出模板

```markdown
👋 欢迎贡献！这个 Issue 适合新人入手。

**任务目标：** <一句话说明>

**建议入手位置：**
- 相关文件：`<path/to/file>`
- 主要改动：<具体位置>

**本地准备：**
1. Fork 并克隆仓库
2. 安装依赖并确认能本地运行
3. 创建分支：git checkout -b fix/<简述>

**提交 PR：** 改动完成后提交 PR，关联本 Issue。

有任何问题欢迎在下方留言，社区会及时回复 🤝
```

---

## 工作流 3：项目入门指南

**触发场景：** "帮我生成一份新人入门指南"

### Step 1：获取仓库信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo>
```

### Step 2：AI 生成入门指南

综合仓库信息，生成结构化的入门文档：

```markdown
## 🚀 <项目名> 新人入门指南

### 环境准备
- <语言/运行时版本要求>
- <依赖安装方式>

### 项目结构速览
| 目录 | 作用 |
|------|------|
| <dir> | <说明> |

### 第一个贡献
1. 从带 good-first-issue 标签的 Issue 入手
2. <项目特定的开发流程>

### 提交规范
- <commit message 规范>
- <PR 流程>
```

> 该指南可直接展示给用户，或交给 `gitlink-docs-assistant` Skill 写入 Wiki。

---

## Raw API 参考

```bash
# 获取某个 issue 的完整信息
gitlink-cli api GET /:owner/:repo/issues/:number --format json

# 查询可分配的负责人（帮助新人 Issue 找导师）
gitlink-cli issue +assigners --owner <owner> --repo <repo> --format json
```

---

## 注意事项

- **写操作前确认：** 打标签（`issue +update --label`）和写评论（`issue +comment`）会真实修改仓库，执行前务必让用户确认。
- **--label 是覆盖语义：** `issue +update --label <id>` 会替换原有标签。若 Issue 已有标签，需把原标签 ID 一并传入（如 `--label 350191,327271`），否则原标签会丢失。
- **标签颜色格式：** `label +create --color` 必须带 `#` 号（如 `#7057ff`）。
- **评论要个性化：** 避免对所有 Issue 使用同一句引导文案，应结合具体 Issue 内容生成。
- **--number 是网页编号：** `--number` 用 GitLink 网页 URL 中显示的编号，对新人最直观。
- **标签名长度限制：** GitLink 标签名上限 **15 字符**，"good first issue"（16字符）会被截断，建议用 "good first" 或中文"新人入门"。若已创建被截断，可用 `label +update --id <id> --name "good first"` 修正。
- **good-first-issue 颜色惯例：** 社区通用 `#7057ff`（GitHub 惯例），保持一致便于识别。
