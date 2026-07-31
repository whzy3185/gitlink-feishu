# GitLink Review 机器人飞书 Agent 排障交接

日期：2026-07-31

用途：

```text
把本文原样提供给飞书开放平台 Agent 或飞书技术支持，
请求协助确认长连接消息事件为什么没有进入 Go 业务 handler。
```

本文已脱敏，不包含 App Secret、access token、Webhook Secret、真实 chat_id 或
真实 open_id。

## 1. 当前结论

GitLink Review 机器人代码已经更新，飞书自建应用也可以：

- 获取 `tenant_access_token`；
- 获取 bot identity；
- 建立 Channel SDK WebSocket 长连接；
- 读取测试群历史消息；
- 以应用身份向测试群主动发消息。

但是测试群真实用户在机器人在线期间发送：

```text
@gitlink 查看 PR #431
```

之后：

```text
Channel 日志没有出现任何入站消息
本地 SQLite 没有产生 event 或 Job
飞书群没有收到回执
飞书群没有收到最终结果
```

同时发现一个确定的本地配置问题：

```text
最新消息发送者的 open_id
不在绑定文件 admin_user_ids
也不在 allowed_user_ids
```

因此即使事件到达 Gateway，也会被 `sender_not_allowed` 拒绝。但当前连拒绝 receipt
都未出现在监听输出中，说明还存在更前置的事件投递、dispatcher 或 PolicyGate
问题。

## 2. 项目定位

机器人服务于 GitLink 仓库拥有者在短时间内处理大量 PR 的场景：

```text
飞书负责群沟通、分工、进度和后续协作资源
gitlink-cli 负责稳定的工具和执行面
GitLink 负责 PR、Review、线程和合并的正式事实
Agent 负责取证、分析和草拟
拥有者团队保留最终 Review 和合并决定
```

当前阶段严格保持：

```text
mode = preview
GitLink GET-only
mutates_gitlink = false
```

机器人不会从飞书执行批准、拒绝、评论、Reviewer 或合并写入。

## 3. 仓库和版本

仓库：

```text
https://github.com/whzy3185/gitlink-feishu
```

分支：

```text
feat/round2-review-collaboration-p21-live-reply
```

关键提交：

```text
fa3c1fb53dcb9f2fb0eeb9b558a32780f45ceeff
feat: add reliable Feishu live review replies

61c44fdb947818fca69be54a4d60b583391b688d
docs: summarize Feishu review bot capabilities
```

Go SDK：

```text
github.com/larksuite/oapi-sdk-go/v3 v3.9.9
```

运行系统：

```text
Windows
PowerShell
Go 1.26.1
飞书企业自建应用
```

## 4. 实现形式

### 4.1 入口命令

```text
gitlink-cli feishu +review-gateway
```

在线启动：

```powershell
gitlink-cli feishu +review-gateway `
  --listen `
  --bindings .local/review-gateway-bindings.json `
  --state-db .local/review-gateway.db `
  --format json
```

群发现：

```powershell
gitlink-cli feishu +review-gateway `
  --discover-chats `
  --format json
```

### 4.2 关键源码

```text
shortcuts/feishu/review_gateway_command.go
  Channel SDK、WebSocket、dispatcher、消息与卡片入口、handler 预算

shortcuts/feishu/review_gateway_models.go
  事件、群绑定、命令解析、权限判断、Job 模型

shortcuts/feishu/review_gateway_store.go
  SQLite 去重、Job、租约、重试、结果和回复状态

shortcuts/feishu/review_gateway_executor.go
  GitLink GET-only Review Queue / Context 执行器

shortcuts/feishu/review_gateway_reply.go
  飞书回执和最终消息回复
```

## 5. 架构

```mermaid
flowchart LR
    U["飞书群真实用户<br/>@gitlink"] --> L["飞书长连接"]
    L --> D["EventDispatcher<br/>P2MessageReceiveV1"]
    D --> N["normalize.ParseMessage"]
    N --> I["bot identity 与 mention 标记"]
    I --> P["PolicyGate"]
    P --> G["ReviewGateway<br/>群绑定与用户 allowlist"]
    G --> S["SQLite 原子事件与 Job"]
    S --> E["GET-only Executor"]
    E --> R["Channel.Send<br/>回复原消息"]
```

完整链路：

