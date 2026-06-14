---
name: gitlink-stale-issue-manager
version: 1.0.0
description: "过期 Issue 管理：自动识别长期无活动的 Issue，按过期等级标记/提醒/批量关闭，支持白名单保护和干运行模式。当用户需要清理过期 Issue、管理社区积压、维护仓库活跃度时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-stale-issue-manager（过期 Issue 管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 关闭操作不可逆，务必先以干运行模式确认再执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

---

## 功能概述

本技能解决活跃仓库的 Issue 积压问题，提供完整的过期项管理流程：

1. **过期扫描** — 获取所有打开的 Issue，按最后活动时间分类
2. **分级标记** — 按过期天数自动打标签和发提醒评论
3. **批量关闭** — 对超期严重的 Issue 执行关闭
4. **白名单保护** — 保护特定标签的 Issue 不被关闭
5. **干运行模式** — 先预览操作结果，确认后再执行
6. **执行报告** — 统计本次操作的详细结果

---

## 一、过期扫描：获取所有打开的 Issue 

### 1.1 获取 Issue 列表

```bash
# 获取所有打开的 Issue， 如果列表数量过大（>20），需要翻页处理
gitlink-cli issue +list --state open --owner <owner> --repo <repo> --page 1 --limit 20 --format json
```

**AI 必须提取的关键字段**：
- `number`：Issue ID（后续打标签/评论/关闭时使用）
- `subject`：Issue 标题（用于判断是否值得保留）
- `created_at`：创建时间
- `updated_at`：最后更新时间（**核心判断依据**）
- `tags`：已有标签（打标签时需要保留原有tags / 白名单判断）


### 1.2 获取 Issue 评论（精确判断最后活动时间）

```bash
# 获取某个 Issue 的评论列表（如果数据量过大，可能需要翻到最后一页取最后十条）
gitlink-cli api GET /v1/:owner/:repo/issues/:number/journals?category=comment&page=1&limit=50 --format json
```

**AI 判断逻辑**：

```
只有当 Issue 的 created_at 和 updated_at 不同时，才需要查询评论列表。
如果 created_at == updated_at，说明没有任何更新，直接使用 created_at 作为最后活动时间。
```

**评论列表返回数据结构**：

| 字段 | 类型 | 说明 |
|------|------|------|
| journals | array | 评论列表 |
| journals[].id | integer | 评论 ID |
| journals[].notes | string | 评论内容 |
| journals[].created_at | string | 评论创建时间 |
| journals[].user | object | 评论者信息 |

> **注意**：评论列表按创建时间正序排列，取最后一条即为最新评论。

### 1.4 计算过期天数

```
过期天数 = 当前日期 - 最后活动日期

最后活动日期的确定优先级：
1. Issue 最新一条评论的时间（优先）— 只有当 created_at != updated_at 时才查询评论
2. Issue 的 updated_at 字段（其次）
3. Issue 的 created_at 字段（保底）

判断流程：
├─ created_at == updated_at？
│  └─ 是 → 最后活动日期 = created_at（无任何更新，无需查询评论）
│  └─ 否 → 查询评论列表
│     ├─ 有评论？→ 最后活动日期 = 最新一条评论的 created_at
│     └─ 无评论？→ 最后活动日期 = updated_at
```

> **注意**：此处需要额外注意，必须排除掉该skill自动发送的评论！如果排除掉之后没有其他评论，则以create_at作为最后活动日期。

---

## 二、分级标记：按过期等级自动处理

### 2.1 过期等级定义

| 等级 | 过期天数 | 标签 | 操作 | 评论内容 |
|------|---------|------|------|---------|
| 🟢 健康 | 0-29 天 | 无 | 无 | 无 |
| 🟡 迟缓 | 30-59 天 | `迟缓` | 打标签 + 发提醒 | 温和提醒，7天内回复可移除标签 |
| 🟠 不活跃 | 60-89 天 | `不活跃` | 更新标签 + 再次提醒 | 严重警告，即将被关闭 |
| 🔴 过期 | ≥90 天 | `过期` | 更新标签 + 关闭 | 关闭通知，可随时重新打开 |

