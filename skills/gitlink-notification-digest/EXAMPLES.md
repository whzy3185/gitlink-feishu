# gitlink-notification-digest 使用样例

## 样例 1：手动执行通知摘要

**日期**：2026-06-03
**用户**：lindiwen23
**CLI 版本**：支持 `notification` shortcut 的 gitlink-cli

### 执行流程

```bash
# Step 2: 获取未读通知（status=1）
gitlink-cli notification +list --status unread --limit 20 --format json
# → 7 条未读，unread_notification=7, unread_atme=0

# Step 3: 获取已读通知（用于趋势分析和回顾）
gitlink-cli notification +list --status read --limit 20 --format json
# → 21 条已读

# Step 4: 分类统计、生成摘要报告
```

### 关键发现

| 项目 | 值 |
|------|-----|
| 未读通知 | 7 条 |
| @我未读 | 0 条 |
| 总通知 | 28 条（7 未读 + 21 已读） |
| 推荐命令 | `gitlink-cli notification +list` |
| 标记已读 | `gitlink-cli notification +read --ids ... --dry-run/--yes` |

### 原始 API 返回（未读 7 条）

```json
{
  "total_count": 7,
  "type": "",
  "unread_notification": 7,
  "unread_atme": 0,
  "messages": [
    {
      "id": 740214, "status": 1,
      "content": "jiangtx在 <b>jiangtx/gitlink-cli</b> 提交了一个合并请求：<b>label 模块新建</b>",
      "notification_url": "https://www.gitlink.org.cn/jiangtx/gitlink-cli/pulls/15347",
      "source": "ProjectPullRequest",
      "created_at": "2026-06-03 00:27:37", "time_ago": "10小时前",
      "type": "notification"
    },
    {
      "id": 740213, "status": 1,
      "content": "jiangtx在 <b>jiangtx/gitlink-cli</b> 提交了一个合并请求：<b>pr 域补全</b>",
      "notification_url": "https://www.gitlink.org.cn/jiangtx/gitlink-cli/pulls/15346",
      "source": "ProjectPullRequest",
      "created_at": "2026-06-03 00:11:48", "time_ago": "10小时前",
      "type": "notification"
    },
    {
      "id": 740178, "status": 1,
      "content": "jiangtx在 <b>jiangtx/gitlink-cli</b> 提交了一个合并请求：<b>repo 域补全</b>",
      "notification_url": "https://www.gitlink.org.cn/jiangtx/gitlink-cli/pulls/15343",
      "source": "ProjectPullRequest",
      "created_at": "2026-06-02 23:29:52", "time_ago": "11小时前",
      "type": "notification"
    },
    {
      "id": 740076, "status": 1,
      "content": "jiangtx在 <b>jiangtx/gitlink-cli</b> 提交了一个合并请求：<b>基础设施修复</b>",
      "notification_url": "https://www.gitlink.org.cn/jiangtx/gitlink-cli/pulls/15336",
      "source": "ProjectPullRequest",
      "created_at": "2026-06-02 16:56:47", "time_ago": "17小时前",
      "type": "notification"
    },
    {
      "id": 740002, "status": 1,
      "content": "<b>CWQ</b> 点赞了你管理的仓库 <b>CWQ/Aether_Lens_System-v0.0.1</b>",
      "notification_url": "https://www.gitlink.org.cn/caoweiqiong",
      "source": "ProjectPraised",
      "created_at": "2026-06-02 15:12:55", "time_ago": "19小时前",
      "type": "notification"
    },
    {
      "id": 738181, "status": 1,
      "content": "<b>Somebird</b> 已加入项目 <b>CWQ/Aether_Lens_System-v0.0.1</b>",
      "notification_url": "https://www.gitlink.org.cn/caoweiqiong/Aether",
      "source": "ProjectMemberJoined",
      "created_at": "2026-06-01 22:22:07", "time_ago": "1天前",
      "type": "notification"
    },
    {
      "id": 738136, "status": 1,
      "content": "<b>Somebird</b> 点赞了你管理的仓库 <b>CWQ/Aether_Lens_System-v0.0.1</b>",
      "notification_url": "https://www.gitlink.org.cn/Somebird",
      "source": "ProjectPraised",
      "created_at": "2026-06-01 20:18:28", "time_ago": "2天前",
      "type": "notification"
    }
  ]
}
```

