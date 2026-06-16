---
name: gitlink-issue-triage
version: 1.0.0
description: "Issue 智能分拣：扫描未分类 Issue，AI 按语义/关键词自动分类打标签、推荐并分配责任人，再用 notification 验证通知到位，最后批量产出分拣报告。当用户需要治理堆积 Issue、自动打标签、分配负责人或检查通知状态时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-issue-triage（Issue 智能分拣）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前（打标签、分配责任人、改状态），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 工作流概览

| 工作流 | 操作 | AI Agent 角色 | 写入 |
|--------|------|--------------|:----:|
| 工作流 1：自动分类打标签 | 扫描未分类 Issue → AI 判断类型 → 查/建标签 → 打标签 | 语义分类 + 标签创建 | 是 |
| 工作流 2：自动分配 + 通知 | 推荐责任人 → 分配 → 用 notification 验证通知 | 责任人推荐 | 是 |
| 工作流 3：批量分拣 + 报告 | 一次性处理所有未分类 Issue → 汇总报告 | 批处理 + 报告生成 | 是 |

---

## 分类规则表

AI 读 Issue 标题 + 描述后按以下规则分类（关键词只是辅助，**最终以语义为准**，能识别关键词未覆盖的同义表述）：

| Issue 关键词 / 语义 | 推荐标签 | 颜色 | 优先级 |
|---------------------|---------|------|:------:|
| bug / 错误 / 失败 / crash / 异常 / 报错 | bug | `#ee0701` | 🔴 高 |
| feature / 新增 / 建议 / 希望 / 能否支持 | enhancement | `#84b6eb` | 🔵 低 |
| 安全 / 漏洞 / 权限 / 泄露 / 注入 / XSS | security | `#b60205` | 🔴 高 |
| 性能 / 慢 / 卡顿 / 优化 / 内存 / OOM | performance | `#fbca04` | 🟡 中 |
| 文档 / README / 注释 / 示例 / 拼写 | documentation | `#0075ca` | 🔵 低 |
| question / 如何 / 怎么 / 请问 / ？ | question | `#cc317c` | 🟡 中 |

**默认/兜底标签：** `triage`（`#ededed`，灰）—— 无法明确归类时打上，等人工复核。

**分类决策原则：**
1. 安全类最高优先级（涉及漏洞即使同时是 bug 也归 security）
2. bug 优先于 enhancement（描述同时含两者时按 bug 处理）
3. 模糊的 feature/question 难以判断时归 question
4. 完全无法理解 → `triage`

---

## 工作流 1：自动分类打标签

**触发场景：** "帮我自动分拣这个仓库的新 Issue" / "给所有没标签的 Issue 打标签"

### Step 1：获取开放 Issue

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
```

### Step 2：AI 筛选"未分类"Issue

从返回结果中筛选出 `tags` 字段为空数组 `[]` 或缺失的 Issue（即没有任何标签）。已在 `gitlink-onboarding` 标过 `good first` 的 Issue 跳过，避免重复干预。

> 字段说明：`issue +list` 返回的 Issue 对象里，标签字段名是 **`tags`**（注意 `label +list` 用的是 `issue_tags`，两者不同）。每个 Issue 的 `number` 是网页 URL 显示的编号。

### Step 3：逐个读取详情用于分类

```bash
gitlink-cli issue +view --owner <owner> --repo <repo> --number <n> --format json
```

### Step 4：AI 按分类规则表判断类型

综合标题（`subject`）和描述（`description`）做语义分类，输出"类型 + 依据"。

### Step 5：查找或创建对应标签

```bash
# 先查现有标签，命中则复用 ID
gitlink-cli label +list --owner <owner> --repo <repo> --format json

# 仅当现有标签里没有对应分类时才创建（注意：GitLink 标签名限 15 字符）
gitlink-cli label +create --owner <owner> --repo <repo> \
  --name "bug" --color "#ee0701"
```

> ⚠️ **优先复用现有标签（含中文同义词）**：GitLink 仓库通常自带中文标签——`缺陷`(bug) / `功能`(enhancement) / `文档`(documentation) / `疑问`(question) / `协助`(help wanted)。**先匹配这些再考虑新建英文标签**，避免一个仓库里同时存在 `bug` 和 `缺陷` 两套语义重复的标签。颜色建议沿用现有标签的色值，保持视觉一致。

### Step 6：自动打标签

```bash
# --label 是"覆盖"语义：Issue 已有标签时必须把原 ID 一并传入
gitlink-cli issue +update --owner <owner> --repo <repo> \
  --number <n> --label <tag_id>[,<原标签id>...]
```

### Step 7：输出分类报告

```markdown
## 🏷️ Issue 分类报告 — <owner>/<repo>

