# 复赛 PR Review 协作剩余实施计划

日期：2026-08-02

基线分支：`feat/round2-feishu-platform-v2`

代码基线：`edcaf609b5c8be3a67943ce1f7bcb543861af71e`

计划修订：`b12c664113f46a1096e3bc7d33e1c876d4bcad3e`

配套差距清单：`docs/ROUND2_REMAINING_GAPS_20260802.md`

## 1. 实施目标

本计划不继续无边界扩功能，而是把现有能力收敛成可演示、可审计、可回滚的 Owner Review 工作台：

```text
公开 PR 快速查看
-> 协作仓库进入 Review Queue
-> 飞书卡片展示证据和下一步
-> Reviewer 认领与截止时间
-> Agent 生成有证据的评估
-> Owner 预览并确认 common Review
-> GitLink 保存正式 Review
```

Owner 是最终决策者；Agent 是证据生产者；飞书是协作和控制面；GitLink 是 PR 与正式 Review 的状态真源。

## 2. 实施原则

1. 先完成比赛主链路，再扩展企业微信和高级自动化；
2. 只读默认，写入必须显式开启；
3. 公开仓库查看不要求仓库绑定，但不创建协作资源；
4. Base、Doc、Task 只投影已绑定协作仓库；
5. 所有写操作绑定用户、Installation、原始群、仓库、PR、head 和 fingerprint；
6. 网络不确定写入不得自动重试；
7. 平台能力没有真实证据时只标记“代码完成”；
8. 每一阶段单独提交、单独 CI、单独回滚；
9. 不把 approve、reject、merge、行级评论纳入本轮；
10. 不在仓库、日志、卡片或文档保存明文 Token。

## 3. 优先级和里程碑

| 里程碑 | 目标 | 比赛优先级 | 退出条件 |
|---|---|---:|---|
| M0.1 | 写入范围、写后对账与当前 SHA CI | P0 | 跨 Installation/群 POST=0，严格回读，专项 CI 绿色 |
| M1 | 完善 PR 结果卡片 | P0 | 群内一眼看懂 PR 和下一步 |
| M2 | 卡片协作动作与幂等 | P0 | 领取、刷新、截止时间真实可用 |
| M3 | Base 看板、Doc、Task 闭环 | P0 | Owner 团队能共享进度与审计 |
| M4a | Gateway 常驻、健康检查、备份和回滚 | P0 | 关闭终端与重启机器后仍可靠运行 |
| M4b | 身份绑定和一次受控 common Review | P1，可选展示 | 孔明职配真实写入一次并完成对账 |
| M5 | 单 Agent 真实评估 | P1，可选展示 | 一份同 head、有证据 Assessment 进入卡片与 Doc |
| M6 | 公网 Webhook | P2 | 具备时间窗的真实事件自动刷新 |
| M7 | 企业微信真实适配 | P2 | 一条企业微信只读链路跑通 |
| M8 | OAuth、多 Agent、生产 HA | 赛后 | 不阻塞复赛演示 |

## 4. M0.1：安全收口与冻结基线

### 4.1 工作内容

- 保存当前 Git SHA、Go 版本和构建命令；
- ActionPlan 保存 Installation 和原始群，并将两者纳入幂等键；
- 确认前校验同 Installation、同群、Installation allowlist 和原始群仓库绑定；
- POST 成功后新增 Review read-back matcher 和 unknown/verified 状态处理；
- 为当前精确 SHA 运行独立 Round 2 专项 CI；
- 导出不含 ID 和密钥的 v2 配置摘要；
- 记录 Gateway 进程、长连接和 GitLink 写入关闭状态；
- 保存 #431 和 #356 的 Job 摘要；
- 保存 Base、Doc、Task 资源 ID 的哈希；
- 建立比赛证据目录索引；
- 标记历史文档中的过时状态，避免“真实写入待验收”等旧描述干扰。

### 4.2 涉及位置

- `docs/ROUND2_CURRENT_STATUS_AND_OPERATOR_ACTIONS_20260801.md`
- `docs/ROUND2_PLATFORM_EVIDENCE_CHECKLIST.md`
- `scripts/verify-round2-p5.ps1`
- `.local/` 中受忽略的运行证据

### 4.3 验收

```text
git status clean
Round 2 Review Collaboration gate success on the exact SHA
Gateway online
GitLink write flag false
resource ids not present in process args
```

## 5. M1：完善 PR 结果卡片

