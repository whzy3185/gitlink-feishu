---
name: gitlink-pr-topology
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
description: "开源社区 PR 队列关系图谱：面向一个仓库的多条 open Pull Request，识别它们之间的依赖链、功能重叠、替代/超越关系、冲突热点、可打包评审分组和建议处理顺序。用于维护者需要批量梳理 open PR 为什么互相卡住、哪几条其实在做同一件事、哪一条实现更完整、哪些 PR 应该先合并或先关闭，以及如何把复杂队列整理成可执行决策时。"
---

# gitlink-pr-topology

**CRITICAL - 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL - GitLink 平台数据采集和回写只使用 `gitlink-cli`。**
**CRITICAL - 这个 skill 关注的是 PR 与 PR 之间的关系，不代替单条 PR 的代码审查或合并验收。**
**CRITICAL - 如果需要把中文报告写入文件或重定向输出，在 Windows PowerShell 中先切到 UTF-8 输出链路。**

这个 skill 解决的是“PR 太多，维护者看不出它们彼此是什么关系”的问题。

它不只回答“有没有重复”，还要回答：

1. 哪些 PR 有明显的先后依赖，像 stacked PR 一样要按顺序处理。
2. 哪些 PR 实际上在解决同一个需求、同一个 bug、同一个命令入口。
3. 如果两条 PR 目标重叠，哪一条更完整、更稳、更值得保留。
4. 哪些 PR 虽然不完全重复，但会在同一文件、同一命令、同一输出契约上互相打架。
5. 哪些 PR 应该一起评审，避免维护者重复进入同一上下文。
6. 当前 open PR 队列最合理的处理顺序是什么。

## 不覆盖的内容

下面这些不属于本 skill 的职责：

- 单条 PR 的贡献价值、可行性、执行验证：交给 `gitlink-pr-assessor`
- 单条 PR 是否已经具备并入主线的条件：交给 `gitlink-pr-integrator`
- 维护者值班、SLA、review 负载和停滞治理：交给 `gitlink-maintainer-radar`

## Windows UTF-8 前置

如果你在 Windows PowerShell 中运行并准备保存中文报告，先执行：

```powershell
chcp 65001 > $null
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$OutputEncoding = [Console]::OutputEncoding
```

保存报告时显式指定 UTF-8：

```powershell
$report | Set-Content -Path .\pr-topology-report.md -Encoding utf8
```

## 关系类型

先阅读 [`references/relationship-taxonomy.md`](references/relationship-taxonomy.md) 了解关系定义和证据标准。这个 skill 至少识别以下六类关系：

1. `depends_on`
表示 PR B 依赖 PR A 先落地，否则 B 难以独立评审、测试或合并。

2. `overlaps_with`
表示两条 PR 在需求目标、命令入口、模块范围或改动文件上明显重叠。

3. `supersedes`
表示一条较新的 PR 在同一目标上覆盖更完整，足以替代另一条较弱 PR。

4. `conflicts_with`
表示两条 PR 即使目标不同，也会在同一文件、同一 flag、同一 JSON 字段、同一帮助文案或同一 API 包装层上互相冲突。

5. `review_together`
表示几条 PR 共享足够多的上下文，维护者一起看更高效。

6. `merge_after`
表示不是严格代码依赖，但为了减少返工，建议某条 PR 排在另一条之后处理。

## 标准流程

### Step 1：拉取 open PR 队列

先列出目标仓库的 open PR：

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --page 1 --limit 50 --format json
```

注意：

- `--state open` 的服务端过滤并不总是可靠，必须再用 `pull_request_status == 0` 做客户端过滤。
- 队列过大时优先扫描最近活跃的前 20-50 条，而不是一次吃完整个仓库。

### Step 2：为每条 PR 建立关系画像

对每条候选 PR 至少补拉这些信息：

```bash
gitlink-cli pr +view --owner <owner> --repo <repo> --id <number> --format json
gitlink-cli pr +files --owner <owner> --repo <repo> --id <number> --format json
gitlink-cli pr +reviews --owner <owner> --repo <repo> --id <number> --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

优先提取：

- PR 标题、描述、作者、创建时间、最近更新时间
- base/head 分支、fork 来源
- 修改文件、核心目录、是否触及同一条命令或同一 API 封装
- 是否修改测试、帮助文案、README、示例
- review 争议点、是否已有人指出重复或依赖关系
- 关联 issue、里程碑、标签

### Step 3：先做“候选关系”粗筛

先不要急着得结论，先把可能有关联的 PR 成对找出来。粗筛信号包括：

