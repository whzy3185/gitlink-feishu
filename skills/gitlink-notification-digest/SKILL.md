---
name: gitlink-notification-digest
version: 1.0.0
description: "通知摘要：汇总 GitLink 通知并按类型分类，生成通知摘要报告，支持批量标记已读。当用户需要查看通知摘要、整理通知、清理未读通知时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli notification --help"
---

# gitlink-notification-digest（通知摘要）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — `notification +read` 和 `+read-all` 为写操作，执行前需确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

帮助用户高效管理 GitLink 通知：

1. **通知列表** — 获取所有未读通知
2. **自动分类** — 按类型（Issue/PR/评论/系统）分组
3. **优先级判断** — 识别需要立即处理的通知
4. **批量操作** — 支持标记已读（需确认）
5. **摘要报告** — 生成结构化通知摘要

---

## 工作流：通知摘要

### Step 1：获取通知列表

```bash
gitlink-cli notification +list --format json
```

获取参数：
- 默认获取未读通知
- 如需全部通知（含已读）：`--all`
- 如需仅参与的通知：`--participating`
- 分页：`--page 2 --limit 20`

提取每条通知的：
- `id` — 通知 ID（用于 `+read` 单条标记已读）
- `content` — 通知内容（HTML 格式，从中提取摘要文本）
- `source` — 通知来源类型（如 IssueAtme、PullRequestAssigned、ProjectForked 等）
- `notification_url` — 通知链接，从中解析关联仓库（提取 URL path 中的 `/owner/repo/` 段）
- `created_at` — 通知时间
- `status` — 状态（1=未读，2=已读）

### Step 2：分类与优先级

#### 2.1 按类型分类（优先使用 `source` 字段）

| 类型 | `source` 字段匹配 | 处理建议 |
|------|------------------|----------|
| 🔴 **@提及** | 含 `Atme`（如 IssueAtme, PullRequestAtme） | 立即查看回复 |
| 🟡 **Issue 更新** | 含 `Issue`（如 IssueAssigned, IssueClosed） | 当天处理 |
| 🟢 **PR 更新** | 含 `PullRequest`（如 PullRequestAssigned, PullRequestMerged） | 跟进代码 |
| 🔵 **系统通知** | 含 `Project`/`Organization`（如 ProjectForked, ProjectJoined） | 知悉即可 |
| ⚪ **其他** | 不匹配以上 | 按需查看 |

> `source` 字段返回的是结构化枚举值（如 `IssueAtme`），优先以此分类。`content` 字段为 HTML 文本，仅作补充参考。

#### 2.2 优先级排序

| 优先级 | 判定 |
|--------|------|
| **P0 - 立即** | @提及 + 来自自己参与的 Issue/PR |
| **P1 - 今天** | 自己创建的 Issue/PR 有新回复，或分配的 Issue 有更新 |
| **P2 - 本周** | 关注的仓库有新动态 |
| **P3 - 可忽略** | 系统通知、已解决的 Issue |

### Step 3：生成通知摘要

按模板输出。

### Step 4：批量标记已读（可选，需确认）

```bash
# 标记全部已读
gitlink-cli notification +read-all --format json

# 标记单条已读
gitlink-cli notification +read --id <notification_id> --format json
```

> ⚠️ **执行前必须确认用户意图** — `+read` 和 `+read-all` 为写操作。

---

## 输出模板

```markdown
# 🔔 通知摘要

> 生成时间：{{当前时间}}
> 未读通知：{{unread_count}} 条 / 总计：{{total_count}} 条

---

## 一、概要

| 类型 | 未读 | 总计 |
|------|------|------|
| @提及 | {{mention_unread}} | {{mention_total}} |
| Issue 更新 | {{issue_unread}} | {{issue_total}} |
| PR 更新 | {{pr_unread}} | {{pr_total}} |
| 系统通知 | {{system_unread}} | {{system_total}} |
| 其他 | {{other_unread}} | {{other_total}} |

---

## 二、需要立即处理（P0）

> 如无，输出：*🎉 无紧急通知。*

| # | 类型 | 仓库 | 内容摘要 | 时间 |
|---|------|------|----------|------|
| 1 | 🔴@提及 | {{repo}} | {{summary}} | {{time}} |

---

## 三、今天处理（P1）

> 如无，输出：*无待处理通知。*

| # | 类型 | 仓库 | 内容摘要 | 时间 |
|---|------|------|----------|------|

---

## 四、本周关注（P2）

> 如无，输出：*无需要本周关注的通知。*

---

## 五、可忽略（P3）

> 如本段被折叠，输出：*{{p3_count}} 条低优先级通知，已折叠。*

---

## 六、通知趋势

| 时间段 | 通知数 |
|--------|--------|
| 今日 | {{today_count}} |
| 昨日 | {{yesterday_count}} |
| 本周 | {{week_count}} |
| 上周 | {{last_week_count}} |

---

## 操作建议

- 建议标记已读：{{suggest_read_count}} 条 P3 通知
- 需要回复/处理：{{need_action_count}} 条 P0/P1 通知

如需标记全部已读，我可以执行：
`gitlink-cli notification +read-all`
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 无未读通知 | 输出"🎉 所有通知已处理完毕" |
| 通知数量 > 50 | 分批获取（page 1/2/3），优先分析最近 50 条 |
| `notification +list` 返回空 | 检查认证状态（参考 gitlink-shared） |

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **`+read` 和 `+read-all` 为写操作**，执行前必须确认用户意图
- ✅ **本 Skill 默认只读分析**，仅在用户明确要求时标记已读
- ⚠️ **通知类型依赖标题关键词推断**，实际类型可能有偏差
- ⚠️ **通知可能分页**，数量 >20 时需追加 `--page 2` 等
