# gitlink-pr-assessor REFERENCE

> 本文件补充 `gitlink-pr-assessor` 的筛选规则、字段建议、自动化接入方式和报告结构。总体工作流以 [`SKILL.md`](./SKILL.md) 为准。

## 1. 数据源

评估 open PR 队列时，优先从四类信息构建证据。

### 1.1 PR 列表与筛选

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --page 1 --limit 50 --format json
```

注意：

- `--state open` 只作为粗过滤，必须再用 `pull_request_status == 0` 做二次过滤。
- PR 多时需要分页。

建议提取字段：

- `pull_request_number`
- `pull_request_status`
- `subject`
- `author_login`
- `updated_at`
- `files_count`
- `comments_count`

### 1.2 单条 PR 元数据

```bash
gitlink-cli pr +view --id <pr_id> --format json
gitlink-cli pr +reviews --id <pr_id> --format json
gitlink-cli api GET /:owner/:repo/pulls/:id/commits --format json
```

重点关注：

- 标题、描述、作者、base/head
- 现有 review 状态
- commit 数量、提交信息质量
- 关联 issue 和协作上下文

### 1.3 变更内容

```bash
gitlink-cli pr +files --id <pr_id> --format json
gitlink-cli pr +diff --id <pr_id> --format json
```

重点分析：

- 改动是否聚焦
- 是否涉及高风险目录
- 是否新增测试
- 是否新增帮助文档、CLI 输出或用户可见行为

### 1.4 仓库约束与执行证据

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli api GET /:owner/:repo/raw/master/README.md
gitlink-cli api GET /:owner/:repo/raw/master/Makefile
gitlink-cli api GET /:owner/:repo/raw/master/go.mod
```

从仓库中判断：

- 技术栈
- 默认分支
- 官方构建和测试命令
- 是否已有 CI
- 是否有 CONTRIBUTING、docs、examples

本地验证优先使用：

- 已检出的 PR 分支
- 独立 worktree
- 临时验证目录

---

## 2. “未审查”判定细则

### 2.1 基本规则

一条 PR 进入队列，需要满足：

1. `pull_request_status == 0`
2. review 列表中没有维护者 `approved` 或 `rejected`
3. 不存在 assessor 已产出报告的标记

### 2.2 报告标记

推荐在 Markdown 报告顶部写入：

```markdown
<!-- gitlink-pr-assessor:report v1 -->
```

有了这个标记，后续定时扫描时就可以识别“这条 PR 已经评估过”。

### 2.3 需要重新评估的条件

即使已有报告，以下情况也建议重新跑：

- PR 新增了 commit
- PR 描述被明显改写
- CI 状态从失败变为通过，或从通过变为失败
- 维护者明确要求重新评估

推荐记录：

- 上次评估时的 head commit SHA
- 上次评估时间
- 上次结论

---

## 3. 待验证声明抽取规则

把 PR 描述中的自然语言整理成可以验证的声明，每条声明应尽量满足“能验证、能反驳、能给出证据”。

### 3.1 常见声明类型

| 类型 | 例子 | 推荐验证方式 |
|------|------|-------------|
| bug 修复 | 修复 Windows 路径错误 | 跑失败用例、对比 base 与 PR 行为 |
| 新命令 | 新增 `wiki +list` | 构建 CLI、执行 `--help`、跑命令 |
| 参数增强 | 支持 `owner/repo:branch` | 直接执行目标参数组合 |
| 输出优化 | 错误信息更清晰 | 复现场景，检查输出文本 |
| 兼容性修复 | 兼容复杂分支名、空格路径 | 设计特定输入验证 |
| 安全修复 | 避免注入、限制权限 | 代码检查 + 负向测试 |

### 3.2 证据不足与无法验证

以下情况不要强行写“通过”或“失败”：

- PR 描述没有说明修复或新增了什么
- 需要外部服务、私有凭据或特定平台
- 仓库没有可执行验证步骤
- 行为入口不明确

此时应明确写为：

- `evidence_status: insufficient`
- 或 `execution_status: blocked`

---

## 4. 评估维度建议

### 4.1 贡献价值

重点看：

- 问题是否真实、常见或关键
- 是否符合项目定位
- 是否减少维护负担或补齐能力缺口

### 4.2 实现可行性

重点看：

- 是否贴合现有架构
- 是否依赖不存在的接口或假设
- 是否以过高复杂度解决小问题

### 4.3 代码质量

重点看：

- 模块边界是否清晰
- 错误处理是否一致
- 命名和控制流是否易读
- 测试是否覆盖行为而不仅是路径

### 4.4 安全与风险

重点看：

- 输入校验
- 命令、路径、模板、编码、URL、权限边界
- 敏感信息暴露
- 默认行为是否危险

### 4.5 维护成本

重点看：

- 是否引入特例逻辑
- 是否增加长期兼容负担
- 文档、帮助、测试是否同步更新

### 4.6 协作质量

重点看：

- PR 描述是否清楚
- 变更是否过大
- 是否适合拆分
- reviewer 是否容易理解和复现

---

## 5. 执行验证策略

### 5.1 验证模式

#### 快速分诊模式

适用于：

- PR 很多，需要先筛选
- 只需判断是否值得继续看
- 环境重、依赖多，不适合深跑

动作：

- 跑最小构建或最相关测试
- 验证 1-2 条关键声明
- 给出初步管理建议

#### 深度验证模式

适用于：

- PR 价值高
- 涉及核心命令或高风险模块
- 维护者需要强证据

动作：

- 依据仓库规范完整构建和测试
- 对主要声明逐项验证
- 补做回归检查

### 5.2 验证命令选择顺序

按下面优先级选择：

