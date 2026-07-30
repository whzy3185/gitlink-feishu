# 复赛 PR Review 协作 P2 实施记录

日期：2026-07-30

基线提交：

```text
2ae67e56cb8f45a25774d60e734afb19126a4f75
fix: close P1 review collaboration gates
```

开发分支：

```text
feat/round2-review-collaboration-p2
```

## 1. 当前结论

本次完成的是 **P2.0 飞书只读入站与任务基础设施**，不是完整 P2 退出。

已经形成的最小闭环是：

```text
飞书消息或卡片动作
-> 官方 Channel SDK 归一化
-> 群聊与仓库绑定检查
-> 用户 allowlist / 管理员检查
-> message_id / event_id 持久化去重
-> 3 秒处理窗口内保存并排队
-> 异步执行 GitLink GET-only Review Queue 或 Review Context
-> 生成本地 preview 结果或 Review 草稿模板
```

当前明确不做：

```text
不从飞书创建 GitLink Review
不批准或拒绝 PR
不创建、回复、解决或重开行级评论
不请求或移除 Reviewer
不合并 PR
不保存 GitLink 用户 Token
不写飞书 Base、Doc、Card 或 Task
不真正修改群仓库绑定和 Reviewer 认领记录
```

所以当前分支可以用于离线演示、测试群长连接验证和后续 P2 同步层开发，但不能宣称“飞书协作闭环已经全部完成”。

## 2. 为什么这一层先做入站和只读

比赛结束时拥有者面对大量 PR，最需要先解决的是：

1. 在团队共同使用的飞书群中快速发起 Review 工作；
2. 获取稳定、最新、可审计的 GitLink 事实；
3. 避免重复事件产生重复任务；
4. 避免 partial 快照覆盖此前完整事实；
5. 把耗时读取和分析移出飞书事件处理窗口；
6. 在身份、权限和写回合同稳定前，不放大 GitLink 写入风险。

这一设计保持了项目定位：

```text
GitLink 是代码协作事实源
gitlink-cli 是 Agent 和自动化的执行面
Agent 负责取证、分诊和草拟
飞书负责团队沟通、领取、进度和审计
拥有者保留最终 Review 与合并决定
```

## 3. 新增能力

### 3.1 新命令

```text
gitlink-cli feishu +review-gateway
```

离线预览：

```powershell
go run . feishu +review-gateway `
  --from-event shortcuts/feishu/testdata/review_gateway_event.json `
  --bindings shortcuts/feishu/testdata/review_gateway_bindings.json `
  --execute-read-only `
  --format json
```

长连接监听：

```powershell
$env:FEISHU_APP_ID="cli_REDACTED"
$env:FEISHU_APP_SECRET="REDACTED"

go run . feishu +review-gateway `
  --listen `
  --bindings .local/review-gateway-bindings.json `
  --state-db .local/review-gateway.db `
  --admin-users "ou_admin_1,ou_admin_2" `
  --format json
```

离线模式不带 `--execute-read-only` 时只生成 Job 计划；带该参数时会继续执行
同一条 GitLink GET-only 工作并输出摘要结果。P2.0 监听模式对已接受事件自动异步
执行 GET-only Job，但不会向飞书发送消息。真实回复、任务恢复和结果持久化已在
P2.1 单独实现和记录：

```text
docs/ROUND2_PR_REVIEW_COLLABORATION_P21_IMPLEMENTATION.md
```

### 3.2 群与仓库绑定

绑定文件合同：

```json
{
  "schema_version": "feishu.review-bindings/v1",
  "bindings": [
    {
      "chat_id": "oc_round2_review",
      "repository": "Gitlink/gitlink-cli",
      "enabled": true,
      "admin_user_ids": ["ou_owner"],
      "allowed_user_ids": ["ou_owner", "ou_reviewer"],
      "deadline_hours": 24,
      "binding_revision": "config-v1"
    }
  ]
}
```

安全规则：

- 长连接只接受绑定文件中启用的群；
- 群消息必须 @ 机器人；
- 不响应 `@所有人`；
- P2.0 禁用私聊；
- 若配置 `allowed_user_ids`，仅白名单用户或管理员可发起工作；
- “绑定仓库”当前只生成变更计划，不改文件、不改数据库。

绑定文件应保存在 `.local` 或部署系统的受控配置中，不应允许普通群成员直接编辑。

### 3.3 支持的意图

| 飞书输入 | P2.0 行为 | GitLink 访问 |
| --- | --- | --- |
| `帮助` | 返回本地能力说明 | 无 |
| `查看绑定` | 展示当前受控绑定 | 无 |
| `查看待 Review` / `查看待审查` | 读取开放 PR 并生成优先队列 | GET |
| `查看 PR #42` | 读取 PR、files、versions、reviews、threads | GET |
| `刷新 PR #42` | 重新读取完整 Review Context | GET |
| `生成 PR #42 Review 草稿` | 基于当前 Context 生成确定性模板 | GET |
| `领取 PR #42` | 生成认领变更计划 | 无写入 |
| `查看我的 Review 任务` | 返回当前阶段边界说明 | 无 |
| `绑定仓库 owner/repo` | 仅管理员生成绑定变更计划 | 无写入 |