```text
飞书消息
-> WebSocket
-> P2MessageReceiveV1
-> NormalizedMessage
-> bot mention 标记
-> GroupAllowlist / RequireMention / DM policy
-> chat binding / sender allowlist
-> message_id 去重
-> SQLite Job
-> GitLink GET-only
-> result_json
-> ReplyMessageID
```

## 6. Channel SDK 初始化方式

实现逻辑等价于：

```go
eventDispatcher := dispatcher.NewEventDispatcher("", "")

apiClient := lark.NewClient(appID, appSecret)

wsClient := larkws.NewClient(
    appID,
    appSecret,
    larkws.WithEventHandler(eventDispatcher),
)

policy := larktypes.PolicyConfig{
    GroupAllowlist:      enabledBoundChatIDs,
    RequireMention:      true,
    RespondToMentionAll: false,
    DMMode:              "disabled",
}

channel := larkchannel.NewChannel(
    apiClient,
    wsClient,
    larktypes.WithPolicyConfig(policy),
)
```

当前成功日志包括：

```text
event-dispatch is ready
bot/v3/info status=200
bot identity resolved
WebSocket 连接成功
feishu.review-gateway/v1 ready
```

无 Channel error 日志。

## 7. 为什么没有直接使用 Channel.OnMessage

检查 SDK v3.9.9 源码后发现，普通 `Channel.OnMessage` 进入内部 batching pipeline，
业务 handler 的 error 不会继续返回给飞书 event dispatcher。

本项目要求：

```text
SQLite 持久化失败
-> handler 必须返回 error
-> 飞书保留重投机会
```

因此当前实现仍使用官方 SDK 的连接、normalizer、bot identity、PolicyGate 和 Send，
但普通消息直接注册在同一个官方 dispatcher：

```go
dispatcher.OnP2MessageReceiveV1(func(
    ctx context.Context,
    event *larkim.P2MessageReceiveV1,
) error {
    message := normalize.ParseMessage(event)
    bot := channel.GetBotIdentity(ctx)

    // 根据 bot open_id / user_id 标记 MentionedBot 和 mention.IsBot
    // 然后执行 PolicyGate

    if !policy.Evaluate(message).Allowed {
        return nil
    }
    return persistenceHandler(ctx, message)
})
```

卡片动作仍使用：

```go
channel.OnCardAction(...)
```

需要飞书 Agent 重点确认：

> 在 Go Channel SDK v3.9.9 中，将 `OnP2MessageReceiveV1` 直接注册到传给
> `ws.WithEventHandler` 的 dispatcher，是否是受支持的用法？Channel 启动过程
> 是否可能覆盖或绕开该 handler？如果需要业务 error 传播，官方推荐实现是什么？

## 8. PolicyGate 和业务权限

### 8.1 SDK PolicyGate

当前策略：

```text
GroupAllowlist = 仅 enabled 绑定群
RequireMention = true
RespondToMentionAll = false
DMMode = disabled
```

收到消息后先获取 bot identity，再遍历 `NormalizedMessage.Mentions`。当 mention 的
OpenID 或 UserID 与 bot identity 匹配时：

```go
message.MentionedBot = true
mention.IsBot = true
```

然后执行：

```go
policy.Evaluate(message)
```

当前实现对 PolicyGate 拒绝直接 `return nil`，没有记录拒绝原因。这会使以下情况
在终端和飞书端都表现为“没有反应”：

- group not allowed；
- no mention；
- mention all blocked；
- DM disabled。

### 8.2 ReviewGateway 权限

通过 PolicyGate 后，业务层继续验证：

```text
chat_id 是否已绑定
binding 是否 enabled
sender open_id 是否在 allowed_user_ids
sender 是否在 admin_user_ids
命令是否受支持
事件是否过期
```

当前已确认：

```text
最新真实发送者不在 admin_user_ids
最新真实发送者不在 allowed_user_ids
```

因此本地绑定必须更新后重启，或测试期间显式采用安全的测试群成员策略。

当前代码只对 `Accepted=true` 的 Job 发送“已接收”回执。对
`sender_not_allowed`、`unsupported_read_only_command` 等拒绝结果，只输出本地
receipt，不回复飞书消息。这也是用户看到完全无反馈的一个产品缺陷。

## 9. 当前支持的命令

```text
帮助
查看绑定
查看待 Review
查看待审查
查看 PR #431
刷新 PR #431
生成 PR #431 Review 草稿
查看我的 Review 任务
领取 PR #431
绑定仓库 owner/repo
```

其中：

