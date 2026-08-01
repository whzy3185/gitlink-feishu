# GitLink 飞书 PR Review 协作 P2–P5 交接

日期：2026-08-01

分支：`feat/round2-review-collaboration-p2-p5`

当前提交：`de59c3930b4c34c2c30cc6ab1dd91cc6641bd30a`

## 1. 核心结论

P2.1 已于 2026-08-01 完成真实端到端验证：

```text
飞书群 @机器人
-> im.message.receive_v1
-> Channel SDK 长连接
-> Gateway 策略和 SQLite Job
-> GitLink PR #431 GET-only
-> 飞书回执和最终回复
```

两次回复均明确显示 `GitLink 写入：0`。这表示本次命令只读取 GitLink 并把结果返回
飞书，不修改 GitLink PR；不是说项目永久没有写入能力。P3 已实现受控 `common Review`
写回代码，但尚未在专用测试 PR 上进行真实写入验收。

当前平台验收重点已经从“飞书消息能否到达”转移到：

1. Base、Doc、Task 首次写入和幂等；
2. 交互卡片动作；
3. 多仓库群绑定；
4. OAuth 和身份映射；
5. 企业微信真实链路；
6. GitLink 专用测试 PR 的受控 Review 写回。

第二轮飞书平台答复已经过官方文档核验。Task v2 的 `client_token`、负责人、关注人和全天
截止日期合同已补入代码；卡片延时更新、Base 事件、资源 owner 等未完成项仍保持待验收。
核验详情见 `FEISHU_AGENT_PLATFORM_RESPONSE_AUDIT_20260801.md`。

## 2. P2–P5 实现摘要

### P2：飞书协作 Gateway

- 企业自建应用、官方 Go SDK、Channel SDK 和 WebSocket 长连接；
- `im.message.receive_v1` 入站；
- 群 allowlist、用户 allowlist、要求 @、私聊禁用；
- 单实例锁、事件去重、SQLite Job、租约、重试和崩溃恢复；
- 最终回复状态持久化；
- Review Queue、Review Context 和 canonical WorkItem；
- 领取、释放、截止日期和人工协作审计；
- partial 快照不得覆盖完整事实；
- merged/closed 自动归档；
- 交互卡片、Base、Doc、Task Publisher 及资源指纹合同；
- 飞书资源写入必须显式增加 `--sync-feishu-resources`。

### P3：受控 common Review 写回

- 只支持 `common Review`；
- 飞书用户必须绑定 GitLink login；
- 生成 15 分钟 ActionPlan，必须由同一用户二次确认；
- 确认时重新校验 complete、partial、head SHA 和 SourceFingerprint；
- 校验 GitLink Token 的 `/users/me` 身份；
- 只有显式 `--enable-gitlink-review-write` 才允许 POST；
- POST 前持久化 `remote_write_possible`；
- 网络不确定或本地完成状态失败后进入人工对账，禁止自动重试；
- approve、reject、行级评论、解决线程、Reviewer 管理和 merge 继续禁止。

### P4：企业微信适配

- 飞书和企业微信共享 `gitlink.review-work-item/v1`；
- 官方 Node SDK sidecar 加本地 Go Review Core；
- sidecar 不持有 GitLink Token，Core 不持有企业微信 Secret；
- 固定 HTTP JSON 合同，不执行 shell；
- chat/user allowlist 默认 fail-closed；
- 入站只接受 Review 只读命令；
- 真实企业微信长连接和流式回复尚待验收。

### P5：多 Agent Review 协议

- `workflow +review-orchestrate` 生成只读 Agent 分工；
- correctness、tests、security、contributor experience、integration 五类任务；
- 支持 Context 增量文件范围；
- `review-synthesize` 校验 run、head、task、role、完成时间和证据；
- stale、重复、缺证据和角色不匹配结果不计入完成；
- 缺失或冲突时保持 `incomplete` 和 `human_decision_required`；
- `review-warroom` 聚合多仓库事实并叠加 P2 人工协作字段；
- P5 不自行启动模型，实际 Agent 由外部 Agent Host 执行。