📅 分拣时间：<YYYY-MM-DD HH:MM>

| Issue | 标题（节选） | 分类 | 依据 | 标签 ID |
|-------|------------|:----:|------|:------:|
| #12 | 登录后偶发 500 报错 | bug | "500 报错"语义 | 382700 |
| #13 | 希望支持 webhook 自定义 header | enhancement | "希望支持" | 382701 |

### 📊 汇总
- 处理：2 个未分类 Issue
- bug × 1（🔴 高）｜enhancement × 1（🔵 低）
- 兜底 triage：0 个
```

---

## 工作流 2：自动分配 + 通知

**触发场景：** "帮我给这些 Issue 分配责任人，并通知他们"

### Step 1：获取可分配的成员列表

```bash
gitlink-cli issue +assigners --owner <owner> --repo <repo> --format json
```

返回仓库协作成员（含 `id` 和 `login`），AI 据此推荐责任人。

> ⚠️ **个人仓库会返回空数组**（`"assigners": [], "total_count": 0`）。GitLink 的 `/issue_assigners` 只返回具有**显式项目角色**的成员（collaborator/manager 等），**不隐式包含 owner**。空数组不是命令失败，而是真实场景——此时按下方"列表为空"分支处理。

### Step 2：AI 推荐责任人

按 Issue 类型与成员专长做匹配（无成员画像时按公平轮询/按 Issue 类型分组）：

| Issue 类型 | 推荐策略 |
|-----------|---------|
| bug / security | 优先派给最近修过相关模块的成员 |
| documentation | 任意有空闲的成员 |
| question | 仓库 owner 或 maintainer |
| performance | 核心开发成员 |

如可分配列表为空（个人仓库、无 collaborator），跳过分配并在报告中标注"无可分配成员"。

### Step 3：分配责任人（Raw API）

> ⚠️ `issue +update` 当前不支持 `--assignee`（见下方"已知限制"），分配必须走 Raw API PATCH，并保留原 `subject`/`description`。

```bash
# Step 3a：先 GET 拿到当前 subject 和 description（避免被清空）
gitlink-cli issue +view --owner <owner> --repo <repo> --number <n> --format json

# Step 3b：PATCH 分配（assigned_to_id 用 assigners 返回的用户 id）
gitlink-cli api PATCH /v1/<owner>/<repo>/issues/<n> --body '{
  "subject": "<原 subject 原样回传>",
  "description": "<原 description 原样回传>",
  "assigned_to_id": <user_id>
}'
```

### Step 4：用 notification 验证通知到位

GitLink 在分配责任人时会**自动**给被分配人发一条站内消息。读取该成员的通知列表确认：

```bash
# ⚠️ --owner 必须填【当前登录账号】自己的 login，不能填被分配人的
# GitLink 平台限制：notification 只允许查自己的消息，跨用户查询会 403
gitlink-cli notification +list --owner <当前登录账号 login> --format json
```

在返回里查找 `source: ProjectIssue` 且 `notification_url` 含对应 Issue 编号的条目，确认通知已生成。

> ⚠️ **跨用户查询通知会被 403 拒绝**（实测：zhangqing23 查 ylly 的通知返回 `[403] 您没有权限进行该操作`）。所以**第三方无法代为验证**他人是否收到通知——这条限制在工作流 2 的报告中要如实告知用户："已分配给 X，X 是否收到通知需 X 本人 `notification +list --owner X` 自查"。

### Step 5：输出分配结果

```markdown
## 👥 责任人分配报告 — <owner>/<repo>

| Issue | 分类 | 责任人 | 通知状态 |
|-------|:----:|--------|:------:|
| #12 | bug | @zhangqing | ✅ 已通知 |
| #13 | enhancement | @ylly | ✅ 已通知 |

> 责任人可在 GitLink 网页对应 Issue 页右侧"负责人"栏查看。
```

---

## 工作流 3：批量分拣 + 报告

**触发场景：** "把仓库里所有没分类的 Issue 一次性处理掉"

### Step 1：批量拉取未分类 Issue

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
# AI 客户端过滤 tags 为空的 Issue
```

### Step 2：对每个 Issue 执行"工作流 1 + 工作流 2"

依次分类打标签 → 推荐并分配责任人。**批量写入前先给用户一份预览清单**，得到确认后再批量执行。

### Step 3：生成分拣总报告

