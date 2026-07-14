---
name: gitlink-newcomer
version: 1.0.0
description: "新人引导：识别 good-first-issue、评估上手难度与友好度、为候选 Issue 生成个性化引导评论、产出新手任务看板，降低新贡献者参与门槛。当用户提到「适合新人的 Issue」「good first issue」「新手任务」「引导新贡献者」「降低参与门槛」「new contributor」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-newcomer（新人引导）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 发布引导评论属于写操作，执行前必须征得用户确认。本技能默认只生成评论内容，不自动发布。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证与全局参数。

## 何时使用本技能

- 维护者想为新贡献者整理一份「适合上手的 Issue」清单
- 用户问「这个项目有哪些 good first issue / 新手任务」
- 想为某个简单 Issue 自动生成欢迎与引导评论，降低新人门槛
- 需要评估某个 Issue 对新手的友好程度

## 何时不使用

- 只是普通地列出全部 Issue → 用 `gitlink-issue`
- 复杂的项目健康度/协作分析 → 用 `gitlink-health` / `gitlink-insight`

## 能力概览

| 能力 | 说明 |
|------|------|
| good-first-issue 识别 | 综合标签、标题/正文关键词、描述长度、讨论热度等多信号识别 |
| 友好度评分 | 为每个 Issue 输出 0-100 的新手友好度分与难度等级（入门/较易/中等/进阶） |
| 个性化引导 | 为候选 Issue 生成包含上手步骤、Fork/PR 流程的欢迎评论 |
| 新手任务看板 | 汇总候选 Issue 为 Markdown 看板，可贴到 Wiki 或 README |

## 工作流 1：生成新手任务看板

### 方式 A：用配套脚本（推荐，一步到位）

```bash
# 扫描仓库 Issue，输出新手任务看板（Markdown）
python scripts/newcomer.py --owner Gitlink --repo gitlink-cli

# 输出 JSON 供 Agent 进一步处理（含每个候选的引导文案）
python scripts/newcomer.py --owner Gitlink --repo gitlink-cli --format json

# 写入文件
python scripts/newcomer.py --owner Gitlink --repo gitlink-cli --output board.md
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` | string | 是* | 仓库所有者（*或用 `--slug`） |
| `--repo` | string | 是* | 仓库名称 |
| `--slug` | string | 否 | `owner/repo` 或完整 URL，替代 owner/repo |
| `--issue` | int | 否 | 只为指定 Issue 编号（web 序号）生成引导 |
| `--limit` | int | 否 | 扫描 Issue 数量上限，默认 50 |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件路径，缺省打印到标准输出 |

### 方式 B：用 gitlink-cli 命令手动采集

当无法运行脚本时，Agent 可用以下命令采集数据后自行分析：

```bash
# 1. 获取开放 Issue 列表
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --format json

# 2. 查看某个 Issue 详情（--number 用 web 序号，不是全局 id）
gitlink-cli issue +view --owner Gitlink --repo gitlink-cli --number 12 --format json

# 3. 查看仓库标签，确认是否有 good first issue 类标签
gitlink-cli label +list --owner Gitlink --repo gitlink-cli --format json
```

识别规则建议：
- 带 `good first issue` / `beginner` / `新手` 等标签 → 强新手信号
- 标题/正文含 `typo` / `docs` / `文档` / `test` / `翻译` 等 → 易上手信号
- 含 `refactor` / `架构` / `并发` / `性能` 等 → 高难度信号

## 工作流 2：为单个 Issue 生成引导评论

```bash
# 生成引导文案（不发布）
python scripts/newcomer.py --owner Gitlink --repo gitlink-cli --issue 12

# 确认文案后，由用户决定是否发布为评论（写操作，需确认）
gitlink-cli issue +comment --owner Gitlink --repo gitlink-cli --number 12 -b "<引导文案>"
```

## API 注意事项

- **ID 混淆**：GitLink 的 Issue 列表接口（`issue +list`）通常只返回全局数据库 `id`，不含 web 序号。发评论 / 关联 Issue 时，`issue +comment` 的 `--number` 需要 web 序号（URL 中显示的编号），不要把全局 id 当 web 序号用。
- **写操作确认**：`issue +comment` 会真实发布评论，执行前务必向用户确认内容与目标 Issue。
- 数据采集全程只读，脚本默认不发布任何内容。

## 输出示例

参见 [`examples/`](examples/) 目录下的真实运行产物（新手看板、单 Issue 引导）。

## References

- [api-reference.md](references/api-reference.md) — 采集的接口、字段与 ID 混淆说明
- [scoring.md](references/scoring.md) — 新手友好度评分规则与信号词表
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