## 3. 当前绑定模型

当前 SQLite BindingStore 已支撑 P2.1 真实链路。现有模型是：

```text
一个 Gateway -> 多个飞书群
一个飞书群 -> 一个默认 GitLink 仓库
```

当前变更绑定仍依赖本地受控配置和重启。下一步希望支持：

```text
一个群 -> default_repository + repositories allowlist
```

不会直接开放无边界的 `repository: "*"`。

## 4. 已完成的先行核实

### 4.1 OAuth 回调

飞书官方 OAuth 授权页支持把用户重定向到预先登记的 `redirect_uri`，并通过 HTTP GET
查询参数返回一次性 `code`。后端可以接收 code，再调用 OAuth token 接口换取
`user_access_token`。因此飞书侧 OAuth 回调的技术链路成立。

当前未解决的是 GitLink 侧身份授权方式，以及是否需要保存飞书 refresh token。若只把
飞书 `open_id` 映射为 GitLink login，而不以用户身份调用飞书 OpenAPI，则应避免申请和
保存长期飞书用户 Token。

官方资料：

- <https://open.feishu.cn/document/common-capabilities/sso/api/obtain-oauth-code>
- <https://open.feishu.cn/document/authentication-management/access-token/get-user-access-token?lang=zh-CN>

### 4.2 群管理员识别

官方权限中 `im:chat.moderation:read` 指的是读取群发言权限，不是读取群管理员列表。
当前公开资料可以通过群信息获得群主，并提供添加、删除群管理员的写接口，但本轮未确认
存在可供机器人读取全部群管理员的稳定接口。

因此当前安全基线仍采用：

```text
应用级管理员 allowlist
+ binding.admin_user_ids
+ 可核实的群主身份
```

在飞书侧明确存在可靠的管理员读取能力之前，不通过操作失败或成员列表猜测管理员身份。

### 4.3 Base 幂等

当前公开记录 API提供创建、查询、更新和批量操作，但本轮未确认存在按业务唯一键执行的
原子 upsert。现有实现使用：

```text
review_collaboration_resources
remote_id
content_fingerprint
```

即先由本地资源映射锁定远端 record，再执行更新；远端成功但本地保存失败时进入 unknown
和人工对账，不盲目再次创建。仍希望飞书侧确认是否有更成熟的条件更新或唯一键方案。

## 5. 需要飞书 Agent 直接评估的问题

### Q1：Base 是否适合作为机器人配置源

当前 SQLite BindingStore 已经跑通，考虑 Base 不是因为本地存储容量不足，而是希望群管理员
能够在飞书内自助维护“群 -> 仓库集合”的映射。

请评估：Base 是否适合作为机器人配置源？如何避免普通编辑者扩大仓库访问范围？推荐使用
高级权限、指定视图、审批自动化，还是继续由 SQLite 作为真源、Base 只作为管理界面？

### Q2：应用身份还是用户身份创建协作资源

当前 Base、Doc、Task Publisher 默认使用应用身份。应用身份创建的资源所有者不是具体
Owner 用户。

请评估：在团队 PR Review 场景中，这是否可接受？推荐由应用统一持有资源并添加协作者，
还是通过 OAuth 让 Owner 用户身份创建资源？两者在离职、权限回收和长期维护方面如何取舍？

### Q3：PR Review 交互卡片规范

当前已经接入 `OnCardAction`，也已有 WorkItem 卡片结构，但按钮状态和真实卡片动作尚未验收。

请建议适合 PR Review 的卡片信息层级和按钮设计，特别是领取、释放、刷新、打开证据、设置
截止日期和生成 Review 草稿。是否有按钮频率、卡片更新次数、回调超时或跨用户更新限制？

### Q4：同一个 PR 的消息维护策略

