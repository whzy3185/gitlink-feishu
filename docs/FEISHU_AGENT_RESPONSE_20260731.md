# 飞书 Agent 回复与本地行动修订

日期：2026-07-31

来源：

```text
用户将 docs/FEISHU_AGENT_DEBUG_HANDOFF.md 提供给飞书 Agent 后返回的分析。
```

说明：

> 本文保存飞书 Agent 给出的技术判断和本地行动修订，便于协同排障。它不是飞书
> 官方工单结论；涉及控制台订阅、事件版本和权限的判断仍需在真实应用后台验证。

## 1. 飞书 Agent 的主要判断

### 1.1 直接注册 P2 消息 dispatcher 是支持方式

当前实现：

```go
eventHandler := dispatcher.NewEventDispatcher("", "").
    OnP2MessageReceiveV1(func(
        ctx context.Context,
        event *larkim.P2MessageReceiveV1,
    ) error {
        // business handler
        return nil
    })

client := larkws.NewClient(
    appID,
    appSecret,
    larkws.WithEventHandler(eventHandler),
)
```

与飞书 WebSocket SDK 的标准 P2 事件接入形式一致。因此：

```text
dispatcher.OnP2MessageReceiveV1
+ ws.WithEventHandler(dispatcher)
```

本身不应再被列为“非官方用法”。真正需要核对的是事件是否按 P2/v2.0 订阅并投递。

### 1.2 多 WebSocket 连接应提升为 P0

飞书长连接采用集群消费语义，同一应用的多个客户端不会同时收到同一事件。事件会
投递给其中一个连接，而不是广播。

因此如果存在：

- 旧终端；
- 旧 `go run`；
- 另一台机器；
- 调试进程；
- 常驻服务；

新消息可能被其他连接消费。本地排障必须先确认只保留一个实例。

本地已经执行：

```text
枚举所有 gitlink-cli feishu +review-gateway 进程
关闭诊断时遗留的重复实例
只保留一个 PowerShell -> go.exe -> gitlink-cli.exe 进程树
```

但开发者后台或其他机器是否仍有连接，需要飞书侧继续确认。

### 1.3 必须核对事件 v1.0 / v2.0

当前代码注册：

```go
OnP2MessageReceiveV1
```

它面向 P2/v2.0 事件 envelope。开发者后台必须确认“接收消息”
`im.message.receive_v1` 使用与代码匹配的事件版本。

如果后台配置为旧版事件而代码只注册 P2 handler，业务 handler 不会收到消息。

### 1.4 应用发布状态也是 P0

仅在开发者后台编辑：

- 事件订阅；
- 权限；
- 应用可用范围；

不等于测试环境已经使用这些变更。需要确认包含这些配置的应用版本已经发布并在
当前租户生效。

### 1.5 raw handler 必须有无条件命中日志

现有日志只能证明：

```text
dispatcher ready
bot identity ready
WebSocket connected
```

不能证明任何消息事件到达 dispatcher。

建议在 `OnP2MessageReceiveV1` 第一行记录脱敏事件元数据，再进行：

```text
normalize
mention matching
PolicyGate
ReviewGateway
```

至少记录：

- handler hit；
- event ID 是否存在；
- message ID 是否存在；
- chat ID 的不可逆摘要；
- sender ID 的不可逆摘要；
- mention 数量；
- bot mention 是否匹配；
- PolicyGate allowed / reject reason；
- Gateway receipt reason；
- handler latency。

不记录：

- App Secret；
- access token；
-完整消息正文；
- 真实 open_id；
- 真实 chat_id。

## 2. 对原排障问题的回答

### Q1：如何确认事件订阅和发布？

在开发者后台逐项核对：

```text
应用
-> 事件与回调
-> 事件配置
-> 接收消息 im.message.receive_v1
-> 事件版本与 OnP2MessageReceiveV1 匹配
-> 版本管理与发布
-> 当前版本已发布
```

同时核对权限管理和应用可用范围。

### Q2：能否查询 message_id 投递到哪条连接？

飞书 Agent 的判断是没有可直接按 `message_id` 查询 WebSocket 投递路径的接口。
推荐通过关闭所有旧连接、仅保留一个客户端、发送新的消息 ID 来排除。

### Q3：handler error 与重试

飞书 Agent 判断：

```text
dispatcher.OnP2MessageReceiveV1 返回 error
可以保留事件处理失败语义
```

而 `Channel.OnMessage` 的 batching handler 不适合承担本项目的持久化失败回传合同。

当前“2 秒内持久化，失败返回 error，耗时读取异步化”的方向保持不变。

### Q4：直接注册 dispatcher 是否支持？