1. PR 描述里的验证命令
2. README / docs / CONTRIBUTING / Makefile
3. CI 工作流
4. 语言生态默认命令

### 5.3 停止条件

出现以下情况应停止深挖，并明确记录阻塞原因：

- 缺少依赖或凭据
- 命令会修改外部系统
- 构建耗时或资源开销过大
- 用户当前工作树有冲突风险

---

## 6. 结构化 JSON 输出建议

自动化接入时，优先输出结构化 JSON，再派生 Markdown。

### 6.1 单条 PR 输出

```json
{
  "repository": "Gitlink/gitlink-cli",
  "pr_number": 42,
  "title": "feat: add ...",
  "queue_reason": "open-unreviewed",
  "verdict": "needs_followup",
  "priority": "P2",
  "risk_level": "medium",
  "confidence": "medium",
  "execution_status": "partial",
  "required_human_review": ["cli", "testing"],
  "auto_review_eligible": false,
  "summary": "功能方向合理，但边界验证不足。",
  "dimensions": {
    "contribution_value": "strong",
    "feasibility": "medium",
    "code_quality": "medium",
    "security_risk": "low",
    "maintenance_cost": "low",
    "collaboration_quality": "medium"
  },
  "claims": [
    {
      "claim": "支持复杂分支名比较",
      "result": "insufficient",
      "evidence": "现有测试只覆盖简单分支名"
    }
  ],
  "commands": [
    {
      "type": "build",
      "command": "go build ./...",
      "result": "passed",
      "note": "无构建错误"
    }
  ],
  "suggested_review": {
    "status": "common",
    "comment_markdown": "请补充分支名包含特殊字符时的测试。"
  }
}
```

### 6.2 队列扫描输出

```json
{
  "repository": "Gitlink/gitlink-cli",
  "scanned_at": "2026-06-24T10:30:00+08:00",
  "open_pr_total": 18,
  "assessed_pr_total": 6,
  "skipped": [
    {
      "pr_number": 19,
      "reason": "already-approved"
    },
    {
      "pr_number": 20,
      "reason": "already-has-assessor-report"
    }
  ],
  "reports": [
    {
      "pr_number": 21,
      "verdict": "needs_followup",
      "priority": "P1",
      "risk_level": "high"
    }
  ]
}
```

### 6.3 推荐枚举值

| 字段 | 建议值 |
|------|--------|
| `verdict` | `ready_for_maintainer_confirmation` / `needs_followup` / `needs_more_evidence` / `needs_split` / `defer` / `reject` |
| `priority` | `P1` / `P2` / `P3` |
| `risk_level` | `low` / `medium` / `high` |
| `confidence` | `high` / `medium` / `low` |
| `execution_status` | `passed` / `partial` / `failed` / `blocked` / `not_run` |
| `claim.result` | `passed` / `failed` / `insufficient` / `blocked` |

---

## 7. 自动化接入建议

### 7.1 外层触发器

本 Skill 自身不监听 PR。自动化落地依赖外层 runner，例如：

- webhook 处理器
- 定时扫描任务
- `codex exec` 驱动脚本
- 其他 Agent 平台工作流

### 7.2 推荐事件

- PR opened
- PR reopened
- PR synchronized
- 定时全量补偿扫描

### 7.3 幂等策略

至少实现以下之一：

1. 报告正文加入 marker
2. 记录最近处理的 head SHA
3. 写入专用标签或状态文件

### 7.4 回写策略

默认建议：

- 先生成内部报告给维护者
- 不自动批准或拒绝
- 只有在仓库规则允许时，才自动回写普通评论或 review 建议

高风险 PR 一律只出报告，不自动形成强语义结论。

---

## 8. 回写建议

默认只读。只有用户或外层自动化明确要求写回时，才输出适合贴到 PR 的 Markdown。

推荐回写内容：

- 一句话结论
- 3 条以内最关键判断
- 执行验证结果
- 需要作者补充的点

不推荐回写：

- 过长命令日志
- 没有证据支撑的强结论
- 对维护者内部优先级的细节判断

---

## 9. 常见风险模式

- PR 描述很大，但测试很少
- 只新增 happy path，没有失败路径验证
- 声称“修复兼容性”，但没有平台特定验证
- 改动散落多个模块，却没有拆分
- 引入新行为，但帮助文档、命令说明、错误消息未更新
- compare、path、encoding、URL、base64 等边界只用简单样例验证

---

## 10. Windows 中文乱码排查

如果报告里的中文显示成 `?`，优先检查是不是终端编码链路有问题，而不是先怀疑 skill 文件本身损坏。

### 10.1 常见症状

- 控制台里中文正常，但重定向到文件后变成 `?`
- 通过 PowerShell 管道写文件时，中文丢失
- 报告里只有 ASCII 正常，中文和部分标点被替换

### 10.2 根因

在 Windows PowerShell 下，以下任一情况都可能导致中文被降成 `?`：

- 当前代码页不是 UTF-8
- `$OutputEncoding` 仍然是 `us-ascii`
- 用默认编码写文件，没有显式指定 UTF-8

### 10.3 建议修复

先执行：

```powershell
chcp 65001 > $null
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$OutputEncoding = [Console]::OutputEncoding
```

然后保存报告时显式写 UTF-8：

```powershell
$report | Set-Content -Path .\pr-queue-report.md -Encoding utf8
```

如果需要用 Python 生成或中转文本，再补一行：

```powershell
$env:PYTHONUTF8 = '1'
```

### 10.4 快速自检

```powershell
[Console]::OutputEncoding.WebName
$OutputEncoding.WebName
```

理想结果应为：

- `[Console]::OutputEncoding.WebName` 是 `utf-8`
- `$OutputEncoding.WebName` 也是 `utf-8`
