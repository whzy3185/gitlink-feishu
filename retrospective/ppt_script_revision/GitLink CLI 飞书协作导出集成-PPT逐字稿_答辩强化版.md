# GitLink CLI 飞书协作导出集成——PPT 逐字稿（答辩强化版）

赛题：GitLink智能化服务开源项目贡献赛  
团队：南山图灵  
正式口播：约 12 分 30 秒  
现场预算：15 分钟，预留约 1 分 30 秒用于翻页、停顿和设备延迟  
Backup：第 23—29 页，不计入正式口播

## 一、整场答辩只回答一个技术问题

> 当聊天协作入口能够触发 GitLink 真实写操作时，怎样确认操作者身份、冻结目标版本、限制并发写入，并最终确认 GitLink 实际发生了什么？

主线：

```text
Agent 并行开发
→ PR Review 压力
→ 飞书协作入口
→ GitLink 事实源
→ Reliable Review Gateway
→ Stable Review Model
→ ActionPlan
→ Identity + Version Validation
→ Lease
→ Single Mutation
→ GET Readback
→ Completed / Failed / Unknown
→ Reconciliation
→ 可复用的 GitLink CLI 工程能力
```

统一边界：飞书收集消息、协作状态和操作意图；GitLink 保存代码、PR、Review 和仓库状态，并完成最终权限判断。Identity Binding 只关联身份，不授予权限。Claim/Release 只修改飞书协作状态。

---

## 二、正式汇报：第 1—22 页

### 第 1 页｜封面

建议时长：25 秒

逐字稿：

各位评委老师好，我们是南山图灵，项目是“GitLink CLI 飞书协作导出集成”。它不只是把 PR 通知发到群里，而是解决一个更具体的工程问题：当飞书中的协作意图可能触发 GitLink 真实写操作时，怎样验证身份和目标版本，限制并发重复写入，并通过 GitLink 回读确认结果。为此，我们实现了 Review Gateway，把飞书协作入口与 GitLink 事实系统连接起来。

过渡：这个问题来自 Agent 并行开发以后更加集中的 Review 压力。

### 第 2 页｜Agent 加速开发，PR Review 成为新的协作压力点

建议时长：30 秒

逐字稿：

Agent 支持多个开发任务并行以后，PR 可以更集中地产生，而每个 PR 是否正确、是否应当合并，仍然依赖维护者判断。因此，Review 更容易成为新的协作瓶颈。我们不虚构 PR 增长比例；项目处理的是一个真实场景：在提交集中到达时，让维护者更快获得当前版本事实，让团队共享分工，同时不降低真实写操作的安全门槛。

过渡：这个场景最终表现为四类工程问题。

### 第 3 页｜Reviewer 面临的四类瓶颈

建议时长：30 秒

逐字稿：

第一，PR、版本和 Review 信息分散；第二，负责人和截止时间不透明；第三，Head 更新后，旧 Review 或旧操作意图可能过期；第四，写入存在身份错误、重复请求和网络不确定风险。前三类问题最终都会汇聚到第四类：一旦协作入口允许发起 GitLink 操作，系统怎样确保操作仍基于正确身份和当前版本，并避免不确定条件下重复写入。

过渡：因此，我们先确定事实源和协作入口的边界。

### 第 4 页｜把 PR Review 带入团队协作入口

建议时长：30 秒

逐字稿：

结论是：飞书不替代 GitLink。传统流程需要在 GitLink 查询、飞书沟通、确认负责人、再回 GitLink 检查版本和操作，最后再通知团队。新的流程从飞书进入，但数据先经过 Review Model，也就是稳定的 PR 审查数据契约；写入意图再转成 ActionPlan，也就是可持久化的操作计划。GitLink 始终保留事实和权限判断。

过渡：这使一条割裂的人工链路变成可验证流程。

### 第 5 页｜Before / After：减少切换，而不是绕过判断

建议时长：35 秒

