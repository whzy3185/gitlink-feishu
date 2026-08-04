---
name: gitlink-maintainer-radar
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
description: "维护者雷达：面向 GitLink 仓库维护者，联合扫描 open Pull Request、open Issue、消息提醒、review 分配和等待时长，识别响应超时、review 负载失衡、负责人长期停滞等协作瓶颈，生成按优先级排序的处置清单、催办建议和责任调整建议。用于用户需要值班巡检待办、判断哪些事项被晾着了、找出 reviewer 瓶颈、发现有负责人但无进展的条目，或生成维护者今日工作面板时。"
---

# gitlink-maintainer-radar

**CRITICAL - 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，按其中的认证、全局参数和安全规则执行。**
**CRITICAL - 所有 GitLink 操作只使用 `gitlink-cli`，不要改用 `gh`、`glab` 或网页猜测数据。**
**CRITICAL - 默认只读分析。只有在用户明确要求时才回写评论、标签或成员分配。**

这个 skill 不再做“把通知列表抄一遍”的弱摘要，而是把三类真正影响维护者效率的治理信号合在一起：

1. **响应时效雷达**：找出超出响应 SLA 的 Issue 和 PR。
2. **Review 负载雷达**：判断 reviewer 是否失衡，以及哪些 PR 因分配不均而卡住。
3. **责任停滞雷达**：识别已经有 assignee / reviewer，但长期没有推进动作的条目。

把它当作“维护者值班面板”来用，而不是通知中心。

## 核心能力

### 1. 响应时效雷达

优先识别这些最有用的 SLA 违约场景：

- 超过 24 小时无人首次响应的 Issue
- 超过 3 天无人 review 的 PR
- 已经 `approve` 但 2 天内没有推进合并或进一步处理的 PR
- `requested changes` 之后作者长期未更新的 PR
- 作者已经更新、但维护者超过 48 小时未复看的 PR

这些信号最适合直接形成“今天先处理什么”的清单。

### 2. Review 负载雷达

从多人协作角度，重点看：

- 哪些 reviewer 手上挂了太多待处理 PR
- 哪些 reviewer 长期没有被分配，存在空闲容量
- 哪些 PR 因 reviewer 单点瓶颈停滞
- 哪些 PR 一直在同一个 reviewer 身上来回等待
- 哪些高优先级 PR 值得建议重新分配 reviewer

这个模块的目标不是简单统计数量，而是指出“哪里需要重新分配”。

### 3. 责任停滞雷达

重点识别这些“看起来有人负责，实际上没人推进”的场景：

- 有 assignee 的 Issue 长期无新评论、无状态更新
- 有 reviewer / assignee 的 PR 长期无 review、无结论、无合并动作
- 同一个责任人名下积压了过多停滞事项
- 条目虽然被分配，但最近一次动作仍然停留在发起人一侧
- 责任关系已经失效，应该提醒、转派或收口

这个模块比单纯的 stale 检查更实用，因为它关心的是“责任失效”。

### 4. 统一处置建议

把前面三类信号收敛成维护者真正能执行的动作：

- 先回复哪些条目，快速消除首响超时
- 哪些 PR 需要重新分配 reviewer
- 哪些 assignee / reviewer 需要提醒
- 哪些长期停滞条目应该收口、降级或重新明确责任人
- 哪些条目虽然 noisy，但本轮值班可以忽略

## 判断规则

### HOT

- 超过 SLA 且当前明显在等维护者动作
- 已 `approve` 但无人推进合并，且影响版本节奏
- 高优先级 PR 因 reviewer 负载失衡卡住
- 有负责人但超过阈值完全无动作
- 首次贡献者的高质量提交长期无人回应

### WATCH

- 尚未超 SLA，但已经接近阈值
- 责任人存在，但推进节奏明显偏慢
- reviewer 负载有失衡趋势，但还未造成明确阻塞
- 讨论在继续，但一直没有收敛到下一步动作

### BACKGROUND

- 普通消息提醒和不会改变维护节奏的系统通知
- 已明确有人处理且最近有进展的条目
- 短期不影响主线和协作节奏的低优先级事项

详细矩阵和建议阈值见 [`references/triage-matrix.md`](references/triage-matrix.md)。

## 工作流

### Step 1：确认仓库与维护者视角