以下输入故意不识别：

```text
批准 PR
拒绝 PR
合并 PR
解决评论
请求 Reviewer
```

### 3.4 Review 草稿模板

草稿合同：

```text
review-draft/v1
```

模板包含：

- 仓库、PR 编号和标题；
- 当前 patchset；
- 文件、Review 和未解决线程计数；
- 每名 Reviewer 的当前有效决定；
- 最多 10 条当前未解决线程；
- Unknowns；
- 推荐下一步。

线程正文每条最多保留 400 个字符，防止单个回调结果失控。草稿是可复核的材料，不是自动批准结论。

## 4. 事件、任务和结果合同

### 4.1 入站事件

```text
feishu.review-gateway/v1
```

只保留：

```text
event_id
message_id
event_type
chat_id
chat_type
user_id
content
create_time_ms
```

卡片 callback token、tenant token、App Secret 和 GitLink Token 均不进入事件或任务模型。

### 4.2 异步任务

```text
feishu.review-job/v1
```

每个任务显式包含：

```text
mode = preview
mutates_gitlink = false
collaboration_mutation = true/false
requires_admin = true/false
```

### 4.3 执行结果

```text
feishu.review-result/v1
```

结果只输出协作所需摘要：

- collection status 和 partial；
- head SHA 和 source fingerprint；
- Review stage 和 decision；
- Review、线程与开放线程数量；
- 队列前三档数量和最多 10 条重点 PR；
- snapshot plan；
- 可选 Review 草稿。

完整原始 API 响应不会进入 SQLite Job 错误字段。

## 5. 去重与异步处理

### 5.1 去重键

消息事件优先使用：

```text
feishu:message:<message_id>
```

其他事件优先使用：

```text
feishu:event:<event_id>
```

两者均缺失时才使用事件稳定字段的 SHA-256 fallback。SQLite 默认保留 24 小时，进程重启后仍能拒绝重复事件。

之所以对消息优先使用 `message_id`，是因为飞书消息接收文档说明部分重复投递场景需要使用消息 ID 去重；其他 v2 事件使用 `event_id`。

### 5.2 处理窗口

飞书长连接事件处理需要在 3 秒内完成。P2.0 handler 只做：

```text
归一化
-> 校验
-> 去重预留
-> 保存 queued Job
-> 放入内存队列
```

GitLink 网络读取在独立 worker 中执行，默认单任务超时 60 秒。

### 5.3 SQLite 状态

默认路径：

```text
.local/review-gateway.db
```

存储：

- 去重键和过期时间；
- Job payload；
- queued/running/completed/failed；
- 脱敏后的错误摘要。

不存储：

- Feishu App Secret；
- callback token；
- tenant/user access token；
- GitLink Token；
- GitLink 原始错误响应正文。

## 6. partial 快照保护

同步计划采用以下顺序：

```text
merged / closed
-> archive

partial / incomplete
-> preserve_previous

fingerprint 与 head 未变化
-> unchanged

完整且发生变化
-> apply
```

所有计划均设置：

```text
preserve_manual_fields = true
```

因此后续 Base、Doc 和 Task 同步层必须把 GitLink 事实字段与飞书人工协作字段分开，不能用 partial 结果覆盖 Reviewer、截止时间、人工备注和负责人。

## 7. P2 第一个预备修正

### 7.1 错误脱敏

`review.context/v1` 的 `fetch_errors.message`、兼容 Notes 和 Gateway Job 错误现在会处理：

- URL userinfo、query 和 fragment；
- Authorization Bearer；
- Cookie；
- access/refresh token；
- App Secret、client secret 和 password；
- 当前进程已加载的相关 secret；
- 512 字符长度上限。

GitLink API 错误优先输出 HTTP 状态和错误码，不同步原始响应正文。

### 7.2 Reviewer 排序保守化

同一 Reviewer 的两条当前 Review：

```text
双方都有不同时间
-> 按时间排序

双方都没有时间且 ID 均为数字
-> 按数字 ID 排序

只有一方有时间、时间相同或其他顺序无法确定
-> unknown
```

这避免用数字 ID 猜测混合时间记录的真实先后。

## 8. 验证

自动测试覆盖：

- Reviewer 混合时间排序；
- Context 和 Gateway 错误脱敏；
- 中文命令解析；
- 群绑定与用户白名单；
- 管理员绑定计划；
- message ID 重复投递；
- 过期和未知事件；
- SQLite 重启后的持久去重；
- 异步 Job 状态；
- partial、merged、unchanged 和 apply 计划；
- SDK 消息与卡片动作归一化；
- callback token 不进入 Gateway Event；
- 有界 Review 草稿模板；
- Feishu 命令注册。

