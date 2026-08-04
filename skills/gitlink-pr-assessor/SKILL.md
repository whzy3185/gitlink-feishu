---
name: gitlink-pr-assessor
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
description: "开源社区 Pull Request 队列评估与执行验证：面向 open 且尚未形成维护者结论的 PR，批量拉取 GitLink Pull Request 的描述、diff、review、CI 与仓库上下文，评估贡献价值、实现可行性、代码质量、安全性、维护成本、协作质量和回归风险，并在项目规定环境下验证 PR 声明是否与实际行为一致。用于维护者需要自动筛查待审 PR、生成逐条管理报告、给出 review 建议，或为 webhook/定时任务/Agent runner 提供结构化决策结果时。"
---

# gitlink-pr-assessor

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理、GitLink API 特性和安全边界。**
**CRITICAL — GitLink 平台操作只能使用 `gitlink-cli`。禁止用 `gh` 或其他平台 CLI 操作 GitLink PR。**
**CRITICAL — 本 Skill 的默认目标是给维护者生成报告，不默认回写评论、Review、标签或合并结论。**
**CRITICAL — 执行验证优先使用仓库规定的构建/测试命令；不要擅自发明与项目习惯不一致的验证方式。**
**CRITICAL — 不要在用户当前工作树上冒险覆盖代码。执行验证优先使用独立 worktree、临时目录或已明确指定的 PR 检出目录。**
**CRITICAL — 在 Windows PowerShell 中生成或保存中文报告前，先切换到 UTF-8 输出链路；否则报告中的中文可能被写成 `?`。**

> 这个 Skill 是“评估引擎”，不是常驻监听进程。要实现社区里 open PR 自动审查，必须由 webhook、定时任务或 Agent runner 负责触发它。

### Windows 编码前置

如果你在 Windows PowerShell 里演示、重定向或落盘报告，先执行：

```powershell
chcp 65001 > $null
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$OutputEncoding = [Console]::OutputEncoding
```

如果要把报告保存成文件，显式指定 UTF-8，不要依赖默认编码：

```powershell
$report = @"
<!-- gitlink-pr-assessor:report v1 -->
## PR #42 维护者评估报告
...
"@
$report | Set-Content -Path .\pr-42-report.md -Encoding utf8
```

---

## 目标

这个 Skill 解决的是维护者的队列压力，而不只是单条 PR 的代码挑错。

它要回答的是：

1. 当前 open PR 里，哪些还没有形成维护者结论，值得优先看。
2. 每条 PR 是否真的解决了一个有价值的问题。
3. PR 描述中的功能声明，是否能在项目规定环境下跑通。
4. 这条 PR 应该进入哪条处理路径：继续人工评审、补证据、拆分、暂缓，还是直接建议合并。
5. 如果需要人工接手，维护者最应该先看什么，review 文案可以怎么写。

---

## 使用模式

### 模式 1：单条 PR 深度评估

适用于：

- 维护者点名要看一条 PR
- 某条 PR 风险高，需要执行验证
- 用户想判断一条 PR 是否值得继续推进

### 模式 2：open 未审查 PR 队列扫描

适用于：

- 仓库有一批 open PR 等待维护者处理
- 希望自动筛出“尚未形成维护者结论”的 PR
- 希望给每条 PR 生成一份统一结构的报告，供维护者批量浏览

如果用户没有特别指定，优先按“队列扫描”来理解这个 Skill。

---

## open 未审查 PR 的判定规则

这个 Skill 对“未审查”采用保守定义，目标是筛出“还没有维护者结论”的 PR，而不是简单看有没有任何评论。

一条 PR 进入待评估队列，需要同时满足：

1. `pull_request_status == 0`，即 PR 仍然是 open。
2. 没有维护者最终态 review，例如 `approved` 或 `rejected`。
3. 没有本 Skill 已经产出的报告标记，例如 `<!-- gitlink-pr-assessor:report v1 -->`。

以下情况默认仍视为“未审查”：

- 作者自己的补充评论
- 普通讨论评论，但没有形成维护者结论
- 只有零散 review 意见，还没有统一判断

如果仓库另有规则，也可以扩展为：

- 没有 `maintainer-reviewed` 标签
- 没有“已分派 reviewer”记录
- 没有满足门禁的自动评分结果

---

## 标准流程

### Step 1：枚举 open PR