逐字稿：

Before 是“查询、沟通、分工、回仓库、重查版本、手工操作、再同步”；After 是“飞书入口、Review Model、协作状态、ActionPlan、受控执行、GitLink 回读、飞书同步”。减少的是上下文切换、协作状态割裂、旧版本操作风险和写入结果不确定，不是维护者判断。领取不会变成 GitLink Reviewer，飞书也不会替用户获得仓库权限。

过渡：第六页是整场汇报的技术总纲。

### 第 6 页｜总体架构与两条技术契约

建议时长：45 秒

逐字稿：

架构分为三层：飞书承载协作入口，Review Gateway 承载持久化和控制，GitLink 给出最终事实与权限。后面主要看两条契约。读取侧要说明数据来自哪里、是否完整、描述的是哪个代码版本；写入侧要说明谁在操作、操作基于哪个版本、并发时谁获得执行权，以及写后如何确认远端结果。对应实现就是 Review Model、ActionPlan、身份与版本校验、Lease、单次 Mutation 和 GET Readback。

过渡：先看 Gateway 怎样把一个即时消息变成可恢复的后台任务。

### 第 7 页｜从长连接消息到持久任务

建议时长：40 秒

逐字稿：

飞书消息经长连接进入 Channel，先完成事件归一化和策略检查，再在一个 SQLite 事务中完成消息去重、Job 和消费者持久化，随后由分类 Worker 执行。去重解决同一消息重复投递，持久 Job 解决“事件已接收但任务未完成时进程退出”，Worker 则避免 GitLink 网络请求阻塞飞书回调。安全阶段的过期 Lease 可以在重启后回收。独立的 Event Inbox 用于 GitLink Webhook，不把两条入口混为同一实现。

过渡：Worker 读取 GitLink 后，不直接拼卡片，而是先建立稳定数据契约。

### 第 8 页｜Review Model：读取侧的稳定数据契约

建议时长：40 秒

逐字稿：

Review Model 承担四个作用：提供 UI 数据契约，表达采集是否完整，描述当前代码版本，并为 ActionPlan 提供校验基础。它归一化 PR 主体、文件和提交统计、版本、Review、Reviewer 汇总和讨论；分段失败通过 collection status、partial 和结构化 fetch errors 表达。PR 主体缺失会失败，非关键分段失败不会伪装成完整数据，也不会让高风险写入继续。

过渡：模型中的 Head SHA 和 Source Fingerprint 分别保护两种变化。

### 第 9 页｜Head SHA、Fingerprint 与当前有效 Review

建议时长：25 秒

逐字稿：

Head SHA 回答“代码提交是否变化”；Source Fingerprint 回答“整个审查上下文是否变化”。后者对规范化后的 Head、Patchset、Review、Reviewer 汇总、讨论、采集状态和错误信息做稳定排序，再以 SHA-256 计算摘要。即使 Head 不变，Review 或讨论变化也能使 Fingerprint 变化。Reviewer 结论还会按当前 Head 与历史 Head 区分，旧批准不能直接代表新版本。

过渡：稳定事实进入飞书后，协作状态仍要与仓库权限分开。

### 第 10 页｜Claim、Release 与 Deadline 只管理协作

建议时长：30 秒

逐字稿：

领取、取消领取和截止时间只回答“团队里谁负责、何时处理”，状态保存在 SQLite 协作模型中。真实飞书验收已经覆盖领取、设置截止和释放，并复用同一张主卡。但负责人不等于 GitLink Reviewer，领取也不会修改仓库成员、Reviewer 或权限。这条边界避免把协作便利误解为授权能力。

过渡：需要真实写入时，系统再建立完整身份链。

### 第 11 页｜身份关联不等于仓库授权

建议时长：30 秒

逐字稿：

