---
name: gitlink-onboarding
version: 1.0.0
description: "新人引导：为开源项目新贡献者提供从环境搭建到首次提交的完整引导。当用户提到「新人引导」「新手入门」「good first issue」「贡献指南」「如何参与」「onboarding」等场景时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli --help"
---

# gitlink-onboarding（新人引导）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 工作流概览

本 Skill 为开源项目新贡献者提供从零到一的完整引导体验，涵盖环境搭建、项目理解、Issue 选择、代码修改到提交 PR 的全过程。

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| ① 项目概览 | 拉取仓库信息、README、目录结构 | 执行 CLI 命令采集项目信息 |
| ② 环境搭建 | 引导安装依赖、配置开发环境 | 根据项目类型生成环境搭建指南 |
| ③ 寻找任务 | 搜索 good-first-issue 标签的 Issue | 推荐、筛选适合新人的 Issue |
| ④ 代码引导 | 分析 Issue 对应的代码位置 | 生成代码定位和修改指引 |
| ⑤ 提交贡献 | Fork → Branch → Commit → PR | 引导完成 Fork 工作流 |
| ⑥ 发布引导评论 | 在 Issue 中添加新人引导评论 | 自动生成个性化引导内容 |

---

## 详细工作流

### 工作流 1：项目新人入门（Project Onboarding）

**场景**：新人想要参与一个 GitLink 项目，需要了解项目信息和上手指南。

#### Step 1：获取项目概览

```bash
# 获取仓库基本信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 获取 README 内容
gitlink-cli repo +readme --owner <owner> --repo <repo>

# 获取语言统计
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json

# 获取贡献者列表
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json

# 获取目录结构（查看 src 目录）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=src&ref=master'
```

#### Step 2：生成环境搭建指南

根据项目的语言和技术栈，AI 生成对应的环境搭建指南：

**Go 项目模板：**
```markdown
## 🚀 环境搭建指南

### 前置要求
- Go 1.21+
- Git
- gitlink-cli（已安装）

### 步骤
1. Fork 项目：`gitlink-cli repo +fork --owner <owner> --repo <repo>`
2. Clone 你的 Fork：`git clone https://www.gitlink.org.cn/<you>/<repo>.git`
3. 添加 upstream：`git remote add upstream https://www.gitlink.org.cn/<owner>/<repo>.git`
4. 安装依赖：`go mod download`
5. 验证构建：`go build ./...`
6. 运行测试：`go test ./...`
```

**Python 项目模板：**
```markdown
## 🚀 环境搭建指南

### 前置要求
- Python 3.10+
- Git
- gitlink-cli（已安装）

### 步骤
1. Fork 项目：`gitlink-cli repo +fork --owner <owner> --repo <repo>`
2. Clone 你的 Fork：`git clone https://www.gitlink.org.cn/<you>/<repo>.git`
3. 创建虚拟环境：`python -m venv venv && source venv/bin/activate`
4. 安装依赖：`pip install -e ".[dev]"`
5. 运行测试：`pytest tests/`
```

#### Step 3：输出项目结构分析

AI 根据仓库信息和目录结构，输出项目概览报告：

```markdown
## 📋 项目概览 — <owner>/<repo>

| 信息 | 详情 |
|------|------|
| 项目名称 | <name> |
| 描述 | <description> |
| 主要语言 | <language> |
| 开源协议 | <license> |
| 贡献者数 | <count> |
| 开放 Issue | <count> |
| 开放 PR | <count> |

### 📁 核心目录
- `src/` — 源代码
- `tests/` — 测试
- `doc/` — 文档
- `cmd/` — CLI 入口

### 🤝 贡献流程
1. Fork → Branch → Code → Test → PR
2. 遵循 Conventional Commits 规范
3. PR 需要通过 CI 检查和 Code Review
```

---

### 工作流 2：寻找适合新人的 Issue

**场景**：新人不知道从哪里入手，需要推荐适合新手的任务。

#### Step 1：搜索 good-first-issue

```bash
# 搜索带 good-first-issue 标签的 Issue
gitlink-cli search +issues --owner <owner> --repo <repo> --keyword "good first issue" --category opened

