# Round 2：参考 GitHub 聊天集成的数据作用域收口与后续任务

日期：2026-08-04  
分支：`feat/round2-platform-v2-m1-pr-result-card`  
基线：`bd80c549ed3a62f946d866612b7cf2163283911c`

## 1. 本轮结果

本轮把评估文档中的第一优先级问题落实为一个可验收任务：

> 将 GitLink PR 事实与飞书群内人工协作状态拆开，按安装实例和群聊隔离认领、截止时间、卡片及 Base/Doc/Task 资源映射，并移除卡片中写死的负责人占位文案。

完成后的身份模型为：

```text
GitLink PR snapshot
installation_id + repository + pr_number

Feishu collaboration state
installation_id + chat_id + repository + pr_number

Feishu resource mapping
installation_id + chat_id + repository + pr_number + resource_type
```

这保证：

- 同一安装中的不同群可以查看同一份 GitLink PR 事实；
- 不同群可以独立认领同一 PR、设置截止时间和维护协作资源；
- 不同 GitLink installation 即使仓库名相同，也不会共享快照或人工状态；
- 每次 PR 查询仍必须显式写出 `owner/repo`，不存在默认仓库；
- 每次查询都在当前消息下回复完整卡片，不以固定卡片更新替代当前查询结果；
- 公开未绑定仓库只提供 GitLink GET-only 查询，不创建群级协作状态或飞书资源。

## 2. 从 GitHub/Lark 成熟集成采用的模式

GitHub for Slack 的官方使用方式将“应用安装”“用户账号连接”“频道订阅仓库”和“事件通知”拆成独立能力。频道使用显式的 `owner/repo` 订阅命令；用户连接账号后才获得身份相关 mention 和写操作能力。GitHub 官方还建议 Webhook 使用 Secret 验签、delivery ID 去重、先快速返回再异步处理。

Lark 应用目录中的 GitHub Assistant 则证明了“仓库事件 → 群机器人卡片 → 跳转代码平台”的产品路径；飞书开放平台支持应用机器人、长连接事件和卡片回调，可以在此基础上增加群内协作，而不把 GitLink 事实搬成飞书自己的事实。

本项目采用以下共同模式：

| 成熟集成模式 | GitLink 飞书实现 |
| --- | --- |
| 应用安装与频道订阅分离 | `GitLinkInstallation` 与 `ReviewChatBinding` 分离 |
| 频道显式选择 `owner/repo` | 所有 PR/Queue 命令必须携带 `owner/repo` |
| 用户身份连接只服务于身份动作 | 公开查询无需账号绑定；受控 GitLink 写回必须绑定并再次核验 |
| Webhook 验签、delivery 去重、异步处理 | HMAC-SHA256、SQLite 去重、持久 Job 与租约 |
| PR/Issue 通知维持线程上下文 | 当前消息回复完整卡片；另保留群级长期资源映射 |
| 平台卡片是协作面，不是代码事实源 | GitLink Context 为事实源，卡片/Base/Doc/Task 为派生资源 |

没有照搬以下设计：

- 不把某个仓库设为默认仓库；所有仓库平等，避免跨仓误操作；
- 不允许 `*` 仓库通配符进入受信任安装范围；
- 不把“群里能看到机器人”视为有权操作任意 GitLink 仓库；
- 不要求公开只读查询先绑定飞书和 GitLink 账号；
- 不把固定卡片的 PATCH 当成当前消息的最终回复；
- 不在卡片中显示伪造的“已认领（飞书成员）”。

## 3. 已完成的代码结构

### 3.1 PR 快照表

新增 `review_pr_snapshots`：

```text
snapshot_key
installation_id
repository
pr_number
review_stage
decision
collection_status
head_sha
source_fingerprint
gitlink_state
archived
updated_at
```

唯一范围为：

```text
installation_id + repository + pr_number
```

