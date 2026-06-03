# GitLink API 参考

## 认证端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/accounts/login` | POST | 用户名密码登录 |
| `/users/me` | GET | 获取当前用户信息 |

## 全局参数

| 参数 | 位置 | 说明 |
|------|------|------|
| `access_token` | Query | OAuth2 Token（自动注入） |
| `page` | Query | 分页页码（默认 1） |
| `limit` | Query | 每页条数（默认 20） |

## 响应格式

**成功响应**:
```json
{
  "ok": true,
  "data": { ... },
  "meta": { "page": 1, "limit": 20, "total_count": 100 }
}
```

**错误响应**:
```json
{
  "ok": false,
  "error": {
    "code": 401,
    "message": "无效token",
    "suggestion": "请先运行 gitlink-cli auth login 登录"
  }
}
```

## 常见错误码

| 错误码 | 含义 | 解决方案 |
|--------|------|----------|
| 401 | 未认证 | 运行 `gitlink-cli auth login` |
| 403 | 权限不足 | 确认账户权限或联系项目管理员 |
| 404 | 资源不存在 | 检查 owner/repo/id 是否正确 |
| 422 | 参数校验失败 | 检查请求参数 |
| -1 | GitLink 业务错误 | 查看 message 字段获取详情 |

## API 特殊性

### 必需字段

| 操作 | 必需字段 | 说明 |
|------|----------|------|
| Issue 创建 | `done_ratio: 0` | 数据库约束 |
| Issue 更新 | 当前 `subject` 和 `description` | 即使只改状态也应保留，避免清空描述 |
| Release 查看 | `version_id` | 不能用 tag_name |

### 端点前缀

| 操作 | 前缀 | 示例 |
|------|------|------|
| 分支操作 | `/v1/` | `/v1/:owner/:repo/branches` |
| Issue 评论 | 无 | `/issues/:id/journals` |
| 仓库操作 | 无 | `/:owner/:repo/info` |

### 已知 Bug

| Bug | 影响 | 状态 |
|-----|------|------|
| Branch 删除返回"不存在" | 无法删除分支 | 待 GitLink 修复 |
| Release 删除返回"不存在" | 无法删除发布 | 待 GitLink 修复 |
| Create File 返回"已存在" | 无法通过 API 创建文件 | 待 GitLink 修复 |
| `api` 命令路径以 `/` 开头会被解析为本地路径 | 返回 HTML 而非 JSON | 去掉路径前导 `/` 即可 |

---

## 消息（通知）API

GitLink 的通知功能通过「消息」API 实现。

### 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/users/{owner}/messages.json` | GET | 获取用户消息列表 |
| `/users/{owner}/messages/{id}/read` | POST | 标记单条消息已读 |
| `/users/{owner}/messages/{id}` | DELETE | 删除消息 |
| `/users/{owner}/messages/settings` | GET | 平台消息设置 |
| `/users/{owner}/messages/settings/list` | GET | 用户消息设置列表 |
| `/users/{owner}/messages/settings/update` | POST | 更新用户消息设置 |

### 查询参数（GET messages.json）

| 参数 | 类型 | 说明 |
|------|------|------|
| `status` | integer | 1=未读，2=已读，不传=全部 |
| `type` | string | `notification`=系统消息，`atme`=@我消息，不传=全部 |
| `page` | integer | 页码（默认 1） |
| `limit` | integer | 每页条数（默认 20） |

### 响应字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_count` | integer | 总消息数 |
| `unread_notification` | integer | 未读系统消息数 |
| `unread_atme` | integer | 未读@我消息数 |
| `messages[].id` | integer | 消息唯一 ID |
| `messages[].status` | integer | 1=未读，2=已读 |
| `messages[].content` | string | HTML 格式的消息内容 |
| `messages[].source` | enum | 消息来源类型（见下方枚举表） |
| `messages[].notification_url` | string | 消息跳转链接 |
| `messages[].created_at` | string | 创建时间（YYYY-MM-DD HH:mm:ss） |
| `messages[].time_ago` | string | 相对时间描述 |
| `messages[].type` | string | `notification` 或 `atme` |
| `messages[].sender` | object | 发送者信息（id, name, login, image_url） |

### source 枚举值

| 枚举值 | 含义 | 分类 |
|--------|------|------|
| `IssueAssigned` | 有新指派给我的疑修 | Issue |
| `IssueExpire` | 疑修截止日期到达最后一天 | Issue |
| `IssueAtme` | 在疑修中@我 | @提及 |
| `IssueChanged` | 疑修状态变更 | Issue |
| `IssueDeleted` | 疑修被删除 | Issue |
| `IssueJournal` | 疑修有新评论 | Issue |
| `ProjectIssue` | 项目新 Issue | Issue |
| `PullRequestAssigned` | 有新指派给我的 PR | PR |
| `PullReuqestAtme` | 在 PR 中@我（**官方 API 拼写如此**） | @提及 |
| `PullRequestChanged` | PR 状态变更 | PR |
| `PullRequestClosed` | PR 被关闭 | PR |
| `PullRequestJournal` | PR 有新评论 | PR |
| `PullRequestMerged` | PR 已合并 | PR |
| `ProjectPullRequest` | 项目有新 PR | PR |
| `ProjectJoined` | 加入项目 | 系统 |
| `ProjectLeft` | 离开项目 | 系统 |
| `ProjectMemberJoined` | 新成员加入项目 | 系统 |
| `ProjectMemberLeft` | 成员离开项目 | 系统 |
| `ProjectForked` | 项目被 Fork | 系统 |
| `ProjectPraised` | 项目被点赞 | 系统 |
| `ProjectRole` | 项目角色变更 | 系统 |
| `ProjectFollowed` | 项目被关注 | 系统 |
| `ProjectDeleted` | 项目被删除 | 系统 |
| `ProjectTransfer` | 项目转让 | 系统 |
| `ProjectSettingChanged` | 项目设置变更 | 系统 |
| `ProjectMilestone` | 新里程碑 | 系统 |
| `ProjectMilestoneCompleted` | 里程碑完成 | 系统 |
| `ProjectVersion` | 新版本发布 | 系统 |
| `ProjectOpenDevOps` | DevOps 引擎开通 | 系统 |
| `OrganizationJoined` | 加入组织 | 系统 |
| `OrganizationLeft` | 离开组织 | 系统 |
| `OrganizationRole` | 组织角色变更 | 系统 |
| `LoginIpTip` | 登录 IP 提示 | 其他 |

### 调用示例

```bash
# 获取未读通知
gitlink-cli api GET "users/lindiwen23/messages.json" --query "status=1&limit=20" --format json

# 获取 @我 的通知
gitlink-cli api GET "users/lindiwen23/messages.json" --query "type=atme&status=1" --format json

# 标记单条已读
gitlink-cli api POST "users/lindiwen23/messages/740214/read" --format json

# ⚠️ 路径不要以 / 开头，否则会被解析为本地文件路径
```