推荐持续更新一条固定卡片、回复原请求形成线程，还是每次状态变化发送新消息？如何同时满足
低噪声、可审计和多人协作？

### Q5：Base 资源同步幂等

代码已有 `remote_id + content_fingerprint` 本地映射。飞书 Base 是否提供按业务 key 的原子
upsert、条件更新或等价机制？若只能“查询后创建”，官方建议怎样防止多实例竞态？

### Q6：群管理员安全识别

在不申请过度权限的前提下，机器人应如何可靠判断发起者是群主或群管理员？如果无法读取完整
管理员列表，是否推荐应用管理员 allowlist、群主校验或独立审批流程？

### Q7：飞书身份与 GitLink 身份绑定

飞书 OAuth 的 HTTP GET code 回调已经确认可实现。当前消息事件只有 `open_id`，尚无 GitLink
login。请评估标准流程：机器人发送绑定链接、后端接收飞书 code、确认飞书身份，再跳转
GitLink 授权或由 GitLink 提供一次性绑定码。如何做到不在消息、Base 或日志中保存 Token？

### Q8：Base Review Queue 字段和视图

请建议字段、索引和视图设计，覆盖 repository、PR、head、fingerprint、完整性、风险、决策、
负责人、截止日期、归档状态和人工下一步，并支持仓库、负责人、风险、超期、看板和甘特视图。

### Q9：截止日期提醒

截止提醒应优先使用 Base 自动化、Task v2 原生提醒，还是机器人定时扫描？哪种方式在幂等、
权限、提醒噪声和审计方面更适合？

### Q10：长连接生产部署

飞书长连接多实例采用随机投递而非广播。推荐怎样部署主备、限制活动连接、观测连接健康、
执行事件去重和处理进程崩溃？是否有官方推荐的租约或 leader election 模式？

### Q11：正式版本最小权限

真实只读链路当前需要机器人能力、`im.message.receive_v1`、群内 @机器人消息读取和机器人
发消息/回复权限。Base、Doc、Task、卡片动作和 OAuth 应按功能增量申请。

请给出生产最小权限清单，并区分：只读 Review、卡片动作、Base、Doc、Task、OAuth 六组能力，
以便回收测试阶段临时开通的多余权限。

### Q12：成熟集成范式

在 Base、Doc、Task 各完成一条真实链路后，希望对标飞书已有的研发协作或 Agent 集成范式。
请指出最接近本项目的官方示例、Channel SDK 模式或可复用模板。

## 6. 当前验收状态

| 项目 | 状态 |
|---|---|
| 飞书真实消息入站 | 已通过 |
| SQLite Job 和 GitLink GET-only | 已通过真实正常路径 |
| 飞书回执和最终回复 | 已通过 |
| 本次 GitLink 写入为 0 | 已确认 |
| 同 message_id 去重与重启恢复演练 | 待补强证据 |
| Base、Doc、Task 真实同步 | 待验收 |
| Task v2 client_token、assignee、follower、due 请求合同 | 已实现，真实写入待验收 |
| common Review 真实测试 PR 写回 | 待验收 |
| 企业微信真实链路 | 待验收 |
| P5 多 Agent 协议和测试 | 已实现 |
| Round 2 Review Collaboration Gate | 已通过 |

## 7. 安全边界

- 默认 GitLink 写入为 0；
- 默认不创建飞书 Base、Doc、Task；
- 高风险能力必须显式启动；
- 群、用户、仓库都经过绑定或 allowlist；
- Agent 只能给出证据和建议；
- Owner 保留最终 Review 和合并决定；
- 所有 Token、Secret、Webhook key 和原始会话 ID 均不得进入日志或协作文档。

第二轮针对飞书权限、卡片、Base、Task、长连接和真实验收的详细平台问题见：

```text
docs/FEISHU_AGENT_PLATFORM_FOLLOWUP_20260801.md
docs/FEISHU_AGENT_PLATFORM_RESPONSE_AUDIT_20260801.md
```