# 查看所有打开的 Issue
gitlink-cli issue +list --state open --format json

# 获取标签列表（寻找新人友好标签）—— 标签查询暂未封装 Shortcut，用 Raw API
gitlink-cli api GET /v1/<owner>/<repo>/issue_tags --query 'page=1&limit=50'
```

#### Step 2：分析 Issue 新人友好度

AI 对每个开放的 Issue 进行新人友好度评估：

| 评估维度 | 高友好 ✅ | 中友好 🟡 | 低友好 🔴 |
|---------|----------|----------|----------|
| 标题清晰度 | 明确描述问题和期望 | 模糊但可理解 | 标题不清 |
| 描述完整度 | 有复现步骤、预期结果 | 有简要描述 | 只有标题 |
| 代码定位 | 标注了文件/函数 | 可推断位置 | 无任何定位信息 |
| 改动范围 | 单文件、<50 行 | 多文件或 >50 行 | 涉及架构改动 |
| 难度标签 | good-first-issue / easy | medium | hard / critical |

#### Step 3：推荐 Issue 列表

```markdown
## 🎯 推荐新手任务

### ⭐ 强烈推荐（新人友好度：⭐⭐⭐）

1. **Issue #<n>** — <title>
   - 📁 涉及文件：`<file_path>`
   - 📝 改动范围：约 <n> 行
   - 💡 提示：<具体修改建议>
   - 🔗 链接：https://www.gitlink.org.cn/<owner>/<repo>/issues/<n>

### ✅ 值得尝试（新人友好度：⭐⭐）

2. **Issue #<n>** — <title>
   - 📝 需要了解：<相关知识>
   - 💡 提示：<学习建议>
```

---

### 工作流 3：Issue 引导评论生成

**场景**：项目维护者希望为 good-first-issue 自动生成引导评论，帮助新人快速上手。

#### Step 1：获取 Issue 详情

```bash
# 查看 Issue 详情
gitlink-cli issue +view --number <issue_number> --format json

# 获取相关文件内容（用于代码定位）—— 原始文件读取暂未封装 Shortcut，用 Raw API
gitlink-cli api GET /<owner>/<repo>/raw/master/<file_path>
```

#### Step 2：生成引导评论

AI 根据 Issue 内容生成结构化的引导评论：

```markdown
## 🌟 欢迎贡献！

感谢你对本项目的关注！这是一个 **good first issue**，非常适合首次贡献者。

### 📋 任务描述
<用自己的话重述 Issue 内容>

### 🗺️ 代码定位
- 需要修改的文件：`<file_path>`
- 相关函数/类：`<function_name>`（第 <n> 行附近）
- 依赖的上下文：`<related_file>`

### ✏️ 修改步骤
1. **Fork 项目**
   ```bash
   gitlink-cli repo +fork --owner <owner> --repo <repo>
   ```
2. **创建分支**
   ```bash
   git checkout -b fix/<branch-name>
   ```
3. **定位代码**
   - 打开 `<file_path>`
   - 找到 `<function_name>` 函数
   - 理解当前逻辑：<简要说明>
4. **实施修改**
   - <具体修改步骤>
   - 预期改动约 <n> 行
5. **测试验证**
   ```bash
   go test ./<package>/...  # 或 pytest tests/
   ```
6. **提交 PR**
   ```bash
   git add .
   git commit -m "fix: <commit-message>"
   git push origin fix/<branch-name>
   gitlink-cli pr +create --owner <owner> --repo <repo> \
     --head <you>:fix/<branch-name> --base master \
     --title "fix: <title>"
   ```

### 💡 提示
- 不确定的地方可以先在 Issue 中提问
- PR 描述中引用本 Issue：`Fixes #<number>`
- 遵循项目的代码风格和提交规范

