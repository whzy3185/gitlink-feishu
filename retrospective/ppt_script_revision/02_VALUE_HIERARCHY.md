# 答辩价值层级与页面调整建议

## 1. 四层价值结构

### 第一层：核心技术贡献——Controlled Review Execution

这是答辩的技术主角，回答“聊天入口如何安全触发真实 GitLink 操作”。

- ActionPlan：把即时意图冻结为带作用域、身份、版本、内容和有效期的持久计划。
- stale protection：执行前重新读取 Head SHA 和 Source Fingerprint，变化即零写入失效。
- identity verification：Identity Binding 只关联身份，实际 Credential 仍由 `/users/me` 验证。
- Lease：有时限的执行权，阻止并发路径同时越过写边界；只允许在 pre-write 阶段恢复。
- Single Mutation：应用层每次执行最多开放一个 GitLink POST 路径，不宣称分布式 Exactly Once。
- GET Readback：写后读取 GitLink 事实，匹配后才完成。
- Unknown / Reconciliation：远端结果不确定时停止盲目重试，转入核对。

### 第二层：可靠工程基础——Reliable Gateway

这一层回答“为什么它能作为持续运行的协作入口，而不只是演示脚本”。

- 飞书消息：Long Connection → 归一化与策略检查 → SQLite 原子去重和持久 Job → 分类 Worker。
- GitLink Webhook：标准事件模型 → Event Inbox → Subscription Route → 持久 Job。
- Worker：GitLink 读取、协作动作、受控写入和资源写入分池，避免相互阻塞。
- SQLite：保存 Job、Operation、ActionPlan、协作状态、卡片映射和对账状态。
- restart recovery：安全阶段的过期 Lease 可回收；越过远程写边界的非幂等操作进入 Unknown/对账。
- partial collection / structured fetch errors：读取失败被结构化表达，而不是伪装成完整数据。

### 第三层：协作产品能力——Feishu Collaboration

这一层提供用户可见价值，但不应抢占核心技术篇幅。

- PR query、卡片和回复。
- Claim、Release、Deadline。
- Repository Binding 和公开仓库显式查询。
- Identity Binding 与可读成员显示。
- 协作状态与 GitLink Reviewer/权限严格分离。

### 第四层：GitLink CLI 与开源贡献

这一层说明项目不是一次性的飞书机器人。

- 复用 GitLink CLI 的 API Runtime、认证和 Shortcut 注册体系。
- `workflow +triage`、`+health`、`+pr-summary`、`+repo-report` 已真实存在并有测试。
- i18n、Windows/PowerShell、构建和 CI 修复为上游可维护性服务。
- Review Model、Gateway 和 Workflow 都体现“先归一化事实，再供 CLI/Agent/协作入口复用”的方向，但现有 Workflow 并非全部直接调用 Review Model。

## 2. 原 1—29 页调整策略

不改 PPT 的物理顺序，通过口播重构价值顺序：

1. 第 1 页直接提出技术题：身份、版本、并发单次写入、远端结果确认。
2. 第 2—3 页把背景压缩到 65 秒，不使用无证据的 PR 增长幅度。
3. 第 4—5 页用 Before/After 把协作价值连接到 ActionPlan 和 Readback。
4. 第 6 页升级为技术总纲，预告读取侧与写入侧两条契约。
5. 第 7—11 页不再只是功能列表，而是讲持久任务、Review Model、版本新鲜度和身份边界。
6. 第 12—15 页使用 4 分钟，形成技术高潮：Plan → Validation → Lease → Single Mutation → Readback → Unknown。
7. 第 16 页压缩到 15 秒，只说明可选投影边界。
8. 第 17 页加入可核对的工程数据与 Failure Matrix，不使用性能数字。
9. 第 18—20 页合计 90 秒，说明上游维护、真实 Workflow 和架构复用。
10. 第 21 页正式总结，第 22 页正式结束。
11. 第 23—29 页统一作为 Backup / Engineering Evidence，不计正式时长；优先展示 29、27、28、25。

## 3. 前 3 分钟必须建立的评委记忆

> 飞书只收集协作与操作意图；GitLink 保留事实和权限。Review Gateway 的技术价值，是把一条聊天意图转换为可验证的 ActionPlan，并在身份、版本、并发和网络不确定条件下，把真实写入限制在可追踪、可回读的边界内。

## 4. 正式汇报时间层级

| 层级 | 页面 | 时间 | 占正式 12 分 30 秒口播 |
|---|---|---:|---:|
| 核心受控执行 | 1、3、6、12—15、21 | 5 分 25 秒 | 43% |
| 可靠 Gateway | 6—9、17 | 3 分 10 秒 | 25% |
| 飞书协作 | 4—5、10—11、16 | 2 分 25 秒 | 19% |
| CLI 与开源贡献 | 18—20、21 | 1 分 30 秒 | 12% |

