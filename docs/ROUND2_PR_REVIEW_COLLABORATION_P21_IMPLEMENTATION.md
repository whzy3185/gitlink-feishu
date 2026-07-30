# 复赛 PR Review 协作 P2.1 实施记录

日期：2026-07-30

基线提交：

```text
4e86154a43c43921b17a594157142aa64112ba77
feat: add Feishu P2 review gateway foundation
```

开发分支：

```text
feat/round2-review-collaboration-p21-live-reply
```

当前机器人面向产品、协作和飞书开放平台的完整能力说明见：

```text
docs/FEISHU_REVIEW_BOT_CAPABILITY_SUMMARY.md
```

## 1. 定位与边界

P2.1 的目标是：

```text
真实飞书只读收发与可靠性门禁
```

当前链路扩展为：

```text
飞书测试群 @应用机器人
-> 官方 Channel SDK 长连接、事件归一化和策略校验
-> 2 秒 handler 预算内持久化事件和 Job
-> SQLite Job 租约、重试和进程重启恢复
-> GitLink GET-only Review Queue / Review Context
-> 持久化 feishu.review-result/v1
-> 回复原飞书消息
```

本阶段允许向飞书原消息发送：

- 已接收回执；
- Review Queue 只读摘要；
- PR Review Context 只读摘要；
- Review 草稿预览；
- 脱敏失败信息。

本阶段仍然禁止：

```text
从飞书创建 GitLink Review
批准或拒绝 PR
创建、回复、解决或重开行级评论
请求或移除 Reviewer
合并 PR
保存 GitLink 用户 Token
写飞书 Base、Doc 或 Task
把飞书状态覆盖为 GitLink 正式事实
```

所有执行结果继续明确：

```text
mode = preview
mutates_gitlink = false
GitLink 写入 = 0
```

## 2. 任务可靠性

### 2.1 SQLite 是 Job 的事实源

P2.0 的内存 `jobs chan ReviewGatewayJob` 已不再作为任务事实源。内存中只保留
唤醒信号；事件去重占位和 Job 保存合并为一个 SQLite 事务，worker 再从 SQLite
原子领取可执行任务。

任务持久化字段包括：

```text
attempt_count
max_attempts
next_attempt_at
lease_owner
lease_expires_at
result_json
handler_latency_ms
```

启动时和定时轮询时都会：

1. 领取 `queued` 且已到执行时间的任务；
2. 把租约已过期的 `running` 任务恢复为可领取状态；
3. 原子增加 `attempt_count`；
4. 失败后按退避时间重新排队；
5. 达到最大次数后保存最终失败结果。

默认 Job 最大执行次数为 3。任务在“SQLite 已保存、内存唤醒前”或执行中崩溃，
重启后仍可恢复，不再依赖飞书重投来找回任务。P2.0 若曾留下“已有 dedupe key、
没有 Job”的孤儿占位，P2.1 原子入队会识别并补建 Job。

### 2.2 结果和回复状态持久化

成功和最终失败都会保存完整的 `feishu.review-result/v1`。回复状态独立保存：

```text
reply_status
reply_attempt_count
reply_next_attempt_at
reply_lease_owner
reply_lease_expires_at
reply_message_id
reply_error_summary
```

回复 worker 使用独立租约，进程重启后会继续处理未完成回复。成功回复标记为
`sent`，相同 Job 不会在正常重启后再次发送。

发送成功但进程在写入 `sent` 前崩溃，仍存在平台 API 与本地数据库之间无法完全
消除的重复窗口。因此这里承诺的是可恢复的至少一次投递和业务幂等，不宣称跨系统
严格 exactly-once。

## 3. 飞书 3 秒门禁

命令新增：

```text
--handler-timeout-ms  默认 2000，最大 2500
--sqlite-timeout-ms   默认 500，最大 1000，且必须小于 handler 预算
```

监听 handler 中只执行：

```text
事件归一化
权限和绑定检查
SQLite Reserve
SQLite SaveJob
非阻塞输出和唤醒
```

SQLite 使用：

```text
PRAGMA busy_timeout=400
单连接串行写
单次 handler DB context 默认 500ms
```

终端 JSON 输出使用有界异步队列。stdout 阻塞时会丢弃本地遥测，不阻塞飞书
handler。每个已接受 Job 会异步记录 `handler_latency_ms`。

若持久化在预算内失败，handler 返回受控错误，让飞书保留重投机会；不得在尚未
持久化时返回伪成功。

## 4. 为什么消息事件直接注册到官方 dispatcher

当前锁定的 Channel SDK 为：

```text
github.com/larksuite/oapi-sdk-go/v3 v3.9.9
```

该版本的普通 `Channel.OnMessage` 会先进入内部批处理管道，并且不会把业务
handler 返回的错误继续返回给事件 dispatcher。对普通 Agent 对话这是合理默认值，
但不满足本阶段“SQLite 持久化失败必须让飞书重投”的合同。

P2.1 仍然复用官方 SDK 的：

