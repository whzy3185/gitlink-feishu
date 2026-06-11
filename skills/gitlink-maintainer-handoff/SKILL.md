---
name: gitlink-maintainer-handoff
version: 1.0.0
description: "维护者交接摘要：汇总站内消息、开放 Issue/PR、最近发布和仓库风险，生成下一位维护者可直接接手的交接清单。当用户需要值班交接、日报/周报、维护汇总时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli workflow --help"
---

# gitlink-maintainer-handoff（维护者交接摘要）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 默认只读。评论、关闭、合并、删除等远端写操作，必须在用户明确确认后再执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。
> **执行样例：** 参见 [`examples/gitlink-cli-maintainer-handoff.md`](examples/gitlink-cli-maintainer-handoff.md)

---

## 功能概述

本 Skill 面向项目维护者、值班同学和接手协作者，目标是在一次读取中回答四个问题：

1. 现在有没有需要立刻处理的站内消息或仓库风险
2. 当前开放的 PR 和 Issue 哪些最值得优先跟进
3. 最近一次发布之后，仓库当前处在什么状态
4. 下一位维护者接手时，最先该做什么

适用场景：

- 日交接 / 周交接
- 比赛或答辩期间的维护汇总
- 仓库管理员换班
- 长假前的维护状态封板

---

## 工作流：生成维护者交接摘要

### Step 1：确认当前账号和交接范围

```bash
# 确认当前登录账号
gitlink-cli user +me --format json

# 建议显式指定目标仓库，避免当前目录 remote 干扰
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

如果用户没有指定仓库，先确认当前目录的 `origin` 指向，再决定是否继续。

### Step 2：读取站内消息积压

```bash
# 当前用户的未读站内消息
gitlink-cli api GET "users/{login}/messages.json" --query "status=1&limit=20" --format json
```

规则：

- `unread_notification` 和 `unread_atme` 都要纳入摘要
- 即使结果为 0，也要在报告里明确写出“当前无站内消息积压”
- 消息读取属于只读；只有“标记已读”才算写操作

### Step 3：读取仓库治理总览

```bash
# 一次性获取健康度、Issue 摘要和 PR 摘要
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --lang zh-CN --format json
```

重点提取：

- `health.health_score`
- `risk_level`
- `recommendations`
- `issue_summary`
- `pr_summary`

规则：

- 如果 `risk_level=high`，摘要顶部必须给出风险提示
- `workflow +repo-report` 是交接总览，不足以替代单条 PR/Issue 跟进

### Step 4：拉取开放 PR 列表并识别交接重点

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --limit 20 --format json
```

规则：

- GitLink 的 PR 列表接口在部分场景下会返回额外状态，必要时按返回字段做客户端过滤
- 优先保留最近创建、最近更新、标题明确、影响面较大的 PR
- 对 1 到 3 个重点 PR，可继续补充 `workflow +pr-summary`

```bash
gitlink-cli workflow +pr-summary --owner <owner> --repo <repo> --number <pr-number> --lang zh-CN --format markdown
```

### Step 5：拉取开放 Issue 列表并识别阻塞项

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --limit 20 --format json
```

规则：

- `status_id=0` 视为“状态异常但仍需人工确认”的开放 Issue，不要直接忽略
- 优先关注最近更新时间新、描述明确、影响 CLI 可用性的缺陷
- 如果仓库没有标签体系，允许用标题关键词和更新时间代替标签判断优先级

### Step 6：查看最近发布状态

```bash
gitlink-cli release +list --owner <owner> --repo <repo> --limit 3 --format json
```

重点提取：

- 最近 release 的 `tag_name`
- `published_at`
- `target_commitish`
- 附件数量（用于判断资产是否完整）

### Step 7：生成可直接交接的 Markdown 摘要

输出时建议固定为以下结构：

```markdown
# 维护者交接摘要

> 仓库：{{owner}}/{{repo}}
> 生成时间：{{now}}
> 接手账号：{{login}}

## 一、当前状态

- 站内消息：{{message_summary}}
- 仓库风险：{{risk_level}}（健康分 {{health_score}}）
- 最近发布：{{latest_release}}

## 二、优先跟进的 PR

| PR | 标题 | 建议动作 |
|----|------|----------|
| #{{number}} | {{title}} | {{next_action}} |

## 三、优先跟进的 Issue

| Issue | 标题 | 风险提示 | 建议动作 |
|-------|------|----------|----------|
| #{{number}} | {{title}} | {{risk_note}} | {{next_action}} |

## 四、下一位维护者第一小时建议

1. {{first_action}}
2. {{second_action}}
3. {{third_action}}
```

---

## 输出规则

- 先写结论，再写证据，不要把原始 JSON 直接堆给用户
- 没有数据时输出“无积压 / 无紧急项”，不要留空章节
- 交接建议必须是动作导向句子，例如“先看 PR #209 的分页逻辑验证结果”
- 如果仓库风险高，但站内消息为空，要明确说明“风险来自仓库治理数据而非消息积压”

---

## 异常场景处理

### 1. 消息列表为空

这不是失败，直接写明当前没有未读消息即可。

### 2. PR/Issue 列表返回数量异常

优先相信结构化返回字段，并在报告里标注“已按客户端规则过滤”。

### 3. `workflow +repo-report` 无法获取部分指标

保留已有结果，并在结论中注明“部分指标按保守策略评分”。

### 4. 用户希望顺手处理 PR / Issue

先完成交接摘要，再单独征求确认，避免把只读交接和写操作混在一起。

---

## 最佳实践

- 交接报告优先使用 `--lang zh-CN`
- 需要给下一位维护者看的内容，优先用 Markdown 输出
- 需要给另一个 Agent 继续处理的内容，优先保留 JSON 结构化数据
- 如果只打算看单个 PR，不要用 handoff 技能替代 `gitlink-code-review`