身份链是：飞书用户先通过 Identity Binding 关联 GitLink 登录名；执行时，当前 Credential 再调用 `/users/me`，确认真实认证用户与绑定身份一致；最后，操作是否有权限仍由 GitLink 服务端判断。Binding 是身份关联，Credential Identity 是当前实际登录用户，Permission 是 GitLink 的决定。当前没有用户 OAuth、Refresh Token、用户 Credential Store、Owner Token 代理或自动权限提升。

过渡：身份和事实明确以后，聊天意图才会成为可执行对象。

### 第 12 页｜ActionPlan：为什么聊天消息不能直接执行

建议时长：60 秒

逐字稿：

聊天消息是瞬时文本，缺少稳定的执行上下文，所以不能直接越过 GitLink 写边界。系统先生成 ActionPlan，也就是持久化操作计划。真实模型冻结 Plan ID、Installation、来源群、仓库、PR 编号、飞书 Actor、GitLink 登录名、动作、内容、Expected Head、Source Fingerprint、Request ID、幂等键、来源 Job、创建时间、15 分钟有效期和状态。准备阶段只保存计划，GitLink 写入仍为零。这样后续执行检查的是同一个对象，而不是重新解释聊天文本。

过渡：Local 和 Auto 的差别，主要是最终使用哪一份 Credential。

### 第 13 页｜Local / Auto：两种 Credential Boundary

建议时长：60 秒

逐字稿：

Local 模式中，飞书负责收集意图和生成 Plan，本机 CLI 使用用户当前登录的 Credential 完成最终确认；优点是 Gateway 不需要集中保存用户 Token，适合个人凭据边界严格的团队。Auto 模式使用 Gateway 当前 Credential，但只有 `/users/me` 与 Identity Binding 一致时才直接执行，否则保留 Plan，回退到本地确认。两种模式都不授予权限，GitLink 服务端仍完成最终判断。当前真实证据以 Local 为主，Auto 的身份门禁与单次写入由 Fake Server 和 SQLite 测试覆盖。

过渡：进入执行阶段后，还需要区分四个经常被混淆的机制。

### 第 14 页｜ActionPlan、Idempotency、Lease 与 Single Mutation

建议时长：60 秒

逐字稿：

ActionPlan 定义“准备执行什么”；Event Idempotency，也就是事件幂等，避免同一投递重复形成逻辑工作；Lease 是有时限的执行权，用 SQLite 条件更新决定哪个 Worker 可以继续；Single Mutation 则限制真正跨越 GitLink 写边界的调用。Lease 默认两分钟，只能在 pre-write，也就是尚未标记远程写入可能发生时过期恢复。进入 remote-write-possible 后禁止再次 Claim。终态 Plan 也不能重跑。准确说法是单实例 SQLite 下的应用层单次写入约束，不是分布式 Exactly Once。

过渡：即使只发出一次请求，HTTP 返回也不足以说明远端结果。

### 第 15 页｜GET Readback、Unknown 与 Reconciliation

建议时长：60 秒

逐字稿：

写请求发出后，系统进行有限次数的 GitLink GET 回读。远端状态、Review ID、Head、内容和 Request ID 与预期匹配，才进入 completed。写前条件失败则不越过 Mutation 边界；Head 或 Fingerprint 变化在用户界面中表达为计划已失效，当前持久化状态记为 failed，并保留原因，不是独立的 stale 状态。如果请求超时，或回读仍无法确认，就进入 unknown needs reconciliation。Unknown 表示“远端可能已经写入”，系统会停止自动 Mutation，不能盲目重试。当前 ActionPlan 主要按 Request ID、Review 或 PR 状态人工核对；资源 Operation 另有有限的自动与人工 Reconciler。两者不能混称为全部自动恢复。

过渡：核心链路到这里完成，飞书资产只作为可选投影。

### 第 16 页｜Base、DocX、Wiki 与 Task 的边界

建议时长：15 秒

逐字稿：

这些能力只把协作结果投影到多维表格、文档、知识库和任务中，不参与 GitLink 事实判断和受控写入；未配置时不影响核心 Review。当前代码和离线测试存在，但真实平台验收受环境阻塞。