- 标题和描述出现同一需求词、同一命令名、同一 issue 编号
- 修改相同文件
- 修改同一目录或同一 shortcuts 子模块
- 同时触碰同一 flag、同一输出字段、同一错误提示
- 一条 PR 的描述直接提到 “基于 #xx” “依赖 #xx” “替代 #xx”
- 两条 PR 都在补同一类能力，例如 release、attachment、milestone、search

只把这些候选对放进下一步，不要把所有 PR 两两做重分析。

### Step 4：判断具体关系类型

对每个候选对，结合 [`references/relationship-taxonomy.md`](references/relationship-taxonomy.md) 给出单一主关系，必要时允许附加次关系。

判断顺序建议如下：

1. 先看是否存在明确依赖链。
2. 再看是否实际上在做同一件事。
3. 再看是否已出现“更完整版本替代较弱版本”。
4. 如果目标不同但落点冲突，则标为冲突热点。
5. 如果只是共享上下文但不冲突，标为建议一起评审。

没有足够证据时，写成 `possible_overlap` 或 `possible_dependency`，不要过度下结论。

### Step 5：在重叠 PR 中比较“谁更值得保留”

如果两条或多条 PR 目标重叠，读取 [`references/comparison-rubric.md`](references/comparison-rubric.md)，从以下维度比较：

- 需求覆盖是否更完整
- 代码路径是否更贴近现有架构
- 测试是否更充分
- 帮助文档、README、示例是否同步
- 向后兼容性是否更好
- 风险和复杂度是否更低
- review 反馈吸收是否更充分

输出时不要只说“PR A 更好”，而要明确指出：

- A 比 B 多解决了什么
- B 缺了什么
- B 是否还能拆成补充 PR，还是应该直接关闭

### Step 6：生成队列图谱和处理顺序

最终输出的不是一堆散点结论，而是一份维护者可执行的“队列图谱”：

- 哪些是依赖链，先后顺序怎样
- 哪些是一组重叠实现，需要择一保留
- 哪些是热点文件/热点命令，应该集中处理
- 哪些 PR 值得一起 review
- 哪些 PR 可以暂缓，因为上游未定

## 输出要求

同时产出两类结果：

### 1. 关系边列表

每条关系边至少包含：

- `source_pr`
- `target_pr`
- `relation`
- `confidence`
- `evidence`
- `recommended_action`

### 2. 维护者摘要报告

报告至少包含：

1. open PR 总数和本轮纳入分析的数量
2. 主要依赖链
3. 主要重叠簇
4. 明显替代关系
5. 冲突热点文件/模块
6. 建议处理顺序
7. 需要进一步切换到 `gitlink-pr-assessor` 或 `gitlink-pr-integrator` 深挖的对象

## 报告模板

```markdown
# <owner>/<repo> PR 队列关系图谱

扫描时间：<timestamp>
open PR：<n>
纳入分析：<n>

## 1. 依赖链
- #41 -> #44 -> #52
  说明：#44 基于 #41 引入的 API 包装，#52 又建立在 #44 的 CLI 参数层上。

## 2. 重叠实现
- #61 vs #63
  共同点：都在实现同一条命令的编号搜索能力。
  保留建议：优先保留 #63，因为测试覆盖更完整，且同时补了帮助文档和 JSON 输出。

## 3. 替代关系
- #71 supersedes #58
  说明：#71 覆盖了 #58 的核心功能，还补齐了错误处理和帮助文档；#58 可关闭或拆成子改动。

## 4. 冲突热点
- `shortcuts/pr/pr.go`
- `internal/client/client.go`
- `README.md`

## 5. 建议一起评审
- #80, #81, #83
  说明：都在修改 milestone 相关 CLI 行为，一起看更容易统一参数和输出契约。

## 6. 建议处理顺序
1. 先处理 #41，解除后续依赖链阻塞。
2. 在 #61 和 #63 中择一保留，避免重复 review。
3. 将 #80、#81、#83 打包评审，统一命令体验。
4. 暂缓 #52，等待上游 API 包装方案稳定。
```

## 典型触发语句

- “扫描这个仓库的 open PR，找出哪些在做同一件事。”
- “帮我分析这批 PR 的依赖关系和建议合并顺序。”
- “哪些 PR 其实可以一起 review，哪些应该择一保留？”
- “如果有两条 PR 功能重叠，判断哪条实现更完整。”
- “给我一个 open PR 队列关系图谱，方便维护者决定先看谁。”
