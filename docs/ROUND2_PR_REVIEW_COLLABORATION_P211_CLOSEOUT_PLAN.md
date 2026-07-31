# 复赛 PR Review 协作 P2.1.1 收口与真实验收规划

日期：2026-07-31

状态：

```text
规划已执行
P2.1.1 代码门禁已完成
尚未执行新的真实飞书消息测试
```

> 2026-07-31 实施更新：单实例锁、分层脱敏观测、启动预检摘要和绑定群安全拒绝反馈已经实现，
> 定向 Go 测试通过。后续 P2–P5 实施见
> [复赛 PR Review 协作 P2–P5 实施记录](./ROUND2_PR_REVIEW_COLLABORATION_P2_P5_IMPLEMENTATION.md)。

规划基线：

```text
P2.1 代码提交：
fa3c1fb53dcb9f2fb0eeb9b558a32780f45ceeff
feat: add reliable Feishu live review replies

当前完整资料分支：
feat/feishu-complete-review-integration

当前资料提交：
e581e9036aec7baea113b024af11ebd414c4b7f5
docs: consolidate Feishu implementation and research
```

建议后续实现分支：

```text
fix/round2-review-collaboration-p211-live-gate
```

## 1. 结论

P2.1 的本地数据模型、任务恢复、最终回复恢复和 GitLink GET-only 执行框架已经成立，
但真实飞书端到端门禁仍未通过。

准确状态是：

> P2.1 代码门禁基本通过；P2.1.1 需要先补齐单实例、分层可观察性、权限预检和拒绝
> 反馈，再执行唯一一次有证据的真实群验收。

当前不能宣称：

```text
机器人已经可以稳定接收飞书群消息
机器人已经完成真实 PR Review 闭环
重新发布飞书应用后问题已经解决
WebSocket ready 等于入站链路可用
```

P2.1.1 不是增加 Base、Doc、Task 或 GitLink 写回能力，而是把已经存在的只读链路变成
可定位、可验证、可恢复、可审计的最小机器人。

## 2. 项目发心和服务对象

本阶段继续服务比赛结束时的真实场景：

```text
拥有者在短时间内收到大量 PR
-> 团队在飞书共享 Review 进度和上下文
-> gitlink-cli 提供稳定、结构化、Agent 友好的 GitLink 读取
-> Agent 取证、归纳和草拟
-> 人类 Reviewer 保留最终判断
-> GitLink 保留正式 Review、评论和合并事实
```

各系统的边界保持为：

| 系统 | 当前职责 |
| --- | --- |
| GitLink | PR、patchset、Review、线程和合并状态的事实源 |
| gitlink-cli | 面向人和 Agent 的稳定读取、聚合和执行面 |
| 飞书 | 群沟通、协作入口、进度共享和后续资源承载 |
| Agent | 取证、分析、摘要、风险提示和 Review 草稿 |
| 拥有者团队 | Reviewer 分工、最终 Review 和是否合并的决策 |

## 3. P2.1 已成立的能力

### 3.1 GitLink 只读合同

当前执行器只使用 GitLink GET-only 能力：

```text
Review Queue
Review Context
PR files
patchset versions
formal reviews
review threads
deterministic Review draft
```

所有 Job 继续固定：

```text
mode = preview
mutates_gitlink = false
GitLink 写入 = 0
```

### 3.2 SQLite 任务事实源

当前已经保存：