过渡：下面用工程证据说明这些边界不是只写在架构图里。

### 第 17 页｜工程证据与 Failure Matrix

建议时长：35 秒

逐字稿：

当前三个核心包静态统计有 878 个 Go 测试函数，分布在 66 个测试文件；本轮在最新工作树重跑 workflow、feishu 和 pr 三个包，全部通过。历史故障注入记录 20 个场景全部通过。最关键的零写入门禁包括 Plan 过期、身份不匹配、Head 或 Fingerprint 变化、关键数据 partial、重复事件和显式失败 CI；并发确认由 Lease 限制为最多一个 POST；超时或回读不确定则停止新增写入并进入 Unknown。这里没有性能百分比、QPS 或 P95 宣称。

过渡：为了进入上游，项目还做了必要的基础工程收口。

### 第 18 页｜i18n 与跨平台是可维护性工作

建议时长：15 秒

逐字稿：

中文命令和卡片、Windows 与 PowerShell 兼容、构建与测试门禁，目的都是让贡献可以被上游编译、审查和维护。Windows 本地全仓测试、vet 和 build 有历史通过证据；最终 Linux race 证据仍标记 pending，不把它描述成已完成。

过渡：同一仓库还存在一组真实的只读 Workflow Shortcut。

### 第 19 页｜已经存在的 Workflow Shortcut

建议时长：20 秒

逐字稿：

`workflow +triage`、`+health`、`+pr-summary` 和 `+repo-report` 已注册，并有规则、渲染和读取边界测试；triage 与 health 还保存了只读远程 smoke 记录。它们没有全部直接调用当前 Review Model，因此准确关系是：两者体现相同的“先归一化 GitLink 数据，再供 CLI 或 Agent 使用”的工程方向。

过渡：这也是项目能够继续进入 GitLink CLI 的原因。

### 第 20 页｜为什么不是一次性飞书机器人

建议时长：20 秒

逐字稿：

飞书只是当前协作入口，核心资产是 GitLink CLI 的读取能力、稳定 Review 数据契约、持久 Gateway 和受控写入边界。上层可以继续增加新的协作入口，但 GitLink 事实源、身份与版本校验、单次写入和回读规则保持一致。因此项目可以按独立模块进入上游，而不是依赖一套演示环境长期存在。

过渡：最后用三个结论收束。

### 第 21 页｜正式总结

建议时长：30 秒

逐字稿：

第一，Efficient Collaboration：飞书统一查看 PR、负责人和截止时间。第二，Controlled Execution：ActionPlan 冻结意图，身份与版本校验阻止 stale 操作，Lease 与终态保护约束单次 Mutation，GET Readback 确认结果，Unknown 阻止盲目重试。第三，Reusable Engineering：Review Model、Gateway、CLI Workflow 和测试证据可以继续复用。飞书承载协作，Gateway 承载控制，GitLink给出最终事实和权限判断。

### 第 22 页｜正式结束

建议时长：15 秒

逐字稿：

我们的正式汇报到这里。这个项目不是把 Review 搬进聊天，而是把团队协作、GitLink 事实和受控执行连接成一个可验证流程。后续 Backup 页面保留了飞书配置、Local / Auto 和真实 GitLink 写入证据，可根据评委问题展开。谢谢各位老师。

---

## 三、Backup / Engineering Evidence：第 23—29 页

以下页面不计入正式口播时间。推荐查看优先级：29 → 27 → 28 → 25 → 26 → 23 → 24。

### 第 23 页｜Backup：飞书应用与长连接

备用口播，20 秒：

这页证明飞书应用已经发布，机器人和长连接事件订阅已配置。长连接适合当前单实例部署，不要求额外公网回调地址。它只证明飞书消息入口可用，不代表 GitLink 写权限已经开放；写入仍由 ActionPlan 和 GitLink Credential 控制。

