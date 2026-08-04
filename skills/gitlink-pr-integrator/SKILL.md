---
name: gitlink-pr-integrator
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
description: 评估 GitLink Pull Request 是否已经具备集成到主线的条件，输出合并态验证、与其他 open PR 的冲突风险、集成影响面、发布与回移建议以及合并后动作清单。用于维护者需要决定某个 PR 是否可以进入 merge queue、为一批待合并 PR 排顺序、在合并前验证 rebase 或 merge 后是否仍能构建测试通过，或为自动化队列生成集成就绪报告时。
---

# gitlink-pr-integrator

**CRITICAL - 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL - GitLink 平台数据采集和回写只使用 `gitlink-cli`，不要改用 `gh` 或其他平台 CLI。**
**CRITICAL - 默认只做读取、验证和报告；只有用户明确要求时才回写评论或 review。**
**CRITICAL - 不要在用户当前的脏工作树里做合并验证。优先使用独立 worktree、临时 clone 或明确指定的检出目录。**
**CRITICAL - 在 Windows PowerShell 中保存中文报告前，先切到 UTF-8 输出链路，否则中文可能被写成 `?`。**

这个 Skill 解决的是“这个 PR 现在能不能安全并入主线”，不是“这个 PR 有没有价值”。如果需求是判断贡献价值、功能可行性、代码质量或声明是否成立，先使用 `gitlink-pr-assessor`；如果价值判断已经成立，需要决定是否进入合并队列、是否先 rebase、是否会与别的 open PR 打架，再使用这个 Skill。

执行命令前，按需读取 [`references/api_reference.md`](./references/api_reference.md)。其中包含 GitLink CLI 命令、Windows 调用方式、独立 worktree 验证方法和报告字段约定。

## Windows 前置

如果你在 Windows PowerShell 里运行或落盘报告，先执行：

```powershell
chcp 65001 > $null
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$OutputEncoding = [Console]::OutputEncoding
```

如果 PowerShell 因执行策略拦截 `gitlink-cli.ps1`，改用：

```powershell
& "$env:APPDATA\npm\gitlink-cli.cmd" auth status
```

如果全局安装版本落后于当前仓库源码，优先在仓库根目录运行：

```powershell
go run . pr --help
```

## 结论集合

最终结论只从下列集合中选择一个：

- `ready_to_merge`：合并态干净，官方构建/测试通过，冲突和发布风险可接受。
- `ready_after_rebase`：主要阻塞是基线已漂移，rebase 或重新合并后大概率可继续。
- `ready_after_followups`：代码本身接近可合并，但还缺文档、帮助文本、测试、changelog 或发布动作。
- `not_integration_ready`：当前无法安全并入主线，存在冲突、失败验证、较高回归风险或明显的集成阻塞。

同时给出以下评级：

- `merge_readiness`: `high` / `medium` / `low`
- `integration_risk`: `low` / `medium` / `high`
- `conflict_risk`: `low` / `medium` / `high`
- `release_impact`: `none` / `patch` / `minor` / `major`

## 标准流程

### Step 1: 采集 PR 集成上下文

先拿到目标 PR 的元信息、变更范围、已有 review 和仓库默认分支信息。

优先命令：

```bash
gitlink-cli pr +view --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli pr +files --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli pr +reviews --owner <owner> --repo <repo> --id <pr_number> --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
```

至少提取：

- base 分支、head 分支、head 来源仓库
- 变更文件、核心目录、是否涉及 CLI 命令入口、帮助文本、文档、测试
- 当前 review 结论、是否已有 maintainer 明确阻塞项
- 仓库默认分支、语言、CI 是否开启、项目推荐的验证命令

### Step 2: 准备独立的集成验证环境

集成验证必须隔离执行。优先顺序如下：

1. 用户明确提供的临时检出目录
2. 当前仓库下的新 worktree
3. 系统临时目录里的新 clone

禁止直接在用户当前脏工作树里 `merge` 或 `rebase`。如果仓库里已经有未提交改动，只把它当信息源，不把它当验证环境。

### Step 3: 做合并态验证

目标不是只看 PR 自己能不能编译，而是回答“把它并到最新主线后还能不能工作”。

建议流程：

```bash
git fetch origin <base_branch>
git worktree add <temp_dir> origin/<base_branch>
cd <temp_dir>
git switch -c pr-integration-check
git remote add pr-source <head_repo_url>
git fetch pr-source <head_branch>
git merge --no-ff --no-commit FETCH_HEAD
```

如果 PR head 就在同一个远端，也可以直接从 `origin/<head_branch>` 拉取，不必额外加 remote。
注意 GitLink PR 的 `head` 字段常见格式是 `login/branch`。对 fork PR 做本地验证时，不要机械地把它裁成最后一段；如果你的 remote 名就叫这个 login，那么实际 remote-tracking ref 可能是 `refs/remotes/<login>/<head>`，例如 `refs/remotes/mengz/mengz/api-single-call-templates`。