- Review Queue 和 PR Context 会执行 GitLink GET-only；
- Review 草稿只生成本地确定性模板；
- 领取只生成计划；
- 首次绑定必须由管理员本地预配置；
- 个人 Task 尚未真正接入飞书 Task。

## 10. Handler 和可靠队列

飞书入站预算：

```text
handler 总预算：默认 2000ms
SQLite 预算：默认 500ms
SQLite busy_timeout：400ms
耗时 GitLink 读取：异步
stdout：非阻塞
```

SQLite 保存：

```text
message_id / event_id dedupe
Job payload
attempt_count / max_attempts
next_attempt_at
lease_owner / lease_expires_at
result_json
handler_latency_ms
reply_status
reply_attempt_count
reply lease
reply_message_id
reply_error_summary
```

事件去重和 Job 保存处于同一 SQLite 事务。启动时可恢复 queued Job 和租约已过期的
running Job。

## 11. 出站回复

回执和最终结果通过 SDK：

```go
channel.Send(ctx, &types.SendInput{
    ChatID:         job.ChatID,
    ReplyMessageID: job.SourceMessageID,
    MsgType:        "text",
    Text:           result,
})
```

已确认：

```text
应用身份主动发送测试群消息成功
Feishu OpenAPI code=0
返回 message_id
测试群历史消息可读取
```

因此当前不是 App Secret、tenant token、机器人不在群内或完全没有消息发送权限的
问题。

最终回复 worker 尚未运行过，因为入站消息没有生成 Job。

## 12. 真实测试时间线

### 2026-07-30

```text
22:51 左右：旧监听器 WebSocket 连接成功
23:01 左右：真实用户两次发送“查看 PR #431”
之后发现监听器已退出
SQLite 无 Job
```

旧消息不会在机器人重新上线后由飞书自动补投。

### 2026-07-31

```text
12:29:40：最新版单实例监听器 WebSocket 连接成功
12:31:33：真实用户重新发送“查看 PR #431”
消息在正确测试群
消息有 message_id
正文含 @_user_N mention placeholder
发送时间晚于监听器 ready
测试用户 open_id 不匹配本地 admin / allowed allowlist
监听 stdout 无任何入站 receipt
监听 stderr 无错误
SQLite 无 Job
```

另外：

```text
feishu +app-check --remote
passed = 5
warned = 0
failed = 0
```

但该检查只证明：

- Webhook 配置存在；
- App ID / Secret 可用；
- tenant token 可获取。

它不证明：

- `im.message.receive_v1` 已在开发者后台订阅；
- 当前应用版本已发布并包含该订阅；
- 消息权限和应用可用范围已生效；
- 事件实际投递到当前 WebSocket；
- 没有其他实例分走事件。

## 13. 已排除和未排除

### 已排除

| 项目 | 结果 |
|---|---|
| 代码未更新 | 已更新到 P2.1 |
| App ID / Secret 完全无效 | 已排除 |
| tenant token 无法获取 | 已排除 |
| bot identity 无法获取 | 已排除 |
| WebSocket 无法连接 | 已排除 |
| 机器人无法向测试群发消息 | 已排除 |
| 测试消息没有进入群历史 | 已排除 |
| 用户没有真正 mention | 历史正文存在 mention placeholder |
| 消息发送时监听器离线 | 2026-07-31 新消息发送时在线 |
| Job 执行失败 | 尚未进入 Job 阶段 |
| GitLink GET 失败 | 尚未进入 GitLink 阶段 |

### 已确认问题

| 项目 | 结果 |
|---|---|
| 本地发送者 allowlist | 当前真实用户不匹配 |
| PolicyGate 拒绝可观察性 | 缺失，当前静默 |
| 业务拒绝的飞书反馈 | 缺失，仅 Accepted Job 回复 |
| 常驻部署 | 当前只是本机进程，不是正式服务 |

### 尚未排除

| 优先级 | 可能问题 |
|---|---|
| P0 | 开发者后台未订阅 `im.message.receive_v1` |
| P0 | 订阅或权限变更未随应用版本发布/生效 |
| P0 | Channel SDK 自定义 dispatcher 注册方式没有收到事件 |
| P1 | `normalize.ParseMessage` 的 mentions 与 bot identity 比较未命中 |
| P1 | PolicyGate 因 `MentionedBot=false` 静默拒绝 |
| P1 | 存在其他长连接实例，事件被随机投递到另一实例 |
| P1 | 应用可用范围或群事件权限与主动发消息权限不一致 |

