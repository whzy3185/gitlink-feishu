---
name: gitlink-notification-digest
version: 2.0.0
description: "通知摘要：汇总 GitLink 通知并按类型分类，生成通知摘要报告，支持批量标记已读。当用户需要查看通知摘要、整理通知、清理未读通知时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
---

# gitlink-notification-digest（通知摘要）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 标记已读为写操作，执行前需确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。
> **执行样例：** 参见 [`EXAMPLES.md`](EXAMPLES.md)

---

## 功能概述

帮助用户高效管理 GitLink 通知（GitLink 平台称为「消息」）：

1. **通知列表** — 获取所有未读通知
2. **自动分类** — 按 `source` 字段分类（Issue/PR/系统等）
3. **优先级判断** — 识别需要立即处理的通知
4. **批量操作** — 支持标记已读（需确认）
5. **摘要报告** — 生成结构化通知摘要

---

## ⚠️ 关键注意事项

### CLI 路径处理 Bug

**`gitlink-cli api` 的路径参数不要以 `/` 开头**，否则会被错误解析为本地文件路径。

```bash
# ❌ 错误 — 路径以 / 开头会被解析为 D:/Applications/Git/...
gitlink-cli api GET /users/me

# ✅ 正确 — 去掉前导 /
gitlink-cli api GET "users/{owner}/messages.json"
```

### 术语对照

GitLink 平台用「**消息**」（messages）而不是「通知」（notifications）。API 端点和字段均使用 `messages`。

---

## 工作流：通知摘要

### Step 1：获取通知列表

使用 Raw API 调用 `/api/users/{owner}/messages.json`：

```bash
# 获取未读通知（status=1 表示未读，2 表示已读）
gitlink-cli api GET "users/{owner}/messages.json" --query "status=1&limit=20" --format json

# 获取全部通知（含已读）
gitlink-cli api GET "users/{owner}/messages.json" --query "limit=20" --format json

# 分页获取
gitlink-cli api GET "users/{owner}/messages.json" --query "status=1&page=2&limit=20" --format json

# 按类型过滤
# type=notification  系统消息（仓库动态、PR、Issue 等）
# type=atme          @我消息
gitlink-cli api GET "users/{owner}/messages.json" --query "type=atme&status=1&limit=20" --format json
```

**参数说明：**

| 参数 | 位置 | 说明 |
|------|------|------|
| `{owner}` | Path | 当前用户名（从 `gitlink-cli auth status` 获取） |
| `status` | Query | 1=未读，2=已读，不传=全部 |
| `type` | Query | `notification`=系统消息，`atme`=@我消息，不传=全部 |
| `page` | Query | 页码（默认 1） |
| `limit` | Query | 每页条数（默认 20） |

**响应结构：**

```json
{
  "total_count": 28,
  "type": "",
  "unread_notification": 7,
  "unread_atme": 0,
  "messages": [
    {
      "id": 740214,
      "status": 1,
      "content": "jiangtx在 <b>jiangtx/gitlink-cli</b> 提交了一个合并请求：<b>label 模块新建</b>",
      "notification_url": "https://www.gitlink.org.cn/jiangtx/gitlink-cli/pulls/15347",
      "source": "ProjectPullRequest",
      "created_at": "2026-06-03 00:27:37",
      "time_ago": "10小时前",
      "type": "notification",
      "sender": {
        "id": 113,
        "type": "User",
        "name": "jiangtx",
        "login": "jiangtx",
        "image_url": "..."
      }
    }
  ]
}
```

**提取字段：**
- `id` — 消息 ID（用于标记已读）
- `content` — HTML 格式的通知内容
- `source` — 通知来源类型（枚举值，见下方分类表）
- `notification_url` — 跳转链接，可从中解析仓库（提取 URL 中的 `/owner/repo/` 段）
- `created_at` — 通知时间（格式 `YYYY-MM-DD HH:mm:ss`）
- `status` — 1=未读，2=已读
- `type` — `notification` 或 `atme`
- `sender` — 发送者信息（login, name, image_url）

### Step 2：分类与优先级

#### 2.1 按 `source` 字段分类

| 类型 | `source` 枚举值 | 处理建议 |
|------|-----------------|----------|
| 🔴 **@提及** | `IssueAtme`, `PullReuqestAtme`（注意官方 API 拼写如此） | 立即查看回复 |
| 🟡 **Issue 更新** | `IssueAssigned`, `IssueExpire`, `IssueChanged`, `IssueDeleted`, `IssueJournal`, `ProjectIssue` | 当天处理 |
| 🟢 **PR 更新** | `PullRequestAssigned`, `PullRequestChanged`, `PullRequestClosed`, `PullRequestJournal`, `PullRequestMerged`, `ProjectPullRequest` | 跟进代码 |
| 🔵 **系统通知** | `ProjectJoined`, `ProjectLeft`, `ProjectMemberJoined`, `ProjectMemberLeft`, `ProjectForked`, `ProjectPraised`, `ProjectRole`, `ProjectFollowed`, `ProjectDeleted`, `ProjectTransfer`, `ProjectSettingChanged`, `ProjectMilestone`, `ProjectMilestoneCompleted`, `ProjectVersion`, `OrganizationJoined`, `OrganizationLeft`, `OrganizationRole`, `ProjectOpenDevOps` | 知悉即可 |
| ⚪ **其他** | `LoginIpTip` 及未列出的值 | 按需查看 |