### 第 24 页｜Backup：权限与卡片回调

备用口播，20 秒：

核心权限用于接收群内 @消息并以机器人身份回复；交互卡片启用对应回调。Base、DocX、Wiki 和 Task 属于可选权限，不计入核心最小权限。部署应先按核心链路配置，再按实际启用的 Projection 增加权限。

### 第 25 页｜Backup：真实 PR 查询

备用口播，25 秒：

这里展示真实飞书群中的中文帮助和 PR 查询。查询返回 GitLink 当前版本、变更和 Review 结论，GitLink 写入为零。正式证据 `real-gitlink-read.json` 记录了 PR、files、versions、reviews 和 threads 均被读取，卡片创建、更新和回复另有 HTTP 200 证据。

### 第 26 页｜Backup：真实协作状态

备用口播，20 秒：

这里展示领取、Deadline 和 Release。真实证据记录同一 Canonical Card 被复用，领取后进入 reviewing，释放后回到 unassigned；这些操作产生飞书卡片和回复写入，但 GitLink Mutation 为零。

### 第 27 页｜Backup：Local Controlled Execution

备用口播，25 秒：

Local 是当前真实验收证据最完整的执行方式。飞书先生成 ActionPlan，本地 CLI 使用当前登录 Credential 调用 `/users/me`，重新读取当前 PR，再取得 Lease、执行一次写入并回读。普通 Review、批准、需要修改、拒绝并关闭和合并都保存了真实结果与 duplicate zero 证据。

### 第 28 页｜Backup：Auto Controlled Execution

备用口播，25 秒：

Auto 使用 Gateway 当前 Credential。当 `/users/me` 与绑定登录名一致时才继续，否则保留 ActionPlan 并回退到本地确认。Fake Server 和 SQLite 测试覆盖身份匹配、身份不匹配零写入和成功路径单次写入；当前正式证据归档未完整区分一条独立 Auto 真实链路，因此不扩大宣称。

### 第 29 页｜Backup：真实 Mutation + GET Readback

备用口播，30 秒：

这是最高优先级证据页。真实验收记录 common Review、批准、需要修改、拒绝并关闭和合并各发生一次预期 GitLink Mutation，复用已完成 Plan 时新增写入为零。拒绝并关闭回读到 closed，合并回读到 merged，Review 动作回读 Review ID、状态、Head、内容和 Request ID。Head stale 的真实测试则保持零 Mutation。系统最终相信的是 GitLink 回读事实，而不是飞书文案或单次 HTTP 返回。

---

## 四、现场口径红线

1. 不说“飞书保存 GitLink 权限”；说“GitLink 服务端最终校验权限”。
2. 不说“领取以后成为 Reviewer”；说“领取只记录飞书协作负责人”。
3. 不说“Auto 保存每个用户 Token”；当前没有 OAuth、Refresh Token 或用户 Credential Store。
4. 不说“HTTP 成功就是操作完成”；必须经过 GitLink GET Readback。
5. 不说“网络失败说明零写入”；不确定时进入 Unknown 并停止自动 Mutation。
6. 不说“Exactly Once”；说“单实例 SQLite 下的应用层单次写入约束”。
7. 不说“CI 成功才允许 Merge”；当前只阻止明确失败，unknown 不被当作通过。
8. 不说“所有 Unknown 自动恢复”；ActionPlan Unknown 当前主要要求人工核对。
9. 不把 Base、DocX、Wiki、Task 说成核心依赖或真实平台 PASS。
10. 不宣称 QPS、P95、吞吐量、效率百分比或多实例高可用。

## 五、三句评委记忆点

> Agent 支持并行开发，但没有自动增加维护者的 Review 判断带宽。

> 飞书承载协作，Review Gateway 承载控制，GitLink 给出最终事实和权限判断。

> ActionPlan 冻结意图，Lease 约束执行权，GET Readback 决定系统最终能否宣布完成。