建议门禁：

```powershell
go test ./shortcuts/workflow ./shortcuts/feishu
go test ./shortcuts ./shortcuts/pr
go test ./...

go run . feishu +review-gateway `
  --from-event shortcuts/feishu/testdata/review_gateway_event.json `
  --bindings shortcuts/feishu/testdata/review_gateway_bindings.json `
  --execute-read-only `
  --format json
```

真实环境门禁：

```text
[ ] FEISHU_APP_ID / FEISHU_APP_SECRET 已加载
[ ] 自建应用已启用机器人能力
[ ] 事件订阅选择长连接
[ ] 已订阅 im.message.receive_v1
[ ] 机器人已加入测试群
[ ] binding chat_id 与测试群一致
[ ] 测试群 @机器人 “查看 PR #431”
[ ] 3 秒内收到 queued receipt
[ ] SQLite 只有一条对应 message_id
[ ] worker 只产生 GitLink GET
[ ] 重发同一消息不产生第二个 Job
```

### 8.1 本工作站实测

2026-07-30 已执行：

```text
PASS  go test ./shortcuts/workflow ./shortcuts/feishu ./shortcuts ./shortcuts/pr
PASS  go vet ./shortcuts/workflow ./shortcuts/feishu
PASS  git diff --check
PASS  feishu +review-gateway 离线事件 -> GitLink GET-only Job
```

公共 PR #431 的 GET-only 结果：

```text
repository: Gitlink/gitlink-cli
pull_request: 431
collection_status: complete
head_sha: 5b40a9088726134829e68ca656b85cfcaac1a8a2
review_stage: triaged
decision: pending
mutates_gitlink: false
snapshot_plan: apply
preserve_manual_fields: true
```

当前终端没有加载：

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_BASE_APP_TOKEN
FEISHU_PR_TABLE_ID
FEISHU_WEBHOOK_URL
```

因此没有伪造长连接成功，也没有执行任何飞书写入。`--listen` 已验证会在
缺少 App ID/Secret 时立即返回非零退出码。

`go test ./...` 仍未通过，失败集中在本分支未修改的历史包和测试合同，例如
`internal/client`、`shortcuts/issue`、`shortcuts/milestone`、`cmd/alias`、
`shortcuts/release`。P2 目标包、命令注册和 PR 包均通过；全仓历史基线失败
不能写成 P2 成功，也不在本提交中顺带扩张修复范围。

## 9. 仍未关闭的 P2 门禁

P2.0 后仍需完成：

P2.1 已开始关闭这里的真实收发和可靠性前置门禁；协作状态与资源同步仍按以下阶段
继续推进。

### P2.1 协作状态

- 仓库绑定的受控持久化和审计；
- 飞书账号与 GitLink identity reference；
- Reviewer 认领、截止时间、释放和冲突处理；
- “我的 Review 任务”真实查询；
- 人工字段与 GitLink 事实字段分离。

### P2.2 飞书资源同步

- Review Queue 卡片；
- PR WorkItem 多维表格；
- Review Doc；
- Feishu Task；
- merged/closed 归档；
- partial 快照保护的真实 upsert；
- 同步失败重试和审计。

所有飞书写入仍应先支持 preview，只有显式启用测试环境写入后才执行。

### P2.3 测试群验收

- 长连接稳定性；
- 3 秒处理窗口；
- SDK 与 Gateway 双层去重；
- 多实例随机投递下的 SQLite 或共享状态；
- 429、超时和连接恢复；
- 最小权限；
- 脱敏日志复核。

完成以上三项后，才能按设计文档的 P2 退出条件宣称“测试群可从飞书发起只读 Review 工作并稳定同步协作字段”。

## 10. 飞书官方依据

本实现参考：

- Channel SDK 集成 Agent：<https://open.feishu.cn/document/mcp_open_tools/integrating-agents-with-feishu/integrate-feishu-channel>
- 长连接事件订阅配置：<https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case?lang=zh-CN>
- 事件订阅概览：<https://open.feishu.cn/document/server-docs/event-subscription-guide/overview?lang=zh-CN>
- 接收消息事件：<https://open.feishu.cn/document/server-docs/im-v1/message/events/receive?lang=zh-CN>
- Go SDK 包文档：<https://pkg.go.dev/github.com/larksuite/oapi-sdk-go/v3>

当前锁定：

```text
github.com/larksuite/oapi-sdk-go/v3 v3.9.9
```

部署时还要遵守官方长连接限制：自建应用、事件 handler 及时返回、单应用连接数限制，以及多实例时事件随机投递而非广播。多实例生产部署不能依赖单进程内存去重。