当前代码状态（2026-08-02）：有界展示合同、complete/partial/failed 固定卡片、公开只读边界、短 SHA、OpenID 隐藏、长度预算和文本降级已实现；本地与 CI 结果以 `docs/ROUND2_M1_PR_RESULT_CARD_IMPLEMENTATION_20260802.md` 为准。真实 #356 卡片截图仍是退出条件。

### 5.1 用户结果

Owner 在群里查看 PR 后，应看到：

```text
仓库 / PR / 标题
作者、base <- head
open / merged / closed
文件数、增删行、patchset
Review 决定、Reviewer 摘要
未解决线程数量
collection_status / partial
风险、unknowns、建议下一步
负责人和截止日期
GitLink / Doc / Base / Task 链接
GitLink 写入状态
```

### 5.2 代码结构

- 扩充 `ReviewGatewayExecutionResult` 的展示字段；
- 在 `review_gateway_executor.go` 从 canonical Review Context 填充字段；
- 在 `review_gateway_reply.go` 增加简洁文本降级；
- 在卡片构造层生成固定 schema；
- 保留 `collection_status != complete` 的醒目告警；
- 公开未绑定仓库卡片不得包含 Base、Doc、Task 操作按钮。

### 5.3 测试

- complete、partial、failed 三种卡片 golden；
- open、merged、closed；
- 0 Review、多人 Review、同一人多次 Review；
- 长标题和大 unknowns 截断；
- 卡片长度超限时安全降级为文本；
- 不显示完整 SHA 以外的敏感内容。

### 5.4 真实验收

在 #356 上发送一次新消息，截图保存卡片。要求无需打开 Doc 即可理解 PR 当前状态和下一步。

## 6. M2：卡片协作动作与幂等

### 6.1 第一批动作

```text
领取 Review
释放 Review
刷新 PR
设置截止日期
打开 GitLink
打开 Review Doc
```

不加入：

```text
approve
reject
merge
提交 Review
```

### 6.2 执行结构

```text
card action
-> 立即 ACK
-> message/action id 幂等
-> SQLite Job
-> 权限和绑定校验
-> collaboration action
-> 更新 Base / Doc / Task
-> PATCH 原卡片
```

### 6.3 代码位置

- `review_gateway_models.go`：动作合同；
- `review_gateway_command.go`：Channel SDK 回调；
- `review_gateway_store.go`：Job、action 和回复状态；
- `review_collaboration.go`：协作状态；
- `review_collaboration_publisher.go`：Base、Doc、Task；
- `review_gateway_reply.go`：动作结果。

### 6.4 门禁

- 只有群成员或 allowlist 用户可操作；
- 绑定管理员才可改变仓库配置；
- PR 必须属于绑定仓库；
- 重复点击不产生第二次变更；
- ACK 预算小于 2 秒；
- Publisher 失败不能回滚已确认的本地协作状态，但必须显示 warning 和可重试状态；
- GitLink POST 次数为 0。

### 6.5 验收证据

- action payload 脱敏摘要；
- handler latency；
- 单条 collaboration audit；
- Base 同一记录；
- Task 同一 remote ID；
- 原卡片 message ID 不变；
- 重复点击后的无重复证据。

## 7. M3：Base、Doc、Task 协作闭环

### 7.1 Base 视图

创建：

1. 全部 Review Queue；
2. 按负责人分组看板；
3. 按截止日期的甘特图；
4. 待贡献者修改；
5. 等待 Owner 决策；
6. merged/closed 归档；
7. partial/failed 告警。

### 7.2 权限

- 应用身份负责 GitLink 事实字段；
- Owner 可以修改负责人、截止日期、人工下一步；
- 普通 Reviewer 只能修改允许的协作字段；
- 不允许普通编辑者修改 repository、PR number、head、fingerprint；
- Base 不是配置真源，SQLite 继续作为运行时真源。

### 7.3 幂等和对账

- `pr_key` 是业务唯一标识，不宣称为 Base 唯一索引；
- SQLite 保存 record ID 和内容指纹；
- 待实现 Publisher 自身的远端创建租约或 SQLite 临界区；当前单实例 Gateway 和串行 Job 只降低冲突概率，不构成独立的单写者保证；
- 重复同步只更新同一记录；
- 人工字段不被 GitLink 刷新覆盖；
- 删除远端资源时记录 missing 并进入显式恢复流程。

### 7.4 Task 生命周期

- 领取后设置 assignee；
- 设置全天 due；
- Owner 可作为 follower；
- merged/closed 后完成 Task；
- Task 原生提醒与机器人提醒只选择一种，避免重复。