**完整 `source` 枚举参考：**

<details>
<summary>展开查看全部 source 枚举值</summary>

| 枚举值 | 含义 |
|--------|------|
| `IssueAssigned` | 有新指派给我的疑修 |
| `IssueExpire` | 我创建或负责的疑修截止日期到达最后一天 |
| `IssueAtme` | 在疑修中@我 |
| `IssueChanged` | 我创建或负责的疑修状态变更 |
| `IssueDeleted` | 我创建或负责的疑修删除 |
| `IssueJournal` | 我创建或负责的疑修有新的评论 |
| `LoginIpTip` | 登录 IP 提示 |
| `OrganizationJoined` | 加入组织 |
| `OrganizationLeft` | 离开组织 |
| `OrganizationRole` | 组织角色变更 |
| `ProjectDeleted` | 项目被删除 |
| `ProjectFollowed` | 有人关注了项目 |
| `ProjectForked` | 项目被 Fork |
| `ProjectIssue` | 项目新 Issue |
| `ProjectJoined` | 加入项目 |
| `ProjectLeft` | 离开项目 |
| `ProjectMemberJoined` | 新成员加入项目 |
| `ProjectMemberLeft` | 成员离开项目 |
| `ProjectMilestoneCompleted` | 里程碑完成 |
| `ProjectMilestone` | 新里程碑 |
| `ProjectOpenDevOps` | DevOps 引擎开通 |
| `ProjectPraised` | 项目被点赞 |
| `ProjectPullRequest` | 项目新 PR |
| `ProjectRole` | 项目角色变更 |
| `ProjectSettingChanged` | 项目设置变更 |
| `ProjectTransfer` | 项目转让 |
| `ProjectVersion` | 新版本发布 |
| `PullRequestAssigned` | 有指派给我的 PR |
| `PullReuqestAtme` | 在 PR 中@我（**官方拼写如此**） |
| `PullRequestChanged` | PR 状态变更 |
| `PullRequestClosed` | PR 被关闭 |
| `PullRequestJournal` | PR 有新评论 |
| `PullRequestMerged` | PR 已合并 |

</details>

#### 2.2 优先级排序

| 优先级 | 判定 |
|--------|------|
| **P0 - 立即** | 含 `Atme` 的消息（IssueAtme, PullReuqestAtme） |
| **P1 - 今天** | 自己管理的仓库有 PR 合并/关闭，或被分配的 Issue/PR 有更新 |
| **P2 - 本周** | 关注的仓库有新 PR、新 Issue |
| **P3 - 可忽略** | 点赞（ProjectPraised）、成员加入/离开、Fork 等系统通知 |

### Step 3：生成通知摘要

按下方输出模板生成报告。

### Step 4：标记已读（可选，需确认）

```bash
# 标记单条已读
gitlink-cli api POST "users/{owner}/messages/{id}/read" --format json

# 批量标记已读 — 逐条调用，GitLink 暂无批量已读 API
for id in <id1> <id2> <id3>; do
  gitlink-cli api POST "users/{owner}/messages/$id/read" --format json
done
```

> ⚠️ **执行前必须确认用户意图** — 标记已读为写操作。
> ⚠️ **GitLink 没有批量已读 API**，需要逐条标记。

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

## 七、近期已读回顾

> 列出最近 3-5 条已读但值得回顾的通知（如角色变更、PR 合并等）。

---

## 操作建议

- 建议标记已读：{{suggest_read_count}} 条 P3 通知
- 需要回复/处理：{{need_action_count}} 条 P0/P1 通知

如需标记 P3 通知为已读，我可以逐条执行：
`gitlink-cli api POST "users/{owner}/messages/{id}/read"`
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 无未读通知 | 输出"🎉 所有通知已处理完毕" |
| 通知数量 > 50 | 分页获取（page 1/2/3），优先分析最近 50 条 |
| API 返回 HTML 而非 JSON | 路径可能以 `/` 开头导致解析错误，去掉前导 `/` 重试 |
| `unread_notification` > messages 数组长度 | 存在多页数据，追加 `--query "page=2"` 获取 |
| 用户名不确定 | 先执行 `gitlink-cli auth status` 获取当前登录用户 |

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **标记已读为写操作**，执行前必须确认用户意图
- ✅ **本 Skill 默认只读分析**，仅在用户明确要求时标记已读
- ⚠️ **`gitlink-cli api` 路径不要以 `/` 开头**（CLI Bug）
- ⚠️ **GitLink 用「消息（messages）」而非「通知（notifications）」**
- ⚠️ **`source` 字段 `PullReuqestAtme` 是官方拼写错误**，实际使用注意匹配
- ⚠️ **通知可能分页**，数量 >20 时需追加 `--query "page=2"`