```markdown
## 📋 Issue 智能分拣总报告 — <owner>/<repo>

📅 处理时间：<YYYY-MM-DD HH:MM>
🎯 处理范围：所有开放且未分类的 Issue

### 分类分布
| 类型 | 数量 | 占比 |
|------|:----:|:----:|
| 🔴 bug | 3 | 30% |
| 🔴 security | 1 | 10% |
| 🟡 performance | 2 | 20% |
| 🔵 enhancement | 3 | 30% |
| 🔵 documentation | 1 | 10% |
| ⚪ triage（待人工） | 0 | 0% |
| **合计** | **10** | **100%** |

### 责任人分配
| 责任人 | 分到 | 涉及 Issue |
|--------|:----:|-----------|
| @zhangqing | 4 | #12 #15 #18 #20 |
| @ylly | 3 | #13 #14 #17 |
| 待分配 | 3 | #16 #19 #21（无可分配成员） |

### ⚠️ 需人工跟进
- #21 描述过于模糊 → 已打 `triage`，需 owner 复核
- #19 涉及架构重构 → 已打 `enhancement` 但建议核心成员评估

### 🔗 网页验证
- Issue 列表：https://gitlink.org.cn/<owner>/<repo>/issues
- 标签视图：https://gitlink.org.cn/<owner>/<repo>/issues/tags
```

---

## Raw API 参考

```bash
# 分配责任人（issue +update 当前不支持 --assignee，必须走 Raw API）
gitlink-cli api PATCH /v1/<owner>/<repo>/issues/<n> --body '{
  "subject": "<原标题>", "description": "<原描述>", "assigned_to_id": <user_id>
}'

# 查询可分配成员
gitlink-cli issue +assigners --owner <owner> --repo <repo> --format json

# 查询某用户的通知（--owner 填该用户自己的 login）
gitlink-cli notification +list --owner <assignee_login> --format json

# 把通知标记为已读（如需）
gitlink-cli notification +read --owner <assignee_login> --id <notification_id>
```

---

## 注意事项

- **写操作前确认：** 打标签（`issue +update --label`）和分配责任人（`api PATCH`）会真实修改 Issue，批量执行前先给预览清单等用户确认。
- **--label 是覆盖语义：** `issue +update --label <id>` 会替换原有标签。若 Issue 已有标签（例如 `good first`），必须把原标签 ID 一并传入（如 `--label 382660,382700`），否则原标签会丢失。
- **标签颜色格式：** `label +create --color` 必须带 `#` 号（如 `#ee0701`）。
- **标签名长度限制：** GitLink 标签名上限 **15 字符**。中文标签（如"文档"）通常没问题，英文长名（如"enhancement" 11 字符 OK，"good first issue" 16 字符会被截断）需注意。
- **`issue +update` 不支持 `--assignee`：** 当前 Shortcut 的 update 子命令仅支持 `--title/--body/--state/--label`。分配责任人需走 Raw API `PATCH /v1/:owner/:repo/issues/:n`，且必须带上原 `subject` 和 `description`（参考 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) API 注意事项：Issue 更新不带 subject/description 可能被清空）。
- **PowerShell 跑 Raw API --body JSON 会被吞双引号：** Windows PowerShell 5 把含 `"` 的字符串传给原生 exe 时会 strip 引号，导致 `encoding/json` 解析失败（报错 `invalid character 's' looking for beginning of object key string`）。**改用 Git Bash 或 cmd.exe 跑同一条命令可正常通过**（bash 单引号原样保留 JSON）。
- **`assigned_to_id` 用数字 ID：** 不是 login 字符串。从 `issue +assigners` 返回里取 `id` 字段。若 `assigners` 为空（个人仓库），可改用仓库 owner 的 user_id（从 `repo +info` 或 `issue +view` 的 `author.id` 字段拿）。
- **notification +list 是自查询限定：** 该命令查 `/users/<login>/messages`，**GitLink 平台只允许用户查询自己的通知**，跨用户查询返回 `[403] 您没有权限进行该操作`（实测：zhangqing23 查 ylly 的通知被拒）。因此无法第三方代为验证通知到达，只能由责任人本人自查。
- **分配会自动触发通知：** GitLink 平台在 `assigned_to_id` 变更时会自动给被分配人发站内消息，**无需也不存在** "send notification" 命令。`notification +list` 只用于**验证**通知已生成（且只能自验证）。
- **`assigners` 字段两个位置：** Issue 对象里 `assigners` 是已分配人列表（数组），`issue +assigners` 命令返回的是**可分配的候选人**列表。两者不同，别混淆。
- **`--number` 是网页编号：** 用 GitLink 网页 URL 中显示的编号（`/issues/<n>`），不是数据库主键。
- **不要和 `gitlink-onboarding` 冲突：** 已被 onboarding 标记为 `good first` 的 Issue 通常已分类，分拣时可跳过，避免覆盖标签。
- **分类不是终审：** AI 分类有误判可能，对"模棱两可"或"高严重度"的 Issue 建议同时打 `triage` 让人工复核，或在报告中明确标注不确定项。