该表只保存 GitLink 权威事实，不保存飞书负责人和截止时间。`partial` 结果仍不能覆盖已有完整快照。

### 3.2 群级协作状态表

新增 `review_collaboration_states`：

```text
collaboration_key
installation_id
chat_id
repository
pr_number
assigned_to
assigned_display_name
collaboration_status
due_at
updated_by
updated_at
```

唯一范围为：

```text
installation_id + chat_id + repository + pr_number
```

认领、释放和截止日期修改只影响当前群。过去日期会被拒绝；归档 PR 不再接受协作动作。

### 3.3 资源映射

`ReviewCollaborationBundle.UniqueKey` 不再只由 `repository + pr_number` 计算，而是由完整的群级 `collaboration_key` 派生。因此以下远端资源均按群隔离：

- 飞书长期卡片；
- Base 记录；
- Doc 快照；
- Task。

同一次查询仍会在当前消息下发送完整结果卡片。长期资源映射只负责后续事件刷新、幂等和生命周期维护。

### 3.4 负责人展示

认领时数据库保存真实飞书 OpenID。卡片现在输出飞书 Markdown mention：

```text
<at id=OPEN_ID></at>
```

飞书客户端负责把它解析为租户内可见的成员名称。若以后通过成员接口取得显示名，则优先使用 `assigned_display_name`；纯文本、Doc 和日志只显示名称或不可逆短哈希，不输出完整 OpenID。

### 3.5 旧数据迁移

旧表 `review_collaboration_items` 暂时保留为迁移输入：

1. 启动时只迁移能由 `chat_repository_bindings` 唯一确定 installation 的记录；
2. 无法唯一确定 installation 的旧记录不迁移，防止把负责人泄漏到错误安装；
3. 当前 Job 只在同样满足唯一 installation 时执行惰性迁移；
4. 旧的卡片/Base/Doc/Task `remote_id` 和内容指纹一并复制到新作用域键；
5. `INSERT OR IGNORE` 保证迁移可重复运行，并优先保留已经存在的新映射；
6. 旧记录不立即删除，便于回滚和人工对账。

## 4. 本轮自动化验收

新增或强化的测试覆盖：

- 同一安装、同一仓库、不同群的负责人互不继承；
- 不同安装、同名仓库的快照和负责人互不继承；
- 群级 WorkItem/卡片/Base/Doc/Task 键不会碰撞；
- 唯一 installation 的旧记录可幂等迁移；
- 旧卡片资源 ID 可迁移到新作用域键，避免重新创建重复卡片；
- installation 不明确时，旧负责人不会进入任一新作用域；
- 卡片中使用可解析成员 mention，不再出现写死占位文案；
- complete、partial、failed 卡片 golden 继续成立；
- 飞书模块定向测试通过。