- WebSocket 长连接；
- 消息 normalizer；
- bot identity；
- `PolicyGate`；
- `Channel.Send`；
- 卡片 action 处理。

普通消息事件改为在同一个官方 `EventDispatcher` 上注册同步适配器：

```text
P2MessageReceiveV1
-> normalize.ParseMessage
-> bot mention 标记
-> PolicyGate
-> 持久化 handler
-> error 原样返回 dispatcher
```

这样既保留 SDK 的标准模型和安全策略，又能兑现飞书回调的失败语义。

## 5. 群绑定和 mention 处理

首次绑定必须由管理员在本地配置文件完成。未绑定群不在 Channel SDK
`GroupAllowlist` 中，因此不能依赖未绑定群里的聊天命令完成首次绑定。

管理员可先只读发现应用可见群：

```powershell
go run . feishu +review-gateway --discover-chats --format json
```

再把目标 `chat_id` 写入未提交的 `.local/review-gateway-bindings.json`。

测试群策略为：

```text
只允许 enabled 绑定群
群消息必须 @机器人
不响应 @所有人
禁用私聊
可选发送者 allowlist
```

飞书 normalizer 会在正文中保留 bot mention key。P2.1 在策略校验完成后仅移除
已识别为机器人的 mention key，再解析“查看 PR #431”等命令；不会移除其他用户
mention。

## 6. 出站回复

收到有效命令后，回复调度器会尽力异步回复原消息：

```text
已接收只读 Review 请求
任务 ID
GitLink 写入：0
```

后台读取完成后，再通过 `ReplyMessageID` 回复同一条消息。最终回复是持久化的，
回执是非持久化的提示信息；即使回执发送失败，最终结果仍会按 SQLite 状态恢复。

当前回复使用有长度上限的文本，优先保证：

- 仓库和 PR；
- collection status / partial；
- Review 阶段和决定；
- Review、线程和未解决线程数量；
- head SHA；
- snapshot plan；
- 草稿摘要和建议下一步；
- `GitLink 写入：0` 边界。

卡片模板和卡片更新留到后续资源同步阶段，不阻塞 P2.1 的真实收发门禁。

## 7. 测试

### 7.1 自动化测试

目标包覆盖：

- P2.0 SQLite schema 原地迁移；
- 去重占位与 Job 保存原子提交，以及 P2.0 孤儿占位恢复；
- `running` 租约过期后重启恢复；
- 执行失败退避、最大次数和最终失败回复；
- 完整结果持久化；
- reply claim 和成功去重；
- SQLite 锁竞争时受控超时；
- stdout 阻塞不拖慢 handler；
- 回复原消息且包含 `GitLink 写入：0`；
- 仅预绑定群、必须 mention、禁用私聊；
- bot mention key 清理；
- callback token 不进入 Gateway Event。

本工作站执行：

```text
PASS  go test ./shortcuts/feishu
PASS  go vet ./shortcuts/feishu/...
PASS  git diff --check
```

`go test -race` 未执行成功，因为当前 Windows Go 环境未启用 CGO。该限制不能写成
race test 通过。

`go test ./...` 仍存在 P0 已记录的历史基线失败，包含未定义测试辅助函数、缺失
shortcut 和旧 API 路径期望等；本轮代码只修改 `shortcuts/feishu` 和相关文档，
没有扩张修复这些无关包。

### 7.2 真实测试环境

2026-07-30 已完成：

```text
飞书自建应用鉴权：通过
远端应用检查：5/5 通过
只读群发现：通过
测试群预绑定：通过
官方 Channel SDK WebSocket：连接成功
机器人 identity：获取成功
GitLink 公共 PR #431 GET-only：通过
```

真实验收仍需保留以下可观察证据：

```text
测试群 @机器人“查看 PR #431”
-> 2 秒内返回并持久化
-> 收到已接收回复
-> 收到最终只读 Review 回复
-> 重发同一 message_id 不产生第二个 Job 或最终回复
-> 重启后未完成任务自动恢复
```

在该链路实际跑通并保存脱敏记录前，不把 P2.1 标记为完整测试群验收通过。

## 8. 下一阶段

P2.1 关闭后，再进入协作状态和资源同步：

1. Reviewer 认领、释放、截止时间和冲突处理；
2. GitLink 事实字段与飞书人工字段分离；
3. Review Queue 卡片；
4. 多维表格 WorkItem；
5. Review Doc 和 Task；
6. merged / closed 归档；
7. partial 快照保护的真实 upsert；
8. 429、限流、权限失败和多实例共享存储。

这些能力不得改变当前 GitLink GET-only 门禁。正式 Review 写回仍属于 P3。

## 9. 官方依据

- Channel SDK 集成 Agent：<https://open.feishu.cn/document/mcp_open_tools/integrating-agents-with-feishu/integrate-feishu-channel>
- 长连接事件订阅：<https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN>
- 接收消息事件：<https://open.feishu.cn/document/server-docs/im-v1/message/events/receive?lang=zh-CN>
- Go SDK：<https://pkg.go.dev/github.com/larksuite/oapi-sdk-go/v3>