### 2.2 打标签操作

**CRITICAL — 打标签前必须先确保标签存在，不存在则创建。**

#### 2.2.1 标签预创建流程

```
打标签前，必须执行以下流程：
1. 获取仓库现有标签列表
2. 检查目标标签（迟缓/不活跃/过期）是否已存在
3. 如果标签不存在，先创建标签
4. 获取标签 ID 后，再为 Issue 打标签
```

```bash
# Step 1：获取仓库现有标签列表
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 2：检查目标标签是否已存在（AI 在返回结果中查找）
# 如果目标标签不存在，则创建：

# 创建"迟缓"标签（30-59天）
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"迟缓","description":"近期活动频率明显下降，需关注但尚未停滞","color":"#fbca04"}' --format json

# 创建"不活跃"标签（60-89天）
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"不活跃","description":"长期无更新或互动，可能已失去推进动力","color":"#d93f0b"}' --format json

# 创建"过期"标签（≥90天）
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"过期","description":"已超出合理响应周期，建议关闭或重新评估","color":"#b60205"}' --format json

# Step 3：重新获取标签列表，确认标签 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
```

**标签与过期等级对应关系**：

| 过期等级 | 标签名 | 颜色 | 描述 |
|---------|--------|------|------|
| 🟡 迟缓（30-59天） | 迟缓 | `#fbca04` | 近期活动频率明显下降，需关注但尚未停滞 |
| 🟠 不活跃（60-89天） | 不活跃 | `#d93f0b` | 长期无更新或互动，可能已失去推进动力 |
| 🔴 过期（≥90天） | 过期 | `#b60205` | 已超出合理响应周期，建议关闭或重新评估 |

#### 2.2.2 为 Issue 打标签

```bash
# 为 Issue 打标签，通过 PATCH 修改 Issue 的 issue_tag_ids 字段
# 注意：issue_tag_ids 是数组，追加标签时需包含已有标签 ID
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json

# 示例：为 Issue 追加"迟缓"标签（假设迟缓标签 ID 为 374033）
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[374033]}' --format json

# 示例：Issue 已有标签 ID 315216，追加迟缓标签 ID 374033
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[315216,374033]}' --format json
```

> **⚠️ 重要**：`issue_tag_ids` 是完整替换而非追加，设置时必须包含 Issue 已有的所有标签 ID，否则已有标签会被移除。操作前需先查询 Issue 当前的标签列表。

### 2.3 提醒评论模板

**🟡 30-59 天（stale 提醒）**：

```markdown
**⚠️ Stale Issue 提醒**

此 Issue 已 **{days}** 天无活动。

为保持仓库 Issue 列表的整洁，如果 **7 天内** 没有新的回复，此 Issue 将被标记为 `不活跃`。

**如果你仍在关注此问题**，请留下一条评论（哪怕只是 "仍在关注"），即可重置活动计时。

---
*此消息由 [gitlink-stale-issue-manager] 自动发送*
```

**🟠 60-89 天（inactive 警告）**：

```markdown
**🔴 Inactive Issue 警告**

此 Issue 已 **{days}** 天无活动。

如果 **7 天内** 没有新的回复，此 Issue 将被自动关闭。

**如何保留此 Issue**：
- 留下评论说明当前进展
- 分配责任人
- 添加 `置顶` 标签永久保留

---
*此消息由 [gitlink-stale-issue-manager] 自动发送*
```

**🔴 ≥90 天（关闭通知）**：

```markdown
**🔒 过期 Issue 关闭通知**

此 Issue 已 **{days}** 天无活动，现被自动关闭。

这不是对问题本身的否定，而是为了保持 Issue 列表的清晰度。

**如果你认为此问题仍然有效**：
- 留下评论说明原因
- 项目维护者可随时重新打开此 Issue

---
*此消息由 [gitlink-stale-issue-manager] 自动发送*
```