本轮完整专项门禁已经通过，包括飞书、企业微信、Review Core、全仓生产构建和专项 vet。代码提交
`f6f3d374059848bbf1fc57bfc787a66d5dfeb72e` 对应的
[GitHub Actions run 30832374540](https://github.com/whzy3185/gitlink-feishu/actions/runs/30832374540)
状态为 `success`。

本轮没有执行 GitLink POST；服务激活时只发生本地 SQLite 迁移。部署后仍需要用同一 PR 在两个群分别认领，保存卡片截图和数据库作用域证据，才能关闭真实平台门禁。Base、Doc、Task 的旧 remote ID 已完成本地映射迁移，但尚未在本轮主动触发新的远端写入。

## 5. 后续实施任务

以下任务按照评估文档的优先级排序。它们建立在本轮作用域修复之上，但不在本提交中虚构为已完成。

### M2：仓库订阅模型

新增显式 `review_chat_subscriptions`，把“允许访问仓库”和“希望接收哪些事件”分开：

```text
installation_id + chat_id + repository
events: pulls, reviews, comments, commits, workflows
filters: branch, label, author
enabled
revision
```

命令始终显式携带仓库：

```text
订阅 Gitlink/forgeplus pulls reviews
取消订阅 Gitlink/forgeplus reviews
查看 Gitlink/forgeplus 订阅
```

只有群管理员可以修改订阅。第一版只开放 `pulls` 和 `reviews`，其他事件等 GitLink 真实 payload 合同稳定后再启用。

### M3：标准事件与 Webhook 收口

把 GitLink/Gitea 风格 payload 归一化为 `gitlink.review-event/v1`，最少包含：

```text
delivery_id
installation_id
repository
event_type
action
pr_number
head_sha
actor
occurred_at
```

现有 Webhook 入口继续坚持：原始 body 验签、恒定时间比较、最大 1 MiB、delivery 持久去重、事件 allowlist、快速 2xx、异步 Job。新增订阅过滤后，只向实际订阅该事件的群创建刷新任务。

### M4：资源 Outbox

远端飞书写入从 Job 执行器中拆成持久 Outbox：

```text
outbox_id
installation_id
chat_id
repository
pr_number
resource_type
operation
payload_fingerprint
status
attempt_count
next_attempt_at
lease_owner
lease_expires_at
remote_id
last_error_class
```

Base、Doc、Task、长期卡片分别执行；某个资源失败不阻断其他资源。远端结果不确定时进入 `unknown_needs_reconciliation`，禁止盲目重试。

### M5：服务化和可观测性

在保持单体部署的前提下增加：

- `/healthz`：进程存活；
- `/readyz`：SQLite、飞书长连接、Job/Outbox worker 状态；
- 按事件类型统计 handler latency、Job latency、重试和死信；
- 配置 revision、安装 ID、群 ID 只输出短哈希；
- 单实例启动锁；
- 优雅停机时停止接收、归还租约并等待有界中的任务。

多实例在引入共享数据库、分布式租约和飞书长连接拓扑验证前保持不支持。

## 6. 真实平台验收步骤

1. 启动新二进制，确认旧协作状态与卡片 remote ID 已迁移；
2. 在群 A 查询 `查看 Gitlink/forgeplus PR #356`，确认当前消息下出现完整卡片；
3. 在群 A 领取 PR，确认负责人显示真实飞书成员名；
4. 在群 B 查询同一 PR，确认仍是“未认领”；
5. 在群 B 由另一成员领取，确认群 A 的负责人不变化；
6. 分别刷新两群，确认两张长期卡片的 message ID 不同且各自稳定更新；
7. 重启服务后重复刷新，确认没有新增重复 Base/Doc/Task；
8. 全程核对 GitLink POST 次数为 0。

## 7. 参考资料

- [GitHub in Slack：账号连接、显式仓库订阅、线程和 mention](https://docs.github.com/en/integrations/how-tos/slack/use-github-in-slack)
- [GitHub in Slack：通知事件和过滤器](https://docs.github.com/en/integrations/how-tos/slack/customize-notifications)
- [GitHub Webhook 最佳实践](https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks)
- [GitHub Webhook 签名验证](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)
- [Lark App Directory：GitHub Assistant](https://app.larksuite.com/app/cli_9c4b6daaa4bad106)
- [Lark 开放平台：智能体应用的长连接事件与卡片回调](https://open.larksuite.com/document/mcp_open_tools/integrating-agents-with-feishu/overview)

## 8. 准确结论

本轮已经解决的是最底层的数据污染风险和负责人假展示问题。它让后续仓库订阅、Webhook 主动刷新和资源 Outbox 有了可靠的作用域基础。

不能因为代码测试通过就声称两个群的真实平台隔离已经验收；也不能把尚未实现的订阅表、Outbox 和多实例部署写成当前能力。当前可准确表述为：

> Installation/Chat/Repository/PR 作用域隔离、旧资源映射迁移和飞书负责人真实 mention 已完成代码实现，专项 CI 已通过；待双群真实 smoke 通过后关闭本阶段平台门禁。
