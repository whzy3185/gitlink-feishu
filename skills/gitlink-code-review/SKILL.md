---
name: gitlink-code-review
version: 1.0.0
description: "智能代码审查：获取 PR 变更、分析代码质量、自动生成 Review 评论与摘要报告。当用户需要审查 Pull Request、检查代码质量或生成审查报告时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-code-review（智能代码审查）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 工作流概览

本 Skill 提供一套完整的 AI 驱动代码审查工作流，覆盖从获取 PR 变更到生成审查报告的全过程。不需要额外的 CLI Shortcuts——现有 `gitlink-cli` 命令 + AI Agent 的分析能力即可完成。

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| ① 获取上下文 | 拉取 PR 详情、变更文件、Diff | 执行 CLI 命令采集数据 |
| ② 分析代码 | 检查每个文件的变更 | 逐文件审查，标记问题 |
| ③ 结构化反馈 | 按严重程度分级输出审查意见 | 生成分级 Review 评论 |
| ④ 提交评论 | 发表 Review 到 PR | 通过 API 提交 |
| ⑤ 生成报告 | 输出审查摘要 | 生成 Markdown 摘要 |

---

## 详细工作流

### 工作流 1：PR 代码审查

**场景**：收到 PR Review 请求后，进行完整代码审查。

#### Step 1：获取 PR 上下文

```bash
# 获取 PR 详情
gitlink-cli pr +view --id <pr_id> --format json

# 获取变更文件列表
gitlink-cli pr +files --id <pr_id> --format json

# 获取 Diff 内容（含变更行号和代码上下文）
gitlink-cli pr +diff --id <pr_id> --format json
```

#### Step 2：逐文件分析

对每个变更文件，根据文件类型执行针对性检查：

**Python 文件检查项：**
- 语法与导入：未使用的 import、循环导入、wildcard import
- 代码规范：PEP 8 风格偏离、过长行（>88 chars）、命名规范
- 安全：硬编码密钥、SQL 注入风险、`eval()`/`exec()` 使用
- 性能：不必要的循环、缺少缓存、N+1 查询
- 错误处理：裸 `except`、吞异常、缺少 finally

**JavaScript/TypeScript 文件检查项：**
- 安全：`innerHTML` 直接赋值、`eval()` 使用
- 类型安全：`any` 滥用、缺失类型定义
- 性能：不必要的 re-render、大对象深拷贝
- 异步：未处理的 Promise、缺少 error boundary
- 依赖：已废弃 API 使用

**Go 文件检查项：**
- 错误处理：未检查的 error return、panic 滥用
- 并发：goroutine 泄漏、缺少 sync 保护
- 资源管理：未关闭的 file/conn、defer 使用
- 命名：导出标识符缺少注释、变量 shadowing

**通用检查项：**
- 硬编码的配置值、密钥、URL
- 缺少或错误的边界条件检查
- 过于复杂的函数（圈复杂度高）
- 魔法数字（未命名的常量）
- 重复代码（DRY 违反）
- 缺少或过时的注释
- 测试覆盖不足

#### Step 3：生成结构化审查结果

按以下 Severity 分级输出：

```markdown
## PR #<id> 代码审查报告

### 🔴 Critical（必须修改）
- <问题描述> — <文件>:<行号>
  > <修改建议>

### 🟡 Warning（建议修改）
- <问题描述> — <文件>:<行号>
  > <修改建议>

### 🔵 Suggestion（可选优化）
- <问题描述> — <文件>:<行号>
  > <修改建议>

### ✅ Positive（值得肯定）
- <做得好的地方>
```

#### Step 4：提交 Review 评论

```bash
# 方式 1：提交整体 Review
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{
  "body": "## 审查结果\n\n### 🔴 Critical\n...\n\n### 🟡 Warning\n...\n\n总体评价：...",
  "event": "COMMENT"
}'

# 方式 2：在特定行添加内联评论（逐条提交）
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{
  "body": "这里存在安全风险：用户输入未经转义直接拼接到 SQL 查询中，存在注入风险。建议使用参数化查询。",
  "event": "COMMENT",
  "commit_id": "<commit_sha>",
  "path": "src/query.py",
  "position": 42
}'
```

> **注意：** `event` 参数支持 `COMMENT`（普通评论）和 `APPROVE`（批准）。对于需要修改的问题，使用 `COMMENT`。

#### Step 5：生成审查摘要

审查完成后，输出 Markdown 摘要供用户查阅：

```markdown
## 📋 审查摘要 — PR #<id> <title>

| 指标 | 数据 |
|------|------|
| 审查文件数 | <n> |
| 变更行数 | +<add> / -<del> |
| Critical 问题 | <n> |
| Warning | <n> |
| Suggestion | <n> |

### 主要发现
1. **[Critical]** <最严重的问题>
2. **[Warning]** <次要问题>
3. **[Suggestion]** <优化建议>

### 总体评价
<整体评估：代码质量、审查通过建议>

---
*由 gitlink-code-review Skill 自动生成*
```

---

### 工作流 2：仓库代码健康度扫描

**场景**：对仓库整体代码质量进行评估，不依赖 PR。

```bash
# 1. 获取仓库信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 2. 获取仓库文件列表（遍历关键目录）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=src&ref=master'
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=tests&ref=master'

# 3. 获取关键文件内容
gitlink-cli api GET /:owner/:repo/raw/master/README.md
gitlink-cli api GET /:owner/:repo/raw/master/.gitignore
gitlink-cli api GET /:owner/:repo/raw/master/.eslintrc.js  # 或类似配置
gitlink-cli api GET /:owner/:repo/raw/master/package.json  # 或 go.mod, Cargo.toml

# 4. 获取语言统计和贡献者
gitlink-cli api GET /:owner/:repo/languages
gitlink-cli api GET /:owner/:repo/contributors
```

