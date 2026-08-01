# 飞书 Agent 第二轮答复核验与实现补充

日期：2026-08-01

输入：`FEISHU_AGENT_PLATFORM_FOLLOWUP_20260801.md` 的 Q1–Q18 答复

## 结论

第二轮答复可作为产品讨论材料，但不能整体当作飞书 OpenAPI 合同。经官方文档复核，Task v2、
消息权限、卡片更新、Base 事件和事件重试等关键处存在错误或把工程建议写成官方结论。

本轮仅把已经有明确官方合同、且与现有架构一致的部分落入代码：

- Task v2 创建加入 `client_token`，用于飞书保留期内的重复请求保护；
- Review WorkItem 的飞书 `open_id` 映射为 Task `assignee`；
- `YYYY-MM-DD` 截止日期映射为字符串毫秒时间戳和 `is_all_day=true`；
- 创建请求显式使用 `user_id_type=open_id`；
- 继续保存 Task 返回的 GUID，承担超过 `client_token` 有效期后的长期资源身份。

交互卡片、Base 配置同步、资源 owner 转移和多实例拓扑本轮只校正文档边界，不把未完成的
平台验收写成已实现。

## 已纠正的关键结论

| 主题 | 第二轮答复的问题 | 核验后的工程结论 |
|---|---|---|
| 群聊 @ 消息 | 把 `im.message.receive_v1` 与权限完全割裂，并写成接收无需 Scope | 事件订阅和权限是两层门禁；接收群内 @ 机器人消息仍需对应消息事件权限，例如 `im:message.group_at_msg:readonly` |
| 回复消息 | 只列 `im:message` | 回复接口可由其文档列出的任一允许权限授权，包括 `im:message` 或 `im:message:send_as_bot`；生产应选最小集合 |
| 卡片 ACK | 示例在 handler 返回前同步更新 loading 卡片 | 先快速返回成功 ACK；延时更新必须发生在 ACK 之后，不能在 ACK 前或并发执行 |
| 卡片长期维护 | 把回调 token 更新描述为所有已发送卡片的唯一更新方式 | 回调 token 用于短期延时更新；机器人自己发送的长期卡片应保存 `message_id`，使用消息 PATCH 接口更新 |
| Base 变更 | 只描述自动化 Webhook，并把能力写成确定事实 | 飞书存在 `drive.file.bitable_record_changed_v1` 记录变更事件；自动化的 operator、删除触发、配额等仍需单独实测，不能互相替代 |
| Base 幂等 | 在确认没有唯一约束后仍建议捕获 “已存在” | 没有唯一约束就不能依赖创建时的冲突错误；必须由 SQLite 单写者/分布式锁、本地 `record_id` 映射和对账保护竞态 |
| Task 幂等 | 声称 Task v2 不支持业务幂等键 | 创建 Task 明确支持 body 字段 `client_token`；成功后只保留 5 分钟，不能代替 Task GUID |
| Task 成员 | 声称负责人必须创建后再 add，关注人另有 followers 接口 | 创建请求的 `members` 可同时带 `assignee` 与 `follower` |
| Task 截止时间 | 写成 `yyyy-MM-dd HH:mm:ss` | Task v2 的时间字段是字符串形式的 Unix 毫秒时间戳；全天日期还需 `is_all_day=true` |
| Base 人员字段 | 声称写入始终只能使用 `open_id` | 人员字段接受与请求 `user_id_type` 一致的 `open_id`、`user_id` 或 `union_id`；当前消息链路持有 `open_id`，所以本项目显式使用 `open_id` |
| 事件重试 | 写成飞书最长重试约 24 小时 | 官方事件订阅说明列出的重试间隔为 15 秒、5 分钟、1 小时和 6 小时；本地 TTL 可更长，但不能把 24 小时写成官方事实 |

## Task v2 当前实现合同

输入模型新增：

```text
assignee_open_id
follower_open_ids[]
due_date = YYYY-MM-DD
```

发送模型：

```json
{
  "summary": "Review Gitlink/gitlink-cli PR #431",
  "description": "...",
  "client_token": "32-character-lowercase-hex",
  "members": [
    {"type": "user", "id": "ou_xxx", "role": "assignee"},
    {"type": "user", "id": "ou_yyy", "role": "follower"}
  ],
  "due": {
    "timestamp": "1785888000000",
    "is_all_day": true
  }
}
```

接口：

```text
POST /open-apis/task/v2/tasks?user_id_type=open_id
Scope: task:task:write
```