### 分类处理

按 `source` 字段分类：

| source | 含义 | 数量 | 优先级 |
|--------|------|------|--------|
| `ProjectPullRequest` | 项目新 PR（jiangtx/gitlink-cli） | 4 | P2 |
| `ProjectPraised` | 项目被点赞（CWQ/Aether_Lens_System） | 2 | P3 |
| `ProjectMemberJoined` | 新成员加入 | 1 | P3 |

### 生成的报告

```markdown
# 🔔 通知摘要

> 生成时间：2026-06-03 10:18
> 未读通知：7 条 / 总计：28 条

---

## 一、概要

| 类型 | 未读 | 总计 |
|------|------|------|
| 🔴 @提及 | 0 | 0 |
| 🟡 Issue 更新 | 0 | 0 |
| 🟢 PR 更新 | 4 | ~8 |
| 🔵 系统通知 | 3 | ~19 |

---

## 二、需要立即处理（P0）

🎉 无紧急通知。

## 三、今天处理（P1）

无待处理通知。

## 四、本周关注（P2）

| # | 类型 | 仓库 | 内容摘要 | 时间 |
|---|------|------|----------|------|
| 1 | 🟢 PR | jiangtx/gitlink-cli | label 模块新建 (#15347) | 6/3 00:27 |
| 2 | 🟢 PR | jiangtx/gitlink-cli | pr 域补全 (#15346) | 6/3 00:11 |
| 3 | 🟢 PR | jiangtx/gitlink-cli | repo 域补全 (#15343) | 6/2 23:29 |
| 4 | 🟢 PR | jiangtx/gitlink-cli | 基础设施修复 (#15336) | 6/2 16:56 |

## 五、可忽略（P3）

| # | 类型 | 仓库 | 内容摘要 | 时间 |
|---|------|------|----------|------|
| 1 | 🔵 点赞 | CWQ/Aether_Lens_System-v0.0.1 | CWQ 点赞了仓库 | 6/2 15:12 |
| 2 | 🔵 成员 | CWQ/Aether_Lens_System-v0.0.1 | Somebird 加入项目 | 6/1 22:22 |
| 3 | 🔵 点赞 | CWQ/Aether_Lens_System-v0.0.1 | Somebird 点赞了仓库 | 6/1 20:18 |

## 六、通知趋势

| 时间段 | 通知数 |
|--------|--------|
| 今日（6/3） | 2 |
| 昨日（6/2） | 3 |
| 本周（6/1-6/3） | 8 |

## 操作建议

- 建议标记已读：3 条 P3 通知
- 需要回复/处理：0 条 P0/P1 通知
```

### 经验总结

1. **优先使用 `notification` shortcut**：列表、标记已读、删除和发送 @ 消息均已有封装
2. **API 端点是 `messages` 不是 `notifications`**：GitLink 用「消息」术语，shortcut 已屏蔽路径细节
3. **响应字段 `unread_notification` 和 `unread_atme`**：顶层统计字段可直接用于分类计数，无需遍历全部消息
4. **标记已读是写操作**：必须先 `--dry-run`，用户确认后再 `--yes`
5. **`source` 字段 `PullReuqestAtme`**：官方 API 存在拼写错误（应为 PullRequestAtme），匹配时注意

---

## 样例 2：通过 Agent 调用 Skill（自动摘要）

**日期**：2026-06-03
**调用方式**：`Agent(subagent_type="general-purpose", prompt="请调用 gitlink-notification-digest skill，帮我整理通知。")`

### Agent 自主执行的命令序列

```
工具调用 1: Read  → ../gitlink-shared/SKILL.md              ← 遵循 Skill 前置条件
工具调用 2: Bash  → gitlink-cli notification +list --status unread --limit 20 --format json
工具调用 3: Bash  → gitlink-cli notification +list --status read --limit 20 --format json
工具调用 4: Bash  → gitlink-cli notification +list --limit 20 --format json
```