## 14. 希望飞书 Agent 回答的问题

1. 如何在飞书开发者后台准确确认 `im.message.receive_v1` 已订阅、已随当前版本发布、
   并对当前测试群生效？
2. 是否有 OpenAPI、调试台或事件投递日志可以确认某个 `message_id` 被投递到哪条
   WebSocket 连接？
3. Go Channel SDK v3.9.9 中，`Channel.OnMessage` 是否有办法把业务 handler error
   返回 dispatcher，以触发飞书重试？
4. 直接在传给 `ws.WithEventHandler` 的 `EventDispatcher` 上注册
   `OnP2MessageReceiveV1` 是否受支持？
5. `normalize.ParseMessage` 对群聊 `@机器人` 的 `Mentions`、`OpenID`、`UserID` 和
   `MentionedBot` 的准确语义是什么？
6. `PolicyGate.RequireMention=true` 时，官方推荐如何注入 bot identity 和标记
   `MentionedBot`？
7. 一个应用存在多个 WebSocket 连接时，消息是否随机投递且不广播？如何列出或关闭
   其他连接？
8. 主动发群消息成功但接收群消息事件失败时，需要重点检查哪些权限、订阅、应用
   版本和可用范围？
9. 官方建议如何记录 PolicyGate 的拒绝原因和原始 event envelope，同时避免泄露
   消息内容和用户隐私？
10. 对 3 秒事件时限，先持久化后异步执行、持久化失败返回 error 的模式是否符合
    飞书重试语义？

## 15. 可直接复制给飞书 Agent 的提问

```text
我们正在使用飞书企业自建应用和 Go Channel SDK
github.com/larksuite/oapi-sdk-go/v3 v3.9.9，开发一个 GitLink PR Review
只读机器人。

当前能力：
1. tenant_access_token 获取成功；
2. bot/v3/info 获取成功；
3. Channel SDK WebSocket 显示连接成功；
4. 应用身份可向测试群发消息，OpenAPI code=0；
5. 可读取测试群历史消息；
6. 测试用户在连接 ready 后真实 @机器人发送“查看 PR #431”；
7. 历史消息有 message_id，正文有 @_user_N mention placeholder；
8. 但是 OnP2MessageReceiveV1 handler 没有任何日志，SQLite 没有 Job，群里没有回复。

实现方式：
- dispatcher.NewEventDispatcher("", "")
- ws.NewClient(appID, appSecret, ws.WithEventHandler(dispatcher))
- channel.NewChannel(apiClient, wsClient, WithPolicyConfig)
- GroupAllowlist 只包含测试群
- RequireMention=true
- DMMode=disabled
- 因 Channel.OnMessage batching 不传播业务 error，我们直接在同一个 dispatcher
  注册 OnP2MessageReceiveV1，然后调用 normalize.ParseMessage、GetBotIdentity、
  手动标记 MentionedBot、PolicyGate.Evaluate，再进入 SQLite handler。

另有一个确定问题：当前真实用户 open_id 不在本地业务 allowed_user_ids 中；但这应
发生在业务 Gateway 层。现在连 PolicyGate 后的 receipt 都没有，说明事件没有进入
业务 handler 或被前置策略静默过滤。

请帮助确认：
1. 开发者后台应如何确认 im.message.receive_v1 的订阅、权限、版本发布和可用范围；
2. 直接注册 dispatcher.OnP2MessageReceiveV1 是否是 Channel SDK 支持的方式；
3. RequireMention 下 NormalizedMessage.Mentions 和 MentionedBot 应如何正确处理；
4. 如何查看事件投递日志或确认事件是否被其他 WebSocket 连接分走；
5. 如何在保留 handler error 重试语义的同时使用 Channel SDK 推荐入口。
```

## 16. 本地下一步

在飞书侧回答前，本地建议按以下顺序处理：

1. 把当前真实测试用户加入 `allowed_user_ids`，但不自动赋予管理员权限；
2. 给 raw event、normalizer、mention matching、PolicyGate decision 增加脱敏诊断；
3. 对 `sender_not_allowed` 等安全拒绝回复明确提示，而不是静默；
4. 在开发者后台确认消息事件订阅、权限和版本发布；
5. 确认只有一个 WebSocket 实例；
6. 用一个新的 message_id 重测；
7. 通过后再验证 SQLite Job、GitLink GET-only 和两次回复；
8. 最后部署为常驻服务，而不是依赖临时终端进程。