判断为支持，官方 WebSocket 示例即采用这一模式。

### Q5：mentions 的处理

P2 原始事件的 mention 包含：

```text
Key
ID.OpenID / ID.UserID
Name
```

normalizer 后仍需用 bot identity 与 mention ID 比较，正确设置：

```go
message.MentionedBot = true
mention.IsBot = true
```

### Q6：RequireMention 的依赖

`PolicyGate.RequireMention=true` 依赖 `MentionedBot`。如果 bot identity 比较未命中，
所有群消息都会被静默拒绝。

### Q7：多个连接

多个连接随机消费而非广播，应视为 P0。

### Q8：主动发送成功但接收失败

优先核对：

1. `im.message.receive_v1` 订阅；
2. v1/v2 事件版本；
3. 应用版本发布；
4. 消息事件权限；
5. 应用可用范围；
6. 多连接；
7. mention matching；
8. 业务 sender allowlist。

### Q9：PolicyGate 拒绝日志

应在 PolicyGate 之前记录脱敏元数据，并在 Evaluate 后记录 reject reason。不能直接
打印完整事件 envelope 到公开日志，因为消息内容、open_id 和 chat_id 仍属于需要
保护的数据。

### Q10：先持久化后异步

方向正确。重投使用相同 event/message 标识时，SQLite 必须继续保持幂等。当前实现
优先使用消息 ID，对其他事件使用 event ID，并把去重占位和 Job 保存放在同一事务。

## 3. 修正后的 P0 排障顺序

### 第一步：只保留一个连接

```text
关闭所有本地旧进程
检查其他终端和机器
检查是否存在服务或计划任务
当前测试只启动一个实例
```

### 第二步：核对开发者后台

```text
im.message.receive_v1 已添加
事件版本与 P2 handler 匹配
长连接模式已启用
权限已审批
应用可用范围包含测试用户
包含变更的应用版本已发布
```

### 第三步：增加分层诊断

```text
raw handler hit
normalize result
mention match
PolicyGate decision
Gateway receipt
SQLite save
worker claim
reply result
```

### 第四步：更新测试用户 allowlist

当前真实测试用户不匹配本地绑定的：

```text
admin_user_ids
allowed_user_ids
```

只把用户加入 `allowed_user_ids`，不自动赋予管理员权限。

### 第五步：使用新 message_id 重测

旧消息不会自动补投。必须重新发送一条新的真实 mention 消息。

### 第六步：验证完整链路

```text
raw handler hit
-> receipt accepted
-> SQLite Job
-> 已接收回复
-> GitLink GET-only
-> result_json
-> 最终回复
-> duplicate message 不重复执行
```

## 4. 建议代码修订

### P0 可观察性

1. 增加 `raw_event_received` 脱敏事件；
2. 增加 `mention_evaluated`；
3. 增加 `policy_rejected` 和 reason；
4. 增加 `gateway_rejected` 和 reason；
5. 为未授权用户回复固定安全提示；
6. 为不支持命令回复帮助提示；
7. 不在这些提示中回显仓库敏感数据。

### P0 配置

1. 增加绑定文件诊断命令；
2. 输出当前群是否绑定、用户是否 allowed 的布尔值；
3. 不输出完整 open_id；
4. 启动时输出 enabled chat 数量和 allowlist 数量；
5. 支持显式测试模式，但默认保持最小权限。

### P1 部署

1. 不再依赖临时终端；
2. 使用 Windows 服务、计划任务或独立服务器；
3. 单实例租约；
4. 健康检查；
5. 进程退出告警；
6. 日志轮转；
7. 凭据存储。

## 5. 当前仍未完成

飞书 Agent 的回复提供了明确排障方向，但尚未完成：

- 开发者后台实际截图或配置核对；
- 事件版本实测；
- raw handler 可观察性代码；
- 测试用户 allowlist 修复；
- 新 message_id 的端到端验证；
- 常驻部署。

因此当前状态仍是：

> 应用鉴权、出站消息和 WebSocket 连接成立；入站消息到业务 handler 未成立。

## 6. 相关资料

- [飞书 Agent 排障交接](./FEISHU_AGENT_DEBUG_HANDOFF.md)
- [机器人完整能力总结](./FEISHU_REVIEW_BOT_CAPABILITY_SUMMARY.md)
- [P2.1 实施记录](./ROUND2_PR_REVIEW_COLLABORATION_P21_IMPLEMENTATION.md)
- [飞书 Channel SDK](https://open.feishu.cn/document/mcp_open_tools/integrating-agents-with-feishu/integrate-feishu-channel)
- [长连接事件订阅](https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN)