### Agent 决策过程

Agent **正确遵循了 skill v3.0.0 的工作流**：

1. 先读取 `gitlink-shared/SKILL.md` 了解认证和全局参数
2. 使用 `notification +list` 获取未读、已读、全部三类数据
4. 按 `source` 字段分类：`ProjectPullRequest` → P2，`ProjectPraised`/`ProjectMemberJoined` → P3
5. 按输出模板生成结构化报告，含所有七个章节
6. 主动询问是否需要标记 P3 通知为已读

共消耗 **35,601 tokens**，**6 次工具调用**，耗时 **51.7 秒**。

### Agent 生成的报告

```markdown
# 通知摘要

> 生成时间：2026-06-03 10:30
> 未读通知：7 条 / 总计：28 条（含已读 21 条）
> @我未读：0 条

## 一、概要

| 类型 | 未读 | 总计 |
|------|------|------|
| @提及 | 0 | 0 |
| Issue 更新 | 0 | 0 |
| PR 更新 | 4 | 7 |
| 系统通知 | 3 | 21 |

## 二、需要立即处理（P0）
无紧急通知。

## 三、今天处理（P1）
无待处理通知。

## 四、本周关注（P2）
4 条 jiangtx/gitlink-cli 的 PR 需关注

## 五、可忽略（P3）
3 条 CWQ/Aether_Lens_System-v0.0.1 的点赞和成员通知

## 六、通知趋势
| 时间段 | 通知数 |
|--------|--------|
| 今日 | 2 |
| 昨日 | 3 |
| 本周 | 7 |

## 七、近期已读回顾
| 类型 | 内容 | 时间 |
|------|------|------|
| 加入项目 | 加入 jiangtx/gitlink-cli | 06-01 |
| 成员加入 | wyxttn 加入 yetja/灵枢 | 05-29 |
| PR 合并 | 帮助中心 PR 已通过 | 05-13 |
| 角色变更 | 帮助中心角色改为管理员 | 05-13 |

## 操作建议
- 建议标记已读：3 条 P3 通知
- 需要关注：4 条 P2 通知
```

### 验证结论

✅ skill v3.0.0 验证通过：
- Agent 正确使用了 `gitlink-cli notification +list` 获取消息列表
- Agent 正确使用了 `gitlink-cli notification +read --dry-run` 预览标记已读操作
- Agent 按 `source` 枚举值正确分类，识别出 `PullReuqestAtme` 拼写异常
- Agent 正确区分了 P0/P1/P2/P3 优先级
- Agent 使用 `unread_notification`/`unread_atme` 顶层字段快速统计
- Agent 生成了趋势章节和已读回顾章节
- 报告结构完整，七个章节覆盖全部模板要求

---

## 异常场景速查

| 场景 | 检测方式 | 处理 |
|------|----------|------|
| `notification +list` 失败 | 查看错误信息 | 先确认已登录，再运行 `gitlink-cli notification +list --format json` |
| 未读通知 > 返回条数 | `total_count` > `messages.length` | 追加 `--page 2` |
| 用户名不确定 | shortcut 自动解析当前用户失败 | 先执行 `gitlink-cli auth status` |
| 无未读通知 | `unread_notification == 0` | 输出 "🎉 所有通知已处理完毕" |

---

## 版本兼容性说明

本 skill v3.0.0 基于新增的 `notification` shortcut 编写。关键变更：

| 版本 | `notification` 子命令 | 实际 API | 标记已读 |
|------|----------------------|----------|----------|
| v1.0.0 | `notification +list`（虚构） | 不存在 | `notification +read-all`（虚构） |
| v2.0.0 | 无此子命令 | `GET /api/users/{owner}/messages.json` | `POST /api/users/{owner}/messages/{id}/read` |
| v3.0.0 | `notification +list` | 由 shortcut 封装 messages API | `notification +read --ids ... --dry-run/--yes` |

当 CLI 版本更新后，重新验证可用命令：
```bash
gitlink-cli --help
```