先拿仓库 PR 列表：

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --page 1 --limit 50 --format json
```

注意：

- GitLink 的 `--state open` 过滤不总是精确，必须再用返回里的 `pull_request_status` 做客户端过滤。
- 仓库 PR 多时要分页扫描。

### Step 2：筛选待评估队列

对每条 open PR 补拉 review 信息：

```bash
gitlink-cli pr +reviews --owner <owner> --repo <repo> --id <pull_request_id> --format json
gitlink-cli pr +view --owner <owner> --repo <repo> --id <pull_request_id> --format json
```

然后判断：

- 是否已有 `approved` 或 `rejected`
- 是否已有 assessor 报告标记
- 是否需要跳过，例如已是 draft、明显等待作者补改、或仓库规则要求延后处理

输出一个“本轮待处理 PR 队列”。

### Step 3：为每条 PR 采集评估上下文

```bash
gitlink-cli pr +view --id <pull_request_id> --format json
gitlink-cli pr +files --id <pull_request_id> --format json
gitlink-cli pr +diff --id <pull_request_id> --format json
gitlink-cli pr +reviews --id <pull_request_id> --format json
gitlink-cli api GET /:owner/:repo/pulls/:id/commits --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
```

`pr +view`、`pr +files`、`pr +diff`、`pr +reviews` 在这里统一使用 `pull_request_id`。
`pull_request_number` 只保留给维护者看的报告标题、队列表格和网页链接。

至少提取：

- PR 标题、描述、作者、创建时间、base/head
- 文件数、增删行、核心改动目录
- 现有 review 和 comment 的主要争议点
- commit 粒度与信息质量
- CI 状态
- 仓库默认分支、语言、构建方式和测试方式

### Step 4：提炼“待验证声明”

从 PR 标题、描述、关联 issue、测试说明中提炼作者声称完成的事情，例如：

- 修复某个具体 bug
- 新增某个命令、参数或输出格式
- 改善兼容性、错误提示或跨平台行为
- 不影响已有流程

每条声明都必须能被标记为以下结果之一：

- 通过
- 失败
- 证据不足
- 无法验证

### Step 5：做静态决策评估

至少覆盖以下维度：

| 维度 | 关注点 |
|------|------|
| 贡献价值 | 是否解决真实痛点，是否符合项目方向 |
| 实现可行性 | 是否贴合现有架构，复杂度是否合理 |
| 代码质量 | 结构、命名、错误处理、测试、可读性 |
| 安全与风险 | 输入校验、权限边界、回归风险、危险默认行为 |
| 维护成本 | 后续扩展成本、特例逻辑、文档与帮助是否同步 |
| 协作质量 | PR 描述是否清楚、变更是否过大、是否适合拆分 |

### Step 6：做执行验证

按下面顺序决定验证方式：

1. PR 描述中作者给出的验证步骤
2. 仓库 README、docs、CONTRIBUTING、Makefile
3. CI 配置中的官方命令
4. 语言生态默认命令

常见命令示例：

- Go：`go build ./...`、`go test ./...`
- Node.js：`npm test`、`pnpm test`、`npm run build`
- Python：`pytest`、`python -m pytest`
- Rust：`cargo test`、`cargo build`

执行验证分三层：

1. **环境层**：依赖、构建、测试入口能否正常工作。
2. **功能层**：PR 声明的行为是否复现。
3. **回归层**：核心已有流程是否仍然正常。

### Step 7：给出维护者报告

每条 PR 都要生成一份独立报告，核心必须包含：

- 明确结论
- 风险等级
- 证据强度
- 执行验证状态
- 声明验证结果
- 最值得维护者关注的 2-3 个点
- 建议 review 文案

### Step 8：输出队列总览

如果是批量扫描，还要再生成一个“本轮 PR 队列总览”，至少包括：

- 本轮扫描时间
- 扫描仓库
- open PR 总数
- 纳入评估的 PR 数
- 被跳过的 PR 及原因
- 每条待处理 PR 的结论、风险和优先级

---

## 输出格式

优先同时产出两份结果：

### 1. 结构化 JSON

给 runner、自动化脚本或后续 Agent 消费。字段建议见 [`REFERENCE.md`](./REFERENCE.md)。

### 2. Markdown 报告

给维护者直接阅读，既可以本地存档，也可以在得到授权后回写到 PR 或管理面板。

---

## 维护者报告模板

```markdown
<!-- gitlink-pr-assessor:report v1 -->
## PR #<id> 维护者评估报告