```text
message_id / event_id dedupe
Job payload
queued / running / completed / failed
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

事件去重占位和 Job 保存处于同一事务。进程重启后可以恢复未完成 Job 和最终结果回复。

### 3.3 处理预算

当前默认预算：

```text
handler 总预算：2000 ms
SQLite 预算：500 ms
SQLite busy_timeout：400 ms
GitLink 网络读取：异步执行
stdout：非阻塞输出
```

飞书官方要求长连接事件在 3 秒内完成处理且不抛出异常，否则会触发超时重推。本项目用
2 秒内部预算保留了安全余量。参考：

- [使用长连接接收事件](https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN)
- [事件订阅概述](https://open.feishu.cn/document/server-docs/event-subscription-guide/overview?from=from_parent_docs)

### 3.4 最终回复恢复

最终 `feishu.review-result/v1` 保存到 SQLite，回复 worker 使用独立租约和重试状态。
进程退出后，尚未发送成功的最终结果可以继续恢复。

最终回复必须包含：

```text
GitLink 写入：0
```

## 4. 必须纠正的能力表述

### 4.1 “已接收”回执不是可靠交付

当前“已接收”回执进入内存 `acks` channel：

```text
TryAcknowledge
-> 内存队列
-> Channel.Send
```

因此以下情况可能丢失回执：

```text
acks 队列已满
进程在发送前退出
飞书发送失败
网络瞬时失败
```

当前准确表述必须是：

> 收到有效命令后，机器人尽力发送“已接收”回执；最终 Review 结果回复才具备 SQLite
> 持久恢复能力。

P2.1.1 的退出条件不要求把 ack 改造成持久队列。ack 持久化可作为后续可靠性增强，
但不得用 ack 是否出现替代最终回复门禁。

### 4.2 WebSocket ready 不是业务可用

以下日志只证明连接和机器人身份成立：

```text
event-dispatch is ready
bot identity resolved
WebSocket 连接成功
feishu.review-gateway/v1 ready
```

它们不证明：

```text
im.message.receive_v1 已投递
P2 handler 已命中
mention 匹配成功
PolicyGate 已放行
业务 allowlist 已放行
SQLite 已保存 Job
飞书最终回复成功
```

## 5. 2026-07-31 真实排障事实

### 5.1 已发现三个同时运行的连接

本机曾同时存在三个 `feishu +review-gateway --listen` 实例。

飞书长连接为集群消费模式。同一应用存在多个客户端时，消息只会随机投递给其中一个
连接，不会广播。每初始化一个 client 就占用一个连接。参考：

- [使用长连接接收事件](https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN)

因此“多实例”从排障 P1 提升为 P0。

### 5.2 当前代码没有 raw handler 命中日志

当前代码的第一条业务输出发生在：

```text
normalize
-> bot identity
-> mention 匹配
-> PolicyGate
-> Gateway
-> queue.Enqueue
-> receipt 输出
```

如果消息在 normalize、bot identity 或 PolicyGate 阶段停止，终端可能完全没有 receipt。
飞书 Agent 建议的以下日志前缀已经以结构化 `feishu.review-observation/v1` 实现：

```text
[RAW]
[normalized]
[policy]
[job]
```

所以验收前必须先补可观察性，不能等待代码并不会产生的日志。

### 5.3 PolicyGate 拒绝目前静默

当前逻辑：

```go
if decision := c.policy.Evaluate(message); !decision.Allowed {
    return nil
}
```

因此下列情况对终端和用户都可能表现为“没有反应”：

```text
群不在 allowlist
没有真正 @ 机器人
仅 @所有人
私聊被禁用
MentionedBot 标记未命中
```

### 5.4 业务权限拒绝只写本地 receipt

当前 `sender_not_allowed`、`chat_not_bound`、`unsupported_read_only_command` 等拒绝
不会进入 accepted Job，也不会发送“已接收”回执。

其中：

- 未绑定群和未真实 mention 应保持静默，避免扩大机器人暴露面；
- 已绑定群内的 `sender_not_allowed` 应返回不泄露内部配置的权限提示；
- 已绑定群内的未知只读命令应返回帮助提示。

### 5.5 旧消息不会在监听器上线后补投

2026-07-31 的一次观测中：

```text
测试群最后一条 @gitlink 查看 PR #431：13:22
唯一监听实例 ready：13:30:03
```

该消息早于 ready，不能作为新版本验收消息。

## 6. P2.1.1 实现范围

### 6.1 P0：单实例门禁

目标：

```text
同一工作目录、同一 App、同一 state DB 默认只能运行一个监听实例
```

设计：

1. 启动监听前获取本地实例锁；
2. 锁文件保存在 `.local`，不得提交；
3. 锁元数据只保存 PID、启动时间、state DB 指纹和脱敏 App 指纹；
4. 锁冲突时 fail closed，输出已有实例信息后非零退出；
5. 进程正常退出释放锁；
6. 异常退出后允许识别失效 PID 并安全接管；
7. 后续若需要生产多实例，必须显式切换到共享存储和分布式租约设计，不得通过跳过锁实现。

验收：

```text
第一个实例启动成功
第二个实例启动失败且错误明确
第一个实例退出后新实例可启动
失效 PID 锁可恢复
不同 state DB 的行为有明确测试
```

### 6.2 P0：分层结构化可观察性

新增只包含脱敏元数据的观察合同：

```text
feishu.review-observation/v1
```

建议阶段：

```text
raw
normalized
identity
policy
gateway
job
result
reply
```

最小字段：

```json
{
  "schema_version": "feishu.review-observation/v1",
  "stage": "raw",
  "instance_id": "local-random-id",
  "event_id": "redacted-or-hashed",
  "message_id": "redacted-or-hashed",
  "chat_id_hash": "sha256-prefix",
  "sender_id_hash": "sha256-prefix",
  "chat_type": "group",
  "mention_count": 1,
  "allowed": null,
  "reason": "",
  "latency_ms": 0,
  "observed_at": "RFC3339"
}
```

安全要求：

```text
不输出消息正文
不输出完整 chat_id / open_id / user_id
不输出 App ID / App Secret / token / webhook
不序列化完整原始 event
message_id 和 event_id 仅保留受控摘要或运行期关联值
错误继续走现有脱敏函数
```

日志顺序：

```text
进入 OnP2MessageReceiveV1 第一行
-> raw
normalize 完成
-> normalized
bot identity 与 mention 匹配完成
-> identity
PolicyGate 判定
-> policy
Gateway receipt
-> gateway
SQLite 原子保存成功
-> job
执行完成
-> result
最终回复完成
-> reply
```

`raw` 必须在 normalize 和 PolicyGate 之前无条件记录。

### 6.3 P0：开发者后台配置快照

真实测试前，由应用管理员核对并记录：

```text
企业自建应用
机器人能力已开启
事件订阅方式为长连接
已添加“接收消息 v2.0”
事件类型为 im.message.receive_v1
具备群聊中 @机器人消息的只读权限
最新应用版本已发布
应用可用范围包含测试用户
机器人已在测试群
```

飞书官方“接收消息”文档明确要求订阅“接收消息 v2.0”，并说明群聊 @机器人消息所需的
权限以及特殊情况下应按 `message_id` 去重：

- [接收消息事件](https://open.feishu.cn/document/server-docs/im-v1/message/events/receive?lang=zh-CN)

快照只记录开关、版本号、权限名称和发布时间，不记录 App Secret。

### 6.4 P0：测试发送者权限预检

测试前必须确认：

```text
测试群 chat_id 与 binding 一致
binding enabled = true
测试用户 open_id 存在于 allowed_user_ids
管理员能力仅在确有需要时加入 admin_user_ids
binding_revision 已更新
```

规则：

1. 测试用户只需要只读命令权限，不自动授予管理员权限；
2. 完整 open_id 只保存在 `.local` 配置；
3. 日志和报告只记录哈希匹配结果；
4. 配置更新后必须重启监听器；
5. 启动时输出绑定数量、启用数量、allowed/admin 数量，不输出真实 ID。

### 6.5 P1：安全拒绝反馈

在已绑定群、真实 @机器人且消息可安全回复的前提下：

| 拒绝原因 | 飞书反馈 |
| --- | --- |
| `sender_not_allowed` | “当前账号没有该仓库 Review 读取权限，请联系群管理员。” |
| `unsupported_read_only_command` | 返回只读帮助和支持的命令 |
| `stale_event` | 不回复，只记录观察事件 |
| `duplicate_event` | 不产生第二个 Job，不产生第二条最终回复 |
| `chat_not_bound` | 保持静默 |
| PolicyGate 未 mention | 保持静默 |
| 私聊禁用 | 保持静默 |

拒绝回复不得包含：

```text
真实 allowlist
管理员 ID
绑定文件路径
App 信息
内部错误
GitLink Token
```

### 6.6 P1：ack 语义固定

P2.1.1 保持：

```text
ack = best-effort
final reply = persistent and recoverable
```

文档、演示词和验收报告统一使用：

> 机器人尽力发送“已接收”提示；最终 Review 结果由持久化回复队列保证恢复。

禁止继续写：

```text
一定先收到“已接收”
ack exactly-once
ack 可崩溃恢复
```

### 6.7 P1：真实验收辅助脚本

新增只读验收辅助脚本，职责仅限：

```text
检查相关进程数量
检查配置字段是否存在
启动前生成 run_id
记录 ready 时间
跟踪脱敏 observation
查询 SQLite 状态
生成验收报告
结束后清理本轮临时进程
```

脚本不得：

```text
保存或打印 Secret
代替真实用户生成入站消息
修改 GitLink
修改飞书后台
自动授予测试用户权限
提交 .local 数据
```

## 7. 预计代码影响

| 范围 | 预计变更 |
| --- | ---: |
| 单实例锁和启动检查 | 80–140 行 |
| 结构化 observation 与脱敏 | 120–180 行 |
| Policy/Gateway 拒绝反馈 | 70–120 行 |
| 启动配置摘要和权限预检 | 40–80 行 |
| 自动化测试 | 220–340 行 |
| smoke 辅助脚本和报告模板 | 120–220 行 |
| 文档修订 | 100–180 行 |
| 合计 | 750–1260 行 |

该估算包含测试和文档，不代表运行时代码需要新增一个独立 Agent。P2.1.1 继续使用现有
Review Gateway、SQLite worker 和确定性执行器。

## 8. 自动化测试规划

### 8.1 单实例

新增测试：

```text
TestReviewGatewayInstanceLockRejectsSecondListener
TestReviewGatewayInstanceLockReleasesOnClose
TestReviewGatewayInstanceLockRecoversStalePID
TestReviewGatewayInstanceLockDoesNotExposeAppID
```

### 8.2 分层观察

新增测试：

```text
TestReviewGatewayEmitsRawObservationBeforeNormalize
TestReviewGatewayEmitsPolicyDecision
TestReviewGatewayObservationRedactsIdentifiersAndContent
TestReviewGatewayObservationQueueDoesNotBlockHandler
```

### 8.3 权限和拒绝

新增测试：

```text
TestReviewGatewayBoundUnauthorizedSenderGetsSafeReply
TestReviewGatewayUnknownCommandGetsHelp
TestReviewGatewayUnboundChatRemainsSilent
TestReviewGatewayNoMentionRemainsSilent
```

### 8.4 ack 与最终回复

保留并扩展：

```text
ack 队列满时 Job 仍能完成
ack 发送失败时最终回复仍可发送
进程重启后最终回复继续恢复
reply_status=sent 后不重复发送
```

### 8.5 幂等

飞书客户端无法手工重新发送相同的 `message_id`。因此幂等验收拆成：

1. 真实群发送一条全新消息，保存脱敏事件引用；
2. 自动化或受控集成测试将同一 `message_id` 再交给同一 SQLite queue；
3. 断言第二次 receipt 为 `duplicate_event`；
4. 断言 Job 数量仍为 1；
5. 断言最终 `reply_message_id` 仍只有 1 个。

不得要求用户在飞书客户端“重复发送同一个 message_id”。

## 9. 测试层级

### 9.1 L0：静态和定向测试

```powershell
go test ./shortcuts/feishu
go test ./shortcuts/workflow ./shortcuts ./shortcuts/pr
go vet ./shortcuts/feishu/...
git diff --check
```

`go test ./...` 的历史基线失败单独记录，不得混写为本轮新增失败，也不得伪造全仓通过。

### 9.2 L1：离线合同

使用受控 fixture：

```powershell
go run . feishu +review-gateway `
  --from-event shortcuts/feishu/testdata/review_gateway_event.json `
  --bindings shortcuts/feishu/testdata/review_gateway_bindings.json `
  --execute-read-only `
  --format json