`client_token` 根据业务 key 与规范化请求内容生成。相同请求得到相同 token；请求正文变化时
token 也变化，避免以相同 token 发送不同参数的未定义行为。飞书成功幂等保留期只有 5 分钟，
因此长期同步仍遵循：

```text
pr_key -> 本地 review_collaboration_resources -> Task GUID -> 更新/完成/对账
```

## 卡片更新的准确边界

卡片按钮动作应采用：

```text
回调到达
-> 在预算内持久化 action、token、message_id 和去重键
-> handler 返回成功 ACK
-> 后台 Worker 执行 GitLink GET 或安全计划
-> ACK 之后使用回调 token 做延时更新
```

短期延时更新与长期固定卡片是两条路径：

| 场景 | 标识 | 接口 | 边界 |
|---|---|---|---|
| 本次按钮回调的延时结果 | callback token | `POST /interactive/v1/card/update` | token 30 分钟有效，最多更新 2 次，且必须在 ACK 后调用 |
| 长期维护机器人已发送卡片 | message_id | `PATCH /im/v1/messages/:message_id` | 保存消息 ID；适合刷新 PR 状态、归档和重启后继续更新 |

当前仓库已有卡片动作入口、Job 和 WorkItem 卡片模型，但上述真实发送/延时更新链路尚未完成
平台验收。因此状态仍是“可实现、待接入”，不是“已完成”。

## Base 配置与同步边界

Base 可作为管理员界面，SQLite 继续作为运行时真源：

```text
Base 编辑或记录变更事件
-> Gateway 验证来源与操作人权限
-> 校验 repository 是否在应用级 allowlist
-> 生成待确认变更或执行受控同步
-> SQLite 原子更新并写审计
```

`pr_key` 只能称为业务唯一标识，不能称为 Base 唯一索引。多实例下应由单写者或共享锁串行化
“查询—创建”；远端写入结果不确定时禁止自动再次创建。`drive.file.bitable_record_changed_v1`
证明记录变更事件存在，但其字段、删除语义、顺序、重复和真实权限仍要用专用测试 Base 留证。

## 仍待平台验证，不能据答复直接实现

1. Card 2.0 的准确 JSON、Markdown、组件和频率限制；
2. Base 自动化触发延迟、配额、失败重试、删除事件和 `operator_id` 的真实合同；
3. 群主与群管理员列表的准确接口、Scope 和群主变更事件；
4. 应用身份创建 Base/Doc 后的 owner 归属、协作者权限与转移 API；
5. 文档写入的长期幂等与远端成功、本地失败对账；
6. 多实例长连接的官方连接上限和推荐拓扑；
7. Task 全天日期在测试租户客户端中的跨时区展示；
8. Base、Doc、Task 的真实首次写入、重复执行、更新、删除后恢复。

## 下一轮真实验收顺序

1. 在测试租户创建一条带负责人和全天截止日期的 Task，保存 GUID 和脱敏请求证据；
2. 用同一 `client_token`、相同请求在 5 分钟内重复调用，确认不产生第二条任务；
3. 修改内容后使用新 token，确认本地 GUID 映射仍阻止重复主任务；
4. 创建一条 Review Base record，重复同步并验证没有第二条业务记录；
5. 创建一份 Doc，重复同步并验证更新/追加策略；
6. 发送真实 Review 卡片，完成“点击—ACK—异步结果—长期 message_id 更新”验收。

所有证据只保存哈希化会话/资源标识、错误码、request_id、耗时和应用版本，不保存 Token、
Secret、消息全文或用户绑定码。

## 官方依据

- [接收消息事件](https://open.feishu.cn/document/server-docs/im-v1/message/events/receive?lang=zh-CN)
- [回复消息](https://open.feishu.cn/document/server-docs/im-v1/message/reply?lang=zh-CN)
- [卡片交互与延时更新](https://open.feishu.cn/document/common-capabilities/message-card/add-card-interaction/interaction-module)
- [消息与卡片更新能力概览](https://open.feishu.cn/document/server-docs/im-v1/introduction?lang=zh-CN)
- [Task v2 概述、成员、时间与幂等](https://open.feishu.cn/document/task-v2/overview)
- [创建 Base 记录与人员字段](https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/create?lang=zh-CN)
- [Base 记录变更事件列表](https://open.feishu.cn/document/ukTMukTMukTM/uYDNxYjL2QTM24iN0EjN/event-list)
- [事件订阅重试说明](https://open.feishu.cn/document/server-docs/event-subscription-guide/overview?lang=zh-CN)
- [云文档权限概述](https://open.feishu.cn/document/server-docs/docs/permission/overview)