### 2.4 发送提醒评论

```bash
# 发送提醒评论
gitlink-cli issue +comment \
  --number <issue_id> \
  --owner <owner> \
  --repo <repo> \
  --body "<上述评论模板内容>"
```

---

## 三、批量关闭：对超期严重的 Issue 执行关闭

### 3.1 关闭单个 Issue

```bash
# 关闭 Issue
gitlink-cli issue +close \
  --number <issue_id> \
  --owner <owner> \
  --repo <repo>
```

### 3.2 批量关闭流程

**AI 必须遵循以下流程**，禁止跳步：

```
Step 1：扫描 — 获取所有 ≥90 天无活动的 Issue
Step 2：过滤 — 排除白名单中的 Issue（见第四章）
Step 3：预览 — 列出将被关闭的 Issue（干运行模式）
Step 4：确认 — 询问用户是否确认关闭
Step 5：执行 — 逐一发送关闭评论 + 关闭 Issue
Step 6：报告 — 输出本次操作统计
```

### 3.3 关闭前必须发送通知评论

**CRITICAL**：关闭 Issue 前必须先发送关闭通知评论，再执行关闭操作。

```bash
# 先发评论
gitlink-cli issue +comment \
  --number <issue_id> \
  --owner <owner> \
  --repo <repo> \
  --body "<关闭通知模板>"

# 再关闭
gitlink-cli issue +close \
  --number <issue_id> \
  --owner <owner> \
  --repo <repo>
```

---

## 四、白名单保护：防止误关闭重要 Issue

### 4.1 白名单标签

以下标签的 Issue **不会被标记为过期，也不会被关闭**：

| 标签 | 含义 | 保护级别 |
|------|------|---------|
| `pinned` | 置顶/长期跟踪 | 永久保护 |
| `security` | 安全相关 | 永久保护 |
| `bug` | 确认的 Bug | 永久保护 |
| `enhancement` | 已确认的功能需求 | 永久保护 |
| `help-wanted` | 寻求社区帮助 | 永久保护 |
| `good-first-issue` | 新人友好 | 永久保护 |
| `wontfix` | 不修复但需保留 | 永久保护 |

### 4.2 白名单判断逻辑

```
对每个 Issue：
1. 读取其标签列表（tags 字段）
2. 如果包含白名单标签中的任意一个 → 跳过，不做任何操作
3. 如果不包含白名单标签 → 按过期等级处理
```

### 4.3 自定义白名单

用户可指定额外的保护标签：

```bash
# 用户可以在对话中指定自定义白名单标签，如果不指定，则使用默认的标签
```

### 4.4 Issue 标题关键词保护

以下标题关键词的 Issue 也应保护（即使无白名单标签）：

```
保护关键词（标题包含即跳过）：
- "[Security]" / "[安全]"
- "[Pinned]" / "[长期]"
- "[Tracking]" / "[跟踪]"
- "严重" / "紧急" / "critical" / "urgent"
```

---

## 五、干运行模式：先预览再执行

### 5.1 干运行逻辑

**CRITICAL**：首次执行时必须使用干运行模式，让用户确认后再真正执行。

```
干运行模式下，AI 仅输出以下信息，不执行任何写操作：
1. 将被标记为 stale 的 Issue 列表（30-59 天）
2. 将被标记为 inactive 的 Issue 列表（60-89 天）
3. 将被关闭的 Issue 列表（≥90 天）
4. 被白名单保护的 Issue 列表
5. 本次操作统计
```

### 5.2 干运行报告格式