**健康度检查清单：**

| 检查项 | 标准 | 评分依据 |
|--------|------|----------|
| 文档完整性 | 有 README、CONTRIBUTING、CHANGELOG | 文件是否存在、内容质量 |
| 许可证 | 有 LICENSE 文件 | 是否存在、是否合规 |
| CI 配置 | 有 CI 配置（.github/workflows, Jenkinsfile 等） | 文件是否存在 |
| 代码规范 | 有 linter 配置 | eslint/prettier/ruff/pylint 等 |
| 测试覆盖 | 有 test 目录或测试文件 | 测试文件比例 |
| 依赖管理 | 依赖文件完整且无已知漏洞 | package-lock/go.sum/poetry.lock |
| Issue 健康度 | Issue 有分类标签、响应及时 | 通过 Issue 列表分析 |

**输出格式：**

```markdown
## 🏥 仓库健康度报告 — <owner>/<repo>

### 总体评分：<⭐x/5>

| 维度 | 状态 | 评分 | 建议 |
|------|:----:|:----:|------|
| 📖 文档 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 📜 许可证 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 🔧 CI/CD | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 🎨 代码规范 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 🧪 测试覆盖 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 📦 依赖安全 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |
| 🐛 Issue 管理 | ✅/⚠️/❌ | ☆☆☆☆☆ | <建议> |

### 关键发现
1. <最需要改进的问题>
2. <次要问题>
3. <做得好的方面>

### 改进路线图
- **紧急（本周）：** ...
- **短期（本月）：** ...
- **长期（本季度）：** ...
```

---

### 工作流 3：批量 Issue Triage + 自动分配

**场景**：对新 Issue 进行自动分类、标签分配和责任人推荐。

```bash
# 1. 获取未标记的 Issue
gitlink-cli issue +list --state open --format json

# 2. 逐个分析 Issue 内容
gitlink-cli issue +view --id <issue_id> --format json

# 3. 根据内容智能分类
# 分析标题和描述后，通过 Raw API 打标签
gitlink-cli api POST /:owner/:repo/issues/:id --body '{
  "issue_tag_ids": [<tag_id>],
  "done_ratio": 0,
  "subject": "<原始标题>",
  "description": "<原始描述>"
}'
```

**分类规则参考：**

| Issue 关键词 | 推荐标签 | 优先级 |
|-------------|----------|:------:|
| bug, 错误, 失败, crash, 崩溃 | bug | 🔴 High |
| feature, 新增, 建议, 希望 | enhancement | 🔵 Low |
| 安全, 漏洞, 权限, 泄露 | security | 🔴 High |
| 性能, 慢, 卡顿, 优化 | performance | 🟡 Medium |
| 文档, README, 注释 | documentation | 🔵 Low |
| question, 如何, 怎么, 请问 | question | 🟡 Medium |
| 测试, test, 覆盖率 | testing | 🔵 Low |

---

## Raw API 参考

代码审查相关的 GitLink API 端点：

```bash
# 获取 PR 详情
gitlink-cli api GET /:owner/:repo/pulls/:id --format json

# 获取 PR 变更文件列表
gitlink-cli api GET /:owner/:repo/pulls/:id/files --format json

# 获取 PR Diff
gitlink-cli api GET /:owner/:repo/pulls/:id/diff --format json

# 提交 PR Review
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{"body":"...","event":"COMMENT"}'

# 获取仓库文件列表
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=<path>&ref=<branch>'

# 获取仓库语言统计
gitlink-cli api GET /:owner/:repo/languages --format json

# 获取贡献者列表
gitlink-cli api GET /:owner/:repo/contributors --format json

# 获取仓库动态
gitlink-cli api GET /:owner/:repo/activity --format json
```

## 代码审查最佳实践

### 审查原则

1. **先大局后细节**：先理解 PR 的目的和整体变更范围，再逐文件审查
2. **关注行为，而非风格**：自动化工具（linter/formatter）能处理的风格问题优先交给工具
3. **提供可操作的建议**：不只是指出问题，要给出具体的修改方案
4. **肯定好的代码**：发现好的设计、清晰的命名、完善的测试时给予正面反馈
5. **控制评论量**：避免信息过载——最严重的 3-5 个问题比 20 个小问题更有价值

### 安全红线

以下问题必须标记为 **Critical**，不得忽略：

- 硬编码的密钥 / Token / 密码
- SQL / NoSQL 注入漏洞
- 命令注入（shell 命令拼接）
- 路径遍历（用户输入直接用于文件路径）
- 不安全的反序列化
- XSS（未转义的用户输入直接渲染）

### 输出规范

- 始终使用 `--format json` 获取结构化数据
- 审查报告输出为 **Markdown 格式**，便于直接粘贴到 PR 评论
- 涉及文件/行号时使用精准引用，方便定位
- 批量操作前使用 `--dry-run` 预检

## 注意事项

- PR Review 提交后会通知所有关注该 PR 的参与者，评论内容请保持专业
- `pr +diff` 输出可能很大（大型 PR），Agent 应分段处理
- API 的 PR files 和 diff 接口有频率限制，避免短时间内重复请求
- 对于 draft PR（草稿），应提示用户先将其标记为 Ready for Review