### 7.5 验收

对 #356 重复执行两次：

```text
Base business rows = 1
Doc creates = 1
Task creates = 1
第二次 action = unchanged 或 updated
人工负责人和截止日期被保留
```

## 8. M4b：身份绑定与一次受控 common Review 写回

### 8.1 测试前置

由用户完成：

- 在孔明职配创建只改测试文档的开放 PR；
- 确认测试 GitLink 用户拥有 common Review 权限；
- 提供本地环境变量形式的测试 Token；
- 确认允许进行一次真实 common Review。

实现侧不得把 Token 写入仓库、命令参数、飞书或文档。

### 8.2 第一阶段身份绑定

比赛阶段先采用管理员核验映射：

```text
installation_id
feishu open_id hash
gitlink login
verification_method=owner_admin_verified
verified_at
enabled
```

完整 OAuth 放到 M8，不阻塞一次安全验收。

### 8.3 写入流程

```text
查看绑定仓库 PR
-> complete 且 partial=false
-> 生成 common Review 草稿
-> 创建 ActionPlan
-> 卡片展示 actor、head、fingerprint、内容和过期时间
-> 用户二次确认 plan_id
-> 校验原始 Installation、原始群和两级仓库范围
-> 再次 GET PR 和 patchset
-> 校验 actor、repository、PR、head、fingerprint
-> 获取单次 lease
-> POST common Review
-> 保存 Review ID
-> 再次 GET Review
-> read-back matcher 校验 ID、common、head、内容指纹和可用 actor
-> 匹配时 verified；缺失、失败或不匹配时 unknown_needs_reconciliation
-> 关闭写入开关
```

### 8.4 代码位置

- `review_action_plan.go`
- `review_write.go`
- `review_runtime.go`
- `review_installation.go`
- `review_gateway_store.go`
- `review_gateway_reply.go`

必须新增并保持独立测试的代码合同：

- Review read-back matcher；
- reconciliation result；
- GET 失败、Review 缺失和字段不匹配的 unknown handling；
- POST 已确认后的所有 unknown 路径禁止自动重试。

### 8.5 停止条件

以下任一发生，POST 必须为 0：

- Context 非 complete；
- partial=true；
- head 变化；
- fingerprint 变化；
- actor 未绑定；
- 确认 Job 与 ActionPlan 的 Installation 不一致；
- 确认 Job 与 ActionPlan 的原始群不一致；
- Installation 不是 write；
- 仓库不在 allowlist；
- 启动开关未开启；
- ActionPlan 过期或已使用；
- Review 状态不是 common。

POST 网络结果不确定时：

```text
mutation_status=possible
status=unknown_needs_reconciliation
禁止自动重试
```

### 8.6 验收证据

- before Review JSON 脱敏摘要；
- ActionPlan；
- 确认消息；
- POST count=1；
- Review ID；
- after Review；
- reconciliation=verified，且来源为 POST 后的 GitLink GET；
- 写入开关关闭；
- Token 从环境移除。

## 9. M5：单 Agent 真实评估

### 9.1 范围

先接一个只读代码 Review Agent，不立即做多 Agent Warroom。当前 Invocation 尚未携带 canonical Review Context、diff 或文件证据，因此 M5 不能直接进入真实验收。必须先在 Invocation 内提供受限证据，或提供能证明同 head 的独立只读获取合同，然后才能输出结构化 Assessment：

```text
summary
findings
severity
evidence
unknowns
confidence
coverage
evidence_checked
```

### 9.2 安全

- 仓库内容视为不可信输入；
- Invocation 标记 untrusted content 并限制最大输入大小；
- Provider 必须读取同一 head 的 patch/文件证据，并回传 SourceFingerprint；
- 不允许 PR 文本改变系统指令或工具权限；
- Agent 无 GitLink 写工具；
- Agent 输出不能直接成为 Review 写入；
- 每条 finding 必须指向文件、diff 或线程证据；
- 空 Assessment 不能推动状态；
- 超时和无效输出降级为人工 Review。

### 9.3 代码位置

- `shortcuts/workflow/review_agent_runner.go`
- `shortcuts/workflow/review_agent_protocol.go` 或现有协议文件
- `review_gateway_executor.go`
- `review_gateway_command.go`

### 9.4 验收