```markdown
## 🔍 过期 Issue 扫描报告（干运行）

**仓库**：`<owner>/<repo>`
**扫描时间**：2026-06-11
**扫描范围**：所有打开的 Issue

---

### 📊 统计概览

| 类别 | 数量 |
|------|------|
| 打开的 Issue | 42 |
| 🟡 迟缓（30-59天） | 12 |
| 🟠 不活跃（60-89天） | 5 |
| 🔴 过期（≥90天） | 3 |
| 🛡️ 白名单保护 | 4 |

---

### 🟡 将标记为 `迟缓`（12 个）

| # | Issue | 标题 | 最后活动 | 过期天数 |
|---|-------|------|---------|---------|
| 1 | #156 | 文档中示例代码过期 | 2026-04-25 | 47 |
| 2 | #178 | 请求支持暗色模式 | 2026-04-18 | 54 |
| ... | ... | ... | ... | ... |

### 🟠 将标记为 `不活跃`（5 个）

| # | Issue | 标题 | 最后活动 | 过期天数 |
|---|-------|------|---------|---------|
| 1 | #98 | 首页加载速度优化 | 2026-03-15 | 88 |
| ... | ... | ... | ... | ... |

### 🔴 将被关闭（3 个）

| # | Issue | 标题 | 最后活动 | 过期天数 |
|---|-------|------|---------|---------|
| 1 | #45 | 旧版 API 兼容问题 | 2025-12-20 | 173 |
| 2 | #67 | 建议添加 X 功能 | 2025-11-05 | 218 |
| 3 | #89 | 拼写错误 | 2025-10-01 | 253 |

### 🛡️ 白名单保护（4 个）

| # | Issue | 标题 | 保护原因 |
|---|-------|------|---------|
| 1 | #12 | [Security] XSS 漏洞 | 标签：security |
| 2 | #34 | 跟踪 v2.0 发布计划 | 标签：pinned |
| 3 | #56 | 用户认证失败 | 标签：bug |
| 4 | #78 | 添加国际化支持 | 标题含 "紧急" |

---

### ⚠️ 即将执行的操作

1. 对 12 个 Issue 添加 `迟缓` 标签并发送提醒评论
2. 对 5 个 Issue 更新为 `不活跃` 标签并发送警告评论
3. 对 3 个 Issue 发送关闭通知并关闭

**确认后将执行以上操作。是否继续？**
```

---

## 六、执行步骤总览

### 6.1 完整流程

```bash
# Step 1：获取所有打开的 Issue
gitlink-cli issue +list --state open --owner <owner> --repo <repo> --format json

# Step 2：判断 Issue 最后活动时间
#   a. 如果 created_at == updated_at → 最后活动时间 = created_at（无需查询评论）
#   b. 如果 created_at != updated_at → 查询评论列表获取最新评论时间
gitlink-cli api GET /v1/:owner/:repo/issues/:number/journals --format json
#   - 有评论 → 最后活动时间 = 最新评论的 created_at
#   - 无评论 → 最后活动时间 = updated_at
# 需注意排除skill自动发送的评论内容

# Step 3：AI 计算过期天数，按等级分类，排除白名单

# Step 4（干运行）：输出扫描报告，等待用户确认

# Step 5（执行 — 用户确认后）：
#   a. 预创建标签（确保标签存在）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
# 检查迟缓/不活跃/过期标签是否存在，不存在则创建：
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"迟缓","description":"近期活动频率明显下降，需关注但尚未停滞","color":"#fbca04"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"不活跃","description":"长期无更新或互动，可能已失去推进动力","color":"#d93f0b"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"过期","description":"已超出合理响应周期，建议关闭或重新评估","color":"#b60205"}' --format json

#   b. 对 30-59 天 Issue：打"迟缓"标签 + 发提醒评论
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json
gitlink-cli issue +comment --number <issue_id> --owner <owner> --repo <repo> --body "<stale 提醒>"

#   c. 对 60-89 天 Issue：打"不活跃"标签 + 发警告评论
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json
gitlink-cli issue +comment --number <issue_id> --owner <owner> --repo <repo> --body "<inactive 警告>"

#   d. 对 ≥90 天 Issue：打"过期"标签 + 发关闭通知 + 关闭
gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}' --format json
gitlink-cli issue +comment --number <issue_id> --owner <owner> --repo <repo> --body "<关闭通知>"
gitlink-cli issue +close --number <issue_id> --owner <owner> --repo <repo>
```

### 6.2 可选：仅扫描模式