**结论：** 建议小改后再进入人工评审
**风险等级：** 中
**证据强度：** 中
**执行验证：** 部分通过

### 核心判断
1. <这条 PR 到底解决了什么问题，值不值得继续看>
2. <维护者现在最需要关注的实现问题或风险>
3. <最关键的验证结果或阻塞项>

### 维度评估
| 维度 | 结论 | 说明 |
|------|------|------|
| 贡献价值 | 强 / 中 / 弱 | ... |
| 实现可行性 | 强 / 中 / 弱 | ... |
| 代码质量 | 强 / 中 / 弱 | ... |
| 安全与风险 | 低 / 中 / 高 | ... |
| 维护成本 | 低 / 中 / 高 | ... |
| 协作质量 | 强 / 中 / 弱 | ... |

### 声明验证
| 声明 | 结果 | 证据 |
|------|------|------|
| <作者声称修复/新增的内容> | 通过 / 失败 / 证据不足 / 无法验证 | <命令、测试或观察> |

### 执行验证记录
| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `...` | 通过 | ... |
| 测试 | `...` | 通过 | ... |
| 场景验证 | `...` | 失败 | ... |

### 建议 review 文案
<给维护者可直接改写或粘贴的 review 建议，重点指出应让作者补什么、维护者接下来怎么处理。>
```

---

## 队列总览模板

```markdown
# <owner>/<repo> PR 待审队列报告

扫描时间：<timestamp>
open PR：<n>
纳入评估：<n>
跳过：<n>

| PR | 标题 | 结论 | 风险 | 优先级 | 说明 |
|----|------|------|------|--------|------|
| #12 | ... | 建议优先人工评审 | 高 | P1 | 涉及认证与权限 |
| #13 | ... | 建议补测试后再审 | 中 | P2 | 功能价值明确，但证据不足 |
| #14 | ... | 建议暂缓 | 低 | P3 | 依赖上层设计决策 |

## 跳过项
- #15：已有 `approved`
- #16：已存在 assessor 报告标记
```

---

## 自动化接入方式

如果要把它真正放进开源社区，不要要求维护者手工逐条调用，而是用外层系统定时或事件触发它。

推荐的触发方式有两类：

### 方式 1：PR 事件触发

在以下事件触发一次评估：

- PR opened
- PR reopened
- PR synchronized / push new commits

适合及时反馈，但要注意避免重复生成报告。

### 方式 2：定时队列扫描

例如每 10 分钟或每小时扫一次仓库 open PR：

- 列出 open PR
- 过滤出未审查项
- 为每条生成报告
- 汇总为维护者面板或日报

适合社区管理场景，也更容易补偿 webhook 漏触发。

### 幂等规则

自动模式必须有幂等设计，避免同一条 PR 重复刷报告：

- 在回写内容中加入 `<!-- gitlink-pr-assessor:report v1 -->`
- 或记录最近处理的 commit SHA
- 或给 PR 打上专用标签，例如 `assessor-reviewed`

---

## 结论规则

最终结论必须是明确动作，而不是泛泛而谈。推荐使用以下集合：

- 建议直接进入合并前人工确认
- 建议小改后继续人工评审
- 建议补测试或补文档后再审
- 建议拆分后重提
- 建议暂缓
- 建议拒绝

同时标注：

- `risk_level`：低 / 中 / 高
- `confidence`：高 / 中 / 低
- `execution_status`：通过 / 部分通过 / 未通过 / 无法验证
- `priority`：P1 / P2 / P3

---

## 边界

- 不要把“无法验证”写成“失败”。
- 不要把“作者写了测试”写成“功能已经被证明正确”。
- 不要只看代码风格而忽略真实价值和维护成本。
- 不要只跑全量测试而忽略 PR 描述中的关键声明。
- 不要在缺少证据时给出过度确定的结论。
- 不要把这个 Skill 伪装成自动监听器；自动化必须由外层 webhook、cron 或 runner 提供。

更多字段建议、筛选规则、自动化运行建议和 JSON 结构见 [`REFERENCE.md`](./REFERENCE.md)。
实际验证产物见 [`examples/codex-first40-validation.md`](./examples/codex-first40-validation.md)。