- #356 或孔明职配测试 PR；
- 一次真实 provider invocation；
- 输入 head 与输出 head 一致；
- Assessment status=completed；
- 至少一项 coverage 或 evidence_checked；
- 结果进入 Doc 和卡片；
- GitLink 写入 0；
- Owner 明确选择采用、修改或忽略。

## 10. M4a 常驻部署与 M6 公网 Webhook

### 10.1 M6 公网 Webhook

- 配置 HTTPS 入口；
- 校验签名、时间窗、body size；
- 拒绝过期事件，并定义代理后的时间戳与签名版本合同；
- delivery 幂等；
- 只接受绑定仓库；
- PR update 自动刷新；
- merged/closed 自动归档；
- 新 patchset 使旧 Agent/ActionPlan stale。

### 10.2 M4a 常驻部署

优先选择一种：

```text
Windows Service
Windows 计划任务
Linux systemd
容器 + restart policy
```

比赛机器至少实现：

- 开机自启；
- 自动重启；
- 单实例锁；
- 健康检查；
- 日志轮转；
- SQLite 定期备份；
- 配置和凭据与代码分离；
- 一条回滚命令。

### 10.3 验收

```text
关闭交互终端后 Gateway 仍在线
重启机器后自动恢复
旧 queued Job 被恢复
重复事件不重复回复
健康检查显示长连接和最近 GitLink GET 状态
```

## 11. M7：企业微信真实适配

### 11.1 最小范围

只做：

```text
企业微信消息
-> Sidecar
-> loopback Review Core
-> GitLink GET-only
-> 流式回复
```

不做 GitLink 写回，不同步飞书资源。

### 11.2 验收

- 真实企业微信应用；
- 群或用户 allowlist；
- 一个绑定仓库 PR；
- 一个公开未绑定 PR；
- 重复事件只执行一次；
- 写类命令拒绝；
- 重启恢复；
- GitLink 写入 0。

## 12. M8：赛后演进

比赛后再开展：

- GitLink OAuth 或 App Installation；
- 短期 Installation token；
- 用户自助绑定和撤销；
- 多 Agent 专家分工；
- Base 配置管理界面；
- 多实例共享数据库与队列；
- 指标、告警和 SLO；
- approve/reject 是否开放的独立安全评审。

## 13. 分支和提交建议

每个里程碑从最新通过 CI 的提交创建短分支：

```text
feat/round2-review-card
feat/round2-card-actions
feat/round2-collaboration-views
fix/round2-platform-v2-m0-write-scope
feat/round2-common-review-live
feat/round2-agent-live
feat/round2-webhook-deployment
feat/round2-wecom-live
```

每个分支保持：

- 一个目标；
- 小提交；
- 不改写 P0–P5 历史；
- 本地定向测试；
- `verify-round2-p5.ps1`；
- GitHub Actions 成功；
- 对应真实证据文档。

## 14. 推荐执行顺序

比赛前按下列顺序推进：

```text
M0.1 写入范围、回读对账与精确 SHA CI
-> M1 丰富卡片
-> M2 卡片协作动作
-> M3 Base/Doc/Task 幂等和视图
-> M4a 常驻部署、健康检查、备份和回滚
-> 可选 M4b 孔明职配 common Review
-> 可选 M5 单 Agent 真实评估
-> P2 M6 公网 Webhook
```

M7 企业微信可以并行准备，但不应阻塞飞书主链路。M8 不进入当前复赛关键路径。

## 15. 最终复赛演示脚本

理想演示：

1. 查看一个未绑定公开 PR，证明无需绑定、无凭据、只读；
2. 查看 `Gitlink/forgeplus#356`，展示多仓库、卡片和 Base/Doc/Task；
3. Reviewer 点击领取并设置截止日期，原卡片和看板同步；
4. 展示 Agent 产生的有证据 Assessment；
5. 打开孔明职配测试 PR，生成 common Review ActionPlan；
6. Owner 二次确认；
7. GitLink 出现一条正式 common Review；
8. 展示 Review ID、head/fingerprint、审计和对账；
9. 强调 approve、reject、merge 仍被禁止；
10. 展示进程重启后任务恢复。

完成 M0.1、M1–M3 和 M4a 后，项目即可形成比赛主链路；M4b common Review、M5 Agent 与 M6 Webhook 只有在各自真实验收后才能进入展示表述：

> 面向 GitLink 仓库 Owner 的飞书 PR Review 工作台，以 GitLink 为状态真源，通过可靠任务、共享协作资源、Agent 证据和受控二次确认，提高海量 PR 的分诊、协作和正式 Review 效率。