如果用户只想查看过期情况，不执行任何操作：

```bash
# 仅扫描，输出报告，不修改任何 Issue
# 在干运行报告末尾提示："本次仅扫描，未修改任何 Issue。如需执行，请告知。"
```

### 6.3 可选：按标签过滤扫描

如果用户只想扫描特定类型的 Issue：

```bash
# 获取所有打开的 Issue（客户端按标签过滤）
gitlink-cli issue +list --state open --owner <owner> --repo <repo> --format json
# AI 在结果中筛选包含特定标签的 Issue
```

---

## 七、执行报告

操作完成后，输出以下格式的执行报告：

```markdown
## ✅ 过期 Issue 管理执行报告

**仓库**：`<owner>/<repo>`
**执行时间**：2026-06-11 18:30
**执行模式**：正式执行 / 仅扫描

---

### 📊 操作统计

| 操作 | 数量 | 成功 | 失败 |
|------|------|------|------|
| 添加 迟缓 标签 | 12 | 12 | 0 |
| 添加 不活跃 标签 | 5 | 5 | 0 |
| 发送提醒评论 | 17 | 17 | 0 |
| 关闭 Issue | 3 | 3 | 0 |
| 白名单保护跳过 | 4 | - | - |

### 📋 已关闭的 Issue

| # | Issue | 标题 | 过期天数 | 关闭状态 |
|---|-------|------|---------|---------|
| 1 | #45 | 旧版 API 兼容问题 | 173 | ✅ 已关闭 |
| 2 | #67 | 建议添加 X 功能 | 218 | ✅ 已关闭 |
| 3 | #89 | 拼写错误 | 253 | ✅ 已关闭 |

### ⚠️ 失败记录

（无失败记录）

---

### 📈 仓库健康度变化

| 指标 | 操作前 | 操作后 | 变化 |
|------|--------|--------|------|
| 打开的 Issue | 42 | 39 | -3 |
| 过期 Issue 占比 | 47.6% | 38.5% | -9.1% |

---

*下次建议执行时间：7 天后（2026-06-18）*
```

---

## 八、可配置参数

用户可在对话中指定以下参数调整行为：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `stale_days` | 30 | 标记为 stale 的天数阈值 |
| `inactive_days` | 60 | 标记为 inactive 的天数阈值 |
| `expire_days` | 90 | 自动关闭的天数阈值 |
| `grace_period` | 7 | 标记后等待回复的天数（stale → inactive 的缓冲期） |
| `dry_run` | true | 是否为干运行模式（首次必须为 true） |
| `close_expired` | false | 是否关闭过期 Issue（需用户显式确认后改为 true） |
| `protect_labels` | pinned,security,bug,enhancement,help-wanted,good-first-issue,wontfix | 白名单标签 |

### 配置示例

```
用户："扫描过期 Issue，stale 设为 45 天，不关闭 "

AI 应解析为：
- stale_days = 45
- inactive_days = 75
- expire_days = 105
- close_expired = false
- dry_run = true（首次必须）
```

---


## 注意事项

- ✅ **首次执行必须干运行**：先输出预览报告，用户确认后再执行
- ✅ **关闭前必须发评论**：给 Issue 作者留下重新打开的途径
- ✅ **白名单保护不可绕过**：即使过期天数超过阈值，白名单内的 Issue 也不处理
- ✅ **批量操作逐条执行**：避免 API 限流，每条操作间隔 1 秒
- ✅ **关闭操作不可逆**：虽然维护者可以重新打开，但评论通知已发出，应谨慎
- ✅ **建议定期执行**：推荐每周执行一次，保持 Issue 列表健康
- ⚠️ **标签操作**：打标签通过 `gitlink-cli api PATCH /v1/:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}'` 完成，`issue_tag_ids` 为完整替换，需包含已有标签 ID
- ⚠️ **标签预创建**：打标签前必须先查询标签列表，确认目标标签存在，不存在则先通过 `POST /v1/:owner/:repo/issue_tags` 创建