### ❓ 需要帮助？
如果遇到任何问题，请随时在下方评论，维护者会尽快回复！
```

#### Step 3：发布引导评论

```bash
# 将引导评论发布到 Issue
gitlink-cli issue +comment \
  --number <issue_number> \
  --body "$(cat <<'EOF'
## 🌟 欢迎贡献！
...引导内容...
EOF
)"
```

---

### 工作流 4：新人贡献全流程引导

**场景**：新人已选定 Issue，需要从 Fork 到提交 PR 的全流程指导。

```bash
# Step 1：Fork 仓库
gitlink-cli repo +fork --owner <owner> --repo <repo>

# Step 2：Clone Fork
git clone https://www.gitlink.org.cn/<you>/<repo>.git
cd <repo>

# Step 3：添加 upstream
git remote add upstream https://www.gitlink.org.cn/<owner>/<repo>.git

# Step 4：创建分支
git checkout -b fix/<issue-descriptor>

# Step 5：（用户进行代码修改）

# Step 6：提交
git add -A
git commit -m "fix: <description> (#<issue_number>)"

# Step 7：推送到 Fork
git push origin fix/<issue-descriptor>

# Step 8：创建 PR
gitlink-cli pr +create \
  --owner <owner> --repo <repo> \
  --head <you>:fix/<issue-descriptor> --base master \
  --title "fix: <title>" \
  --body "## 变更说明\n\nFixes #<issue_number>\n\n### 修改内容\n- ...\n\n### 测试\n- [ ] 单元测试通过\n- [ ] 手动验证"
```

---

## 新人友好度评估标准

用于评估项目是否对新人友好：

| 维度 | 评估方法 | 数据来源 |
|------|---------|---------|
| README 完整性 | README 是否包含项目介绍、安装步骤、贡献指南 | `repo +readme` |
| Issue 标签 | 是否有 good-first-issue / easy 标签 | `api GET /v1/:owner/:repo/issue_tags` |
| 文档覆盖 | 是否有 Wiki、API 文档 | `wiki +list` |
| CI 配置 | 是否有自动化构建和测试 | `.gitea/workflows/` 或 `.github/workflows/` |
| 维护者响应 | Issue 平均响应时间 | `issue +list` + 创建时间分析 |
| 贡献指南 | 是否有 CONTRIBUTING.md | `api GET /:owner/:repo/raw/master/CONTRIBUTING.md` |

---

## 输出模板

### 项目新手上手指南

```markdown
# 🚀 <项目名> 新人上手指南

## 1. 了解项目
<项目简介 + 技术栈>

## 2. 环境搭建
<Step-by-step 安装指南>

## 3. 项目结构
<目录说明 + 核心模块>

## 4. 选择任务
<推荐 Issue 列表>

## 5. 开始贡献
<Fork → Branch → Code → PR 流程>

## 6. 获取帮助
<社区链接 / 维护者联系 / 文档>
```

---

## 决策规则

| 条件 | 操作 |
|------|------|
| 项目无 README | 提示维护者补充 README，但仍提供基础引导 |
| 无 good-first-issue 标签 | 从开放的 Issue 中推荐最简单的（标题包含"文档""修复""小"） |
| Issue 无描述 | 提示用户先在 Issue 中提问获取更多信息 |
| 用户未登录 | 引导执行 `gitlink-cli auth login` |
| 用户无 Fork | 引导执行 Fork 流程 |
| Fork 已存在但未配置 upstream | 引导添加 upstream remote |

---

## 注意事项

- 引导评论发布前确认用户意图（维护者模式）
- 推荐的 Issue 应标注预估改动范围和难度
- Fork 工作流严格遵循 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 中的 PR 协作流程
- 不确定的信息（如代码定位）应明确标注"建议确认"
- 遵循项目的贡献规范（如果存在 CONTRIBUTING.md）
- 所有 CLI 命令使用 `--format json` 以便解析