记录下列结果：

- 是否无冲突完成 merge
- 是否必须 rebase 才能继续
- 官方构建命令是否通过
- 官方测试命令是否通过
- 是否出现只在合并态暴露的问题，例如接口签名漂移、帮助文本未同步、测试夹具过时、文档示例失效

验证命令必须优先使用项目文档、CI 配置、`Makefile` 或仓库惯例，不要发明一套项目从未使用过的检查方式。

### Step 4: 扫描与其他 open PR 的冲突风险

集成就绪度不是单 PR 视角，还要考虑队列里的其他候选项。

先列出 open PR：

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --page 1 --limit 50 --format json
```

然后重点比较：

- 是否修改同一文件
- 是否落在同一目录或模块
- 是否同时修改同一条 CLI 命令、flag、帮助文案或 API 包装层
- 是否会产生相互覆盖的测试或快照

冲突评级建议：

- `high`：同文件或同命令入口直接重叠，合并顺序明显重要
- `medium`：目录或模块重叠，存在行为级联风险
- `low`：基本独立，只存在轻微上下文漂移可能

如果发现明显的先后依赖，给出建议合并顺序。

### Step 5: 输出集成影响矩阵

不要只写“测试通过”。要明确主线在什么面上会被改变。

至少覆盖以下面向：

- CLI 命令行为
- flags / help 输出
- README / docs / 示例
- API 封装或协议兼容性
- 测试与夹具
- release notes / changelog

如果代码改了，但帮助文本、README、示例或测试没有同步，直接记为集成跟进项，而不是轻描淡写地放过。

### Step 6: 给出发布与回移建议

把改动归入以下类型之一：

- bugfix
- feature
- breaking change
- refactor-only

并说明：

- 对版本号的影响更像 `patch` / `minor` / `major`
- 是否需要 release notes
- 是否需要迁移说明或兼容性提示
- 是否适合回移到维护分支

### Step 7: 形成合并后动作清单

如果 PR 代码已经接近可合并，但还差最后几步，明确写成动作清单：

- 补 help / README / 示例
- 补或修正测试
- 更新 changelog / release notes
- 调整 milestone / 看板状态
- 合并后立即跟进的 issue 或回归验证

### Step 8: 可选回写

只有用户明确要求时，才把结论回写到远端。回写前先生成本地 Markdown 报告，并优先 `dry-run`。

适合的回写方式：

- `pr +comment`：发布集成报告
- `pr +review --status common --dry-run`：预览 review 文案

不要默认 approve 或 merge。这个 Skill 的职责是“给出可集成判断”，不是替维护者自动盖章。

## 报告模板

```markdown
<!-- gitlink-pr-integrator:report v1 -->
## PR #<id> 集成就绪报告

**结论：** ready_after_followups
**merge_readiness：** medium
**integration_risk：** medium
**conflict_risk：** high
**release_impact：** minor

### 1. 合并态验证
- 基线：`<base_branch>`
- 结果：可合并 / 需 rebase / 存在冲突
- 构建：通过 / 失败 / 未执行
- 测试：通过 / 失败 / 未执行
- 备注：<只在合并态暴露的问题>

### 2. 与 open PR 的冲突分析
| PR | 风险 | 原因 | 建议顺序 |
|----|------|------|----------|
| #123 | high | 同时修改 `shortcuts/pr/pr.go` | 先合并对方 |

### 3. 集成影响矩阵
| 面向 | 状态 | 说明 |
|------|------|------|
| CLI 行为 | changed | 新增 `...` |
| Help / docs | follow-up needed | 命令帮助已更新，README 未同步 |
| Tests | changed | 新增单测，但缺少回归场景 |

### 4. 发布建议
- 类型：feature
- 版本影响：minor
- 是否需要 release notes：是
- 是否建议回移：否

### 5. 合并后动作
1. <动作 1>
2. <动作 2>
3. <动作 3>
```

## 批量模式

如果用户要求扫描 PR 队列，按下面的顺序执行：

1. 列出 open PR。
2. 过滤掉已经 merged、closed 或已经明确被维护者拒绝的项。
3. 按最近活动时间、冲突密度和合并态风险排序。
4. 对前 N 条候选 PR 逐条生成集成就绪报告。
5. 再输出一份队列总览，包含建议合并顺序和需要先处理的冲突热点文件。

批量模式下，仍然不要默认对全部 PR 执行高成本本地构建。先做元信息和冲突雷达，只有用户指定或风险较高时再进入本地合并验证。

## 示例请求

- “使用 `gitlink-pr-integrator` 检查 `Gitlink/gitlink-cli` 的 PR #281 是否已经具备合并条件，不要回写远端。”
- “使用 `gitlink-pr-integrator` 扫描 `Gitlink/gitlink-cli` 最近 10 个 open PR，给出建议合并顺序、冲突风险和发布影响。”