先确认当前登录用户和仓库范围：

```bash
gitlink-cli auth status
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

如果用户没有显式提供仓库，且当前目录就是目标仓库，可以使用自动解析。

### Step 2：抓取基础队列数据

至少获取这几组数据：

```bash
# 未读消息，用于捕捉 @ 提醒和人工催办信号
gitlink-cli api GET "users/<login>/messages.json" --query "status=1&limit=40" --format json

# open PR
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --format json

# open Issue
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
```

如果队列很大，先拉最近 40 条，再聚焦高风险对象。

### Step 3：为高风险 PR 补拉 review 与等待链路

对命中 SLA、负载失衡或停滞信号的 PR，继续获取：

```bash
gitlink-cli pr +view --owner <owner> --repo <repo> -i <number> --format json
gitlink-cli pr +reviews --owner <owner> --repo <repo> -i <number> --format json
gitlink-cli pr +version-diff --owner <owner> --repo <repo> -i <number> --format json
```

要明确判断：

- 当前在等作者、等 reviewer，还是等 maintainer 决策
- 最近一次有效推进动作是谁完成的
- review 结论是否已经形成，但没有后续动作
- 是否存在 reviewer 过载导致的人工瓶颈

如果用户需要深入判断代码可行性，切换到 `gitlink-pr-assessor`。如果用户要判断是否适合集成主线，切换到 `gitlink-pr-integrator`。

### Step 4：为停滞 Issue 建立责任视图

对 Issue 至少判断：

- 是否已有 assignee
- 最近一次评论或状态更新距离现在多久
- 当前是在等提问方补信息，还是在等维护者接手
- 是否值得补标签、转派或收口

### Step 5：构建统一处置面板

输出不止要说明“发生了什么”，还要告诉维护者下一步怎么做：

- 哪些条目要先回复，修复 SLA 超时
- 哪些 PR 应该重新分配 reviewer
- 哪些条目需要提醒当前责任人
- 哪些条目应该转派、收口或延后
- 如果用户要求，生成适合直接发布的中文提醒评论草稿

评论草稿模板见 [`references/comment-templates.md`](references/comment-templates.md)。

### Step 6：输出维护者报告

报告应至少包含：

1. SLA 超时条目
2. reviewer 负载失衡情况
3. 有负责人但停滞的条目
4. 今天建议先做的动作
5. 可延后的背景项

推荐输出模板：

```markdown
# 维护者雷达报告

## SLA 超时
- Issue #87：超过 24 小时无人首次响应，当前仍无明确责任人。
- PR #123：作者 2 天前已按 review 修改，当前等待维护者复看。

## Review 负载
- reviewer A 当前挂着 5 个待 review PR，是主要瓶颈。
- reviewer B 最近没有分配到待处理条目，存在可释放容量。

## 责任停滞
- Issue #91 已分配给 maintainer C，但 6 天没有新动作。
- PR #118 有 reviewer，但上一轮意见后一直无人推进结论。

## 今日建议动作
- 先回复 Issue #87，避免继续首响超时。
- 把 PR #126 从 reviewer A 转给 reviewer B。
- 对 PR #118 发送一次责任确认提醒。

## 可延后
- 项目关注、点赞、普通系统通知可忽略。
```

## 评论与写操作规则

只有用户明确要求时，才执行这些动作：

- 给 PR / Issue 发表评论
- 调整标签
- 变更成员或分配关系
- 标记消息已读

在回写之前，先把拟执行动作和目标对象列清楚，再执行。

## 处理边界

- 这个 skill 的重点是“维护治理与协作分流”，不是完整代码审查。
- 它比 `gitlink-issue-triage` 更偏责任推进，而不是 Issue 语义分类。
- 它比 `gitlink-health`、`gitlink-insight` 更偏当下值班，而不是全局健康分析。
- 如果仓库没有启用 GitLink 平台 CI，不影响这个 skill 的核心价值。

## 典型触发语句

- “帮我看下这个仓库哪些 Issue 和 PR 被晾着了。”
- “扫描一下 open PR 和 open Issue，给我一个维护者值班清单。”
- “哪些 PR 因 reviewer 负载不均卡住了？”
- “找出有负责人但没进展的条目。”
- “把今天必须回复、必须转派、必须催办的事项挑出来。”
