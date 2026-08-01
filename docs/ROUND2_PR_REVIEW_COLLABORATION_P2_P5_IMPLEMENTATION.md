# 复赛 PR Review 协作 P2–P5 实施记录

> 本文是 `feat/round2-review-collaboration-p2-p5` 的历史实施快照。Installation v2、
> 多仓库、固定卡片 PATCH、GitLink Webhook、外部 Agent Runner 和企业微信多仓库能力的
> 当前事实以 [协作平台 v2 实现说明](./ROUND2_FEISHU_PLATFORM_V2_IMPLEMENTATION.md) 为准。

日期：2026-07-31
分支：`feat/round2-review-collaboration-p2-p5`

## 结论

P2.1.1、P2.2、P3、P4、P5 的代码合同已经实现。当前准确状态是：

> 本地代码门禁完成；飞书消息到 GitLink GET-only 再到飞书最终回复的真实主链路已于
> 2026-08-01 通过。Base、Doc、Task、企业微信和 GitLink 写入仍需在测试资源中单独验收。

本轮没有在实现或测试过程中执行真实 GitLink Review、批准、拒绝、评论、Reviewer 管理或合并。

真实飞书验收使用 `Gitlink/gitlink-cli` PR #431。机器人先返回 Job 回执，再返回
`triaged / pending / unassigned` 的最终结果；两次回复均显示 `GitLink 写入：0`。

## P2.1.1：真实飞书链路收口

已完成：

- 每个状态库只允许一个 Review Gateway 实例；
- 同主机存活 PID 拒绝第二连接，失效锁可回收；
- `raw -> normalized -> identity -> policy -> gateway -> result` 分层观测；
- event/message/chat/user ID 只记录短哈希，不记录消息正文；
- 启动时输出绑定数、允许用户数、管理员数、状态库哈希和处理预算；
- 未绑定群、未 @、私聊保持静默；
- 已绑定群中的无权限用户和不支持命令获得安全提示；
- ack 继续是尽力发送，最终结果继续由 SQLite 恢复。

## P2.2：飞书协作闭环

同一个 PR 使用稳定 `pr_key`，GitLink 事实与人工协作字段分离：

- GitLink 事实：head、fingerprint、Review 阶段、决策、完整性；
- 人工字段：负责人、协作状态、截止时间；
- `领取 PR`、`释放 PR`、`设置 PR 截止` 写入本地协作状态；
- 每次人工变更写入 `review_collaboration_audit`；
- partial 快照不能覆盖已有完整事实；
- merged/closed 自动进入归档状态；
- 同一 WorkItem 生成飞书卡片、Base record、Doc Markdown 和 Task candidate；
- Task v2 创建包含内容稳定的 `client_token`、`open_id` 负责人/关注人和全天截止日期；
- 最终回复可直接使用飞书交互卡片；
- 可选 Publisher 将同一 WorkItem 同步到飞书多维表格、云文档和任务；
- `review_collaboration_resources` 保存远端 ID 与内容指纹，避免重复创建任务或重复追加同一文档快照；
- 远端写入成功但本地幂等状态保存失败时标记为 `unknown`，不伪报成功，也不自动重试。

Base、Doc 和 Task 的真实 OpenAPI 写入必须显式增加 `--sync-feishu-resources`。至少配置一个
同步目标；Base 还必须同时提供 App Token 与表 ID。Gateway 不会因为一次只读查询隐式创建
远端资源。

## P3：受控 common Review 写回

仅实现 `common` Review：

1. 飞书账号必须显式绑定 GitLink login；
2. 读取完整 Review Context 和当前 head；
3. 生成 15 分钟有效的 `review.action-plan/v1`；
4. 同一飞书账号发送 `确认 Review <plan-id>`；
5. 重新读取 head 和完整性；
6. 同时比较最新 `SourceFingerprint`，Review 或线程事实变化会使计划失效；
7. 校验当前 GitLink Token 的 `/users/me` 身份；
8. 使用带超时的执行租约；只允许在远端写入边界之前恢复过期执行；
9. 只有启动时显式传入 `--enable-gitlink-review-write` 才执行 POST；
10. POST 前持久化 `remote_write_possible` 对账边界；
11. 成功后保存 Review ID；
12. 网络结果或本地完成状态不确定时进入 `unknown_needs_reconciliation`，禁止自动重试。

以下仍强制禁用：

```text
approved
rejected
line comment
resolve thread
reviewer management
merge
```

## P4：企业微信第二适配

新增平台无关合同：

```text
internal/collab
└── gitlink.review-work-item/v1
```

飞书和企业微信共同消费该模型。企业微信实现包括：

- `wecom +check`：只校验配置，不发送；
- `wecom +review-notify`：模板卡片预览，只有 `--send` 才调用群机器人 Webhook；
- Webhook key 输出脱敏；
- 官方 `@wecom/aibot-node-sdk@1.0.7` 长连接 sidecar；
- sidecar 单实例锁、群/用户 allowlist、脱敏观测；
- chat/user allowlist 默认 fail-closed，只有显式 `WECOM_ALLOW_ALL=true` 才可全量放行；
- 仓库内置 `wecom +review-core`，仅监听回环地址并要求独立 Bearer Token；
- sidecar 只允许回环 HTTP(S) Core 地址，不向 Core 发送 GitLink Token 或企业微信 Bot Secret；
- 固定 HTTP Review Core 协议，不拼接或执行 shell；
- 写操作类指令在 sidecar 侧即被拒绝，不会转发给 Review Core；
- 未配置 Core 时只进入 observe-only smoke；
- P4 入站只接受 Review 只读命令。

## P5：多 Agent Review 协议与离线编排合同

新增命令：