```

验证：

```text
命令解析
绑定
allowlist
dedupe
GitLink GET-only
Review result
GitLink 写入：0
```

### 9.3 L2：OpenAPI 只读预检

```text
tenant token 获取成功
bot identity 获取成功
测试群发现成功
绑定与测试群一致
公开 PR #431 GET-only 完整读取
```

这些检查不等于入站消息验收。

### 9.4 L3：真实飞书端到端

只有 L0、L1、L2 全部通过后才执行。

## 10. 唯一一次真实验收流程

### 10.1 测试前

必须同时满足：

```text
[x] P2.1.1 代码和定向测试通过
[ ] 开发者后台配置快照完成
[ ] 当前版本已发布
[ ] 当前用户已进入 allowed_user_ids
[ ] 所有旧监听实例已退出
[ ] 单实例锁测试通过
[ ] 只启动一个监听器
[ ] ready 后记录 T0 和 run_id
[ ] 新 SQLite 文件为空
[ ] PR #431 当前仍可 GET-only 读取
```

### 10.2 用户触发

监听器 ready 之后，真实用户在已绑定测试群通过 @ 候选选择机器人并发送：

```text
@gitlink 查看 PR #431
```

记录：

```text
发送时间 T1
群名
命令
客户端显示的发送成功状态
```

不在报告中保存完整 chat_id、open_id 或消息正文以外的聊天历史。

### 10.3 五秒内观测

期望顺序：

```text
raw
-> normalized
-> identity mentioned_bot=true
-> policy allowed=true
-> gateway accepted=true reason=queued_preview
-> job saved
```

必须满足：

```text
message_id 存在
handler_latency_ms < 2000
SQLite review_gateway_events = 1
SQLite review_gateway_jobs = 1
Job 初始状态为 queued 或 running
```

ack：

```text
允许出现
不作为硬门禁
失败必须有本地脱敏 ack_reply 记录
```

### 10.4 最终结果

必须满足：

```text
GitLink GET-only 查询成功
collection_status = complete 或有明确 partial 说明
Job status = completed
result_json 存在
reply_status = sent
reply_message_id 存在
最终回复是原消息的回复
最终回复包含“GitLink 写入：0”
```

### 10.5 幂等复核

通过受控集成入口再次提交同一事件引用：

```text
receipt.reason = duplicate_event
Job 总数仍为 1
最终 reply_message_id 数量仍为 1
不产生第二条最终回复
```

### 10.6 清理

测试完成后：

```text
停止本轮临时监听器
确认没有遗留 review-gateway 进程
保留脱敏报告
保留未提交 SQLite 作为本地证据
不得提交 Secret、DB、WAL、SHM、原始日志或真实用户 ID
```

## 11. 失败分流

| 最后出现的阶段 | 结论 | 下一步 |
| --- | --- | --- |
| 无 `raw` | 飞书未把事件交给当前 handler | 查订阅、版本、权限、可用范围和连接数 |
| 有 `raw`、无 `normalized` | normalizer 或事件版本问题 | 保存脱敏 envelope 形状并修 normalizer |
| 有 `normalized`、identity 未命中 | mention 与 bot identity 匹配问题 | 核对 OpenID/UserID/mention key |
| policy denied | Channel 安全策略问题 | 根据 reason 调整绑定或 mention |
| gateway `sender_not_allowed` | 业务 allowlist 问题 | 更新 `.local` allowed 用户后重启 |
| gateway accepted、无 Job | SQLite 原子保存问题 | 检查预算、锁竞争和事务 |
| Job failed | GitLink GET-only 或执行器问题 | 查看结构化 fetch error |
| Job completed、reply 未 sent | 飞书出站或回复 worker 问题 | 检查 reply lease、错误和权限 |
| reply sent、群里不可见 | ReplyMessageID 或客户端展示问题 | 用返回 message_id 查消息 |

## 12. 证据包

本地证据目录：

```text
.local/p21-acceptance/<run_id>/
```

内容：

```text
preflight.json
instance-summary.json
observations.jsonl
receipt.json
job-summary.json
reply-summary.json
sanitized-listener.log
acceptance-report.md
```

允许提交到 Git 的只有重新脱敏后的：

```text
reports/ROUND2_P21_ACCEPTANCE_<date>.md
```

报告必须写明：

```text
分支和 commit
应用版本号和发布时间
run_id
ready 时间
用户发送时间
各阶段是否出现
handler_latency_ms
Job status
reply_status
是否存在 reply_message_id
GitLink 写入：0
未执行的验证
```

## 13. P2.1.1 退出门禁

| 门禁 | 当前状态 | 退出要求 |
| --- | --- | --- |
| GitLink GET-only | 通过 | 保持写入为 0 |
| SQLite Job 恢复 | 通过 | 回归测试继续通过 |
| 最终回复恢复 | 通过 | 真实回复 `sent` |
| ack 可靠性 | best-effort | 文档表述准确 |
| 单实例 | 未通过 | 第二实例默认拒绝启动 |
| raw 可观察性 | 未通过 | handler 第一行有脱敏 observation |
| PolicyGate 可观察性 | 未通过 | allowed/reason 可定位 |
| 测试用户权限 | 待预检 | allowed 匹配成立 |
| 开发者后台配置 | 待留证 | v2.0、权限、发布、范围完整 |
| 真实入站消息 | 未通过 | 新 message_id 进入 Gateway |
| handler 预算 | 待真实验证 | `< 2000 ms` |
| Job 完成 | 待真实验证 | `completed` |
| 飞书最终回复 | 待真实验证 | `reply_status=sent` |
| 幂等 | 自动化已覆盖、真实证据待补 | 同事件不产生第二个 Job/回复 |

只有所有 P0 门禁和真实最终回复门禁通过，才能写：

```text
P2.1 真实飞书只读收发与可靠性门禁关闭
```

## 14. 实施顺序

后续代码阶段严格按以下顺序：

1. 新建 P2.1.1 修复分支；
2. 增加单实例锁及测试；
3. 增加 raw/normalized/identity/policy/gateway observation 及脱敏测试；
4. 增加启动配置摘要和测试用户权限预检；
5. 增加受控拒绝反馈；
6. 固定 ack best-effort 文档语义；
7. 增加验收辅助脚本和报告模板；
8. 完成 L0、L1、L2；
9. 管理员完成开发者后台配置快照；
10. 只启动一个实例；
11. 执行一次真实新消息验收；
12. 完成同事件幂等集成复核；
13. 提交脱敏验收报告；
14. 关闭 P2.1，进入 P2.2。

## 15. P2.2 入口

P2.1.1 关闭前，不开始以下正式同步：

```text
Review Queue 交互卡片
Reviewer 领取与释放
截止时间
多维表格 WorkItem
Review 云文档
飞书 Task
merged / closed 自动归档
飞书账号与 GitLink identity 绑定
```

P2.2 仍保持 GitLink GET-only。GitLink 正式 Review、评论、Reviewer 和合并写回继续属于
更晚阶段，并需要独立授权、权限模型、审计和回滚设计。

## 16. 本规划停止点

本轮只交付规划文档：

```text
不修改 Review Gateway 代码
不启动机器人
不发送真实飞书消息
不执行 GitLink 写入
不执行飞书资源写入
不创建 PR
```

P2.1.1 已实现；当前仅保留真实测试群证据验收。