```text
workflow +review-orchestrate
workflow +review-synthesize
workflow +review-warroom
```

`review-orchestrate`：

- 从 `review.context/v1` 生成 `review.agent-plan/v1`；
- 支持当前和上一个 Context 的增量文件范围；
- 按 correctness、tests、security、contributor_experience、integration 分工；
- 每个任务固定 read-only；
- 并发上限由计划显式给出；
- 合并、批准、拒绝、评论等能力在能力矩阵中保持关闭。

`review-synthesize`：

- 只接收同一 run、同一 head、同一任务的 assessment；
- 只计入 `status=completed` 且 `completed_at` 合法的 assessment；
- Finding 必须具有有效 severity、summary、evidence 和 confidence；
- 拒绝 stale、重复和角色不匹配结果；
- Finding 按严重度排序；
- 缺少 Agent 结果或存在冲突时保持 incomplete；
- 永远输出 `human_decision_required`，不替 Owner 给出批准或合并决定。

`review-warroom`：

- 聚合多个仓库的 Review Context；
- 可通过 `--collaboration` 叠加 P2 canonical WorkItem，保留负责人、协作状态和截止日期；
- 保留每个 PR 的 head、fingerprint、完整性、状态和下一步；
- partial 单独计数；
- 只生成 Owner 工作台事实，不执行写入。

当前 P5 不会自行启动模型或 Agent。它实现的是分工计划、输入输出合同、结果校验和汇总；
实际 Agent 执行仍由外部 Agent Host 完成。

生产门禁：

- GitHub Actions 使用 `go.mod` 指定的 Go 版本；
- CI 同时运行 Go 全仓测试和企业微信 Node 合同测试；
- `scripts/verify-round2-p5.ps1` 可复现本轮格式、定向测试、全仓构建和本轮 vet；
- 飞书和企微长连接均实行单实例；
- 所有平台观测都不得输出 Secret、Token、Webhook key 或原始会话 ID。

## 命令示例

```powershell
go run . workflow +review-orchestrate `
  --from .\review-context-current.json `
  --previous .\review-context-previous.json `
  --max-agents 3

go run . workflow +review-warroom `
  --from .\repo-one-pr-12.json,.\repo-two-pr-31.json `
  --collaboration .\repo-one-work-item.json,.\repo-two-work-item.json

go run . wecom +review-notify `
  --from .\review-work-item.json
```

企业微信长连接与 Review Core 分两个进程启动，并使用同一个本地桥接 Token：

```powershell
$env:GITLINK_REVIEW_CORE_TOKEN = "<random-local-token>"
go run . wecom +review-core --repository Gitlink/gitlink-cli

Set-Location .\bridges\wecom
$env:WECOM_BOT_ID = "<bot-id>"
$env:WECOM_BOT_SECRET = "<bot-secret>"
$env:GITLINK_REVIEW_CORE_URL = "http://127.0.0.1:8765/v1/review/inbound"
$env:GITLINK_REVIEW_CORE_TOKEN = "<same-random-local-token>"
npm start
```

飞书真实资源同步必须显式增加：

```powershell
go run . feishu +review-gateway `
  --bindings .\bindings.json `
  --state-db .\review-gateway.db `
  --sync-feishu-resources `
  --base-app-token $env:FEISHU_BASE_APP_TOKEN `
  --review-table-id $env:FEISHU_REVIEW_TABLE_ID `
  --review-document-id $env:FEISHU_REVIEW_DOCUMENT_ID `
  --sync-feishu-task
```

企业微信真实发送必须显式增加：

```powershell
--send --webhook-url $env:WECOM_WEBHOOK_URL
```

GitLink `common` Review 写回必须同时满足：

```text
账号绑定
ActionPlan 未过期
同一飞书账号确认
head 未变化
完整性 complete
GitLink Token 身份匹配
Gateway 启动显式 --enable-gitlink-review-write
```

## 本地验收

运行：

```powershell
.\scripts\verify-round2-p5.ps1
```

全仓历史基线审计使用：

```powershell
.\scripts\verify-round2-p5.ps1 -Full
```

截至 2026-07-31，`-Full` 仍会复现 P0 已记录的历史失败，包括缺失旧命令注册、旧测试辅助
函数缺失、i18n 文案漂移和 API 旗标漂移。本轮不以机械修改无关历史模块掩盖该基线；无
`-Full` 的 P2–P5 门禁只验证本轮包并要求全仓生产代码能够构建。

代码门禁通过不等同于全部真实平台验收。飞书正常路径已通过，仍应补强其脱敏证据并完成：

- 飞书同一 `message_id` 去重、执行中重启和 reply 恢复故障演练；
- Base、Doc 和 Task 首次写入、重复执行和远端成功/本地失败对账；
- 企业微信 req_id、脱敏 observation、最终流式回复或模板卡片回执；
- GitLink 测试 PR 的 write-before/write-after Review 列表和 Review ID；
- 全程使用的 commit SHA、配置 revision 和脱敏证据。

## 官方开发资料

- [飞书 Channel SDK](https://open.feishu.cn/document/mcp_open_tools/integrating-agents-with-feishu/integrate-feishu-channel)
- [飞书长连接事件订阅](https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN)
- [飞书多维表格记录检索](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/bitable-v1/app-table-record/search)
- [飞书 DocX 数据结构](https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/docx-structure)
- [飞书 Task v2](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/task-v2/overview)
- [企业微信消息推送](https://developer.work.weixin.qq.com/document/path/91770)
- [企业微信智能机器人长连接](https://developer.work.weixin.qq.com/document/path/101463)
- [企业微信智能机器人 Node SDK](https://www.npmjs.com/package/@wecom/aibot-node-sdk)
