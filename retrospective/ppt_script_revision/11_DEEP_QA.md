# 深层答辩题库

## Architecture

### 1. 为什么需要 Review Gateway？

结论：因为飞书消息不能直接承担仓库事实和真实写入状态。Gateway 把即时消息变成持久 Job 或 ActionPlan，统一处理身份、版本、Lease、写入和回读。代码集中在 `shortcuts/feishu`，真实群聊与 GitLink 写入证据证明它不是纯 UI；当前部署仍是单实例 SQLite。

### 2. 为什么不直接由飞书调用 GitLink API？

结论：直接调用缺少稳定的 Actor、版本、幂等和结果确认上下文。ActionPlan 先冻结仓库、PR、身份、Head、Fingerprint、内容和有效期，再执行单次 Mutation 与 GET 回读。这样可以在旧版本、并发确认和网络超时时阻止盲目写入。

### 3. 为什么这部分能力适合与 GitLink CLI 结合？

结论：Gateway 复用 CLI 已有的认证、API Runtime、PR/Workflow Shortcut 和输出契约，不需要复制一套 GitLink 客户端。飞书只是消费入口；Review Model 和受控 Writer 也可由本地 CLI 使用，因此功能可以按独立模块向 GitLink CLI 上游贡献。

### 4. Gateway 是否成为单点故障？

结论：当前是单实例服务，确实存在单点边界。SQLite、持久 Job、Lease、重启恢复、健康检查和备份降低故障影响，但不构成多实例高可用。仓库明确不支持分布式数据库、跨主机 Leader Election 和零停机迁移，不能把当前形态描述成 HA。

## Review Model

### 5. 为什么需要 Review Model？

结论：上层不能依赖多个 GitLink 接口的原始差异。Review Model 同时承担 UI 契约、完整性表达、当前版本描述和 ActionPlan 校验输入。代码归一化 PR、版本、Review、讨论及错误；Fake Server、golden test 和真实 GitLink 读取共同提供证据。

### 6. Head SHA 与 Source Fingerprint 有什么区别？

结论：Head SHA 只判断代码提交是否变化；Fingerprint 判断整个规范化 Review 上下文是否变化。即使 Head 不变，Review、Reviewer 汇总、讨论、采集状态或结构化错误变化，也会改变 Fingerprint。前者防代码 stale，后者防审查事实 stale。

### 7. Fingerprint 如何计算？

结论：代码先稳定排序 Review、Thread、Section Status 和 Fetch Error，再序列化 schema、仓库、PR、Head、Patchset、Review、Reviewer Summary、Thread、Summary 与采集状态，最后计算 SHA-256。它不是完整 Diff Hash，也不包含任意运行时文案。

### 8. Partial data 为什么还能返回卡片？

结论：查询和高风险写入的要求不同。PR 主体存在时，部分文件、Review 或讨论失败仍可展示有限事实和明确的“数据不完整”；但准备或执行受控写入要求 `collection_status=complete` 且 `partial=false`，所以 partial 不能授权高风险操作。

### 9. 哪些 Partial 状态禁止执行高风险操作？

结论：当前实现不是按某几个错误码放行，而是统一要求完整 Review Context。`Partial=true`、`CollectionStatus!=complete` 或当前 Head 缺失，都阻止写入计划或确认。这样更保守；代价是某个非关键接口故障也可能暂时阻止高风险动作。

### 10. 旧 Review 如何区分当前版本和历史版本？

结论：每条 Review 的 `commit_id` 与当前 Head 比较，产生 current、outdated 或 unknown。系统按 Reviewer 分组，只选择当前版本中顺序可确定的最后有效结论；顺序无法确定时保留 unknown，不把旧批准当成当前版本批准。

## ActionPlan

### 11. 为什么一定要 ActionPlan？

结论：聊天文本会变化、会重复，也缺少执行时上下文。ActionPlan 把一次意图冻结为持久、可过期、可取消、可校验的对象，并生成 Request ID。真实 common、批准、需要修改、关闭和合并都通过同一模型，重复确认不会新增 Mutation。

### 12. ActionPlan 保存哪些字段？

结论：v3 保存 Plan、来源群、仓库和 PR、飞书 Actor、GitLink 身份、Expected Head、Fingerprint、Action、Content、Request ID、幂等键、Lease、Mutation、Reconciliation 与时间。执行器因此不用重新解释聊天文本，证据也保留 Plan 和 Request ID。

### 13. Plan 如何过期？

结论：新 Plan 默认 15 分钟有效。Local 执行显式解析 `ExpiresAt`；SQLite Claim 也要求 `expires_at > now`。过期后不能获得 Lease，因此零写入。当前代码门禁明确，但专门的过期测试证据不如 stale 与并发测试完整，答辩中应把它称为代码门禁。

### 14. 用户如何取消 Plan？

结论：取消要求同一 Actor 获得 Plan 后写入 `cancelled` 终态。取消后的 Plan 不能再次 Claim。卡片提供“取消操作”入口，测试覆盖跨 Actor 取消失败和 cancelled 终态不可重跑；取消只影响计划，不执行 GitLink Mutation。

### 15. PR 更新以后旧 Plan 怎么处理？

结论：执行前重新读取完整 Review Context，对比 Expected Head 和 Source Fingerprint。任一变化都把 Plan 标为 stale，并保持 MutationStatus=none。Head stale 已有真实平台零写入证据；Fingerprint-only stale 目前只有离线回归测试。

## Identity / Permission

### 16. Identity Binding 是否授予权限？

结论：不授予。Binding 只表示某飞书用户关联哪个 GitLink 登录名。执行时仍要用当前 Credential 调 `/users/me`，并由 GitLink 服务端判断该用户是否有 Review、关闭或合并权限。Claim、AdminUserIDs 也不能替代 GitLink 权限。

### 17. 用户把自己绑定成别人怎么办？

结论：仅修改绑定不足以执行。实际 Credential 的 `/users/me` 必须与 Plan 中冻结的 GitLink 登录名一致；不一致时 Auto 不写入，Local 也拒绝。绑定配置本身仍需要受控管理，这是部署与运维责任，不是 GitLink 授权机制。

### 18. Auto 如何确认真实身份？

结论：Auto 使用 Gateway 当前 Credential 调 `/users/me`，将真实 login 与 Identity Binding 比较。相等才继续；不相等则 Plan 保留为待本地确认，GitLink 写入为零。Fake Server 集成测试覆盖 match=1 write、mismatch=0 write。

### 19. Gateway Credential 泄漏怎么办？

结论：当前没有专用密钥托管系统，所以应按部署凭据处理：只通过环境变量或 Secret Reference 加载、限制文件和进程权限、定期轮换，并立即撤销泄漏 Token。代码有脱敏与 Secret Scan，但不能把它们说成能消除凭据泄漏风险。

### 20. 为什么没有实现用户 OAuth？

结论：比赛范围优先验证受控 Review 链路，而完整 OAuth 还需要授权回调、PKCE、Refresh Token、撤销、加密存储和多租户治理。当前 Local 模式已经保留个人 Credential 边界；Auto 只适合受控 Gateway Credential 场景，因此没有伪造一套不完整 OAuth。

### 21. Local 与 Auto 应如何选择？

结论：个人身份和凭据边界优先时选 Local；团队接受 Gateway 以固定身份执行、并完成绑定和 Token 运维时才选 Auto。两者都经过版本和回读门禁。当前真实验收以 Local 为主，Auto 的核心合同主要由 Fake Server 和 SQLite 测试支撑。

## Concurrency / Idempotency

### 22. Event Idempotency 和 Lease 有什么区别？

结论：Idempotency 防止同一消息或事件重复形成逻辑工作；Lease 决定已有工作当前由谁执行。前者作用在入队/计划身份，后者作用在并发 Worker 或确认路径。两者缺一不可，但都不能单独证明远端副作用严格只发生一次。

### 23. Lease 与 Plan ID 有什么区别？

结论：Plan ID 标识“哪一个操作计划”，Lease 标识“谁在一段时间内拥有执行权”。同一 Plan ID 在不同时间可能经历 pre-write Lease 恢复；一旦进入 remote-write-possible 或终态，就不能通过新 Lease重新执行。

### 24. 两个 Worker 同时执行怎么办？

结论：SQLite 条件更新只允许一个执行者把 Plan 从 pending 切到 executing 并写入 Lease Owner。另一个执行者看到活动 Lease 或终态后失败。并发测试启动两条确认路径，最终只有一个 completed，Fake Server 只收到一次写入。

### 25. 两次点击确认怎么办？

结论：如果是同一事件，入站幂等先去重；如果形成同一 Plan 的两条确认路径，Lease 和终态保护阻止第二次越过写边界。真实批准、需要修改、关闭和合并证据都记录 second confirmation 被拒绝、additional mutation=0。

### 26. Single Mutation 如何证明？

结论：Writer 的生产代码只有一个受控 POST 点，POST 前调用 `BeforePOST` 持久化 remote-write-possible；测试用计数 Fake Server 断言并发、重复和 stale 场景的 POST 数量；真实验收记录每个授权动作 actual=1、重复确认 additional=0。

### 27. 系统能否宣称 Exactly Once？

结论：不能。当前能宣称的是单实例 SQLite 部署下的应用层 best-effort single mutation。因为网络中断时远端可能成功而本地未知，系统只能依靠持久写边界、有限 GET 回读、Unknown 和停止重试降低重复风险，无法建立分布式事务意义的 Exactly Once。

## Failure Handling

### 28. Mutation timeout 怎么办？

结论：不立即重试 POST。Writer 在一次 POST 后进行最多三次有限 GET 回读，检查 Review 或 PR 状态。若匹配预期则 completed；仍无法判断则 unknown。这样把网络超时视为“结果不确定”，而不是“远端肯定没写”。

### 29. 为什么不直接 retry？

结论：Review、评论、关闭和合并是非幂等或业务幂等不透明的操作，重复 POST 可能产生第二条记录或重复生命周期操作。系统只在确认未越过写边界时恢复；一旦 Mutation 可能发生，就停止自动写入并要求回读或人工核对。

### 30. Unknown 是什么？

结论：Unknown 表示本地无法确认远端副作用是否已经发生。Plan 保存 mutation possible、reconciliation required，清空 Lease，并拒绝再次 Claim。它不是失败的同义词，更不能在卡片上显示“已完成”。

### 31. Reconciliation 怎么做？

结论：GitLink ActionPlan 当前主要使用 Request ID、Review 列表或 PR 状态人工核对，确认前禁止再写。飞书资源 Operation 有独立 Reconciler：Base 可按唯一键自动修复，部分场景转 manual_required。两条机制能力不同，当前没有统一自动 ActionPlan 对账器。

### 32. Mutation 成功但进程立即崩溃怎么办？

结论：POST 前已经持久化 remote-write-possible；优雅关闭或重启逻辑不会把这类 Plan 重置为 pending，而会保留为 Unknown/Needs Reconciliation。故障注入 F13 覆盖“远程请求后进程退出”，但真实平台没有故意制造该故障。

### 33. Readback 自己失败怎么办？

结论：在有限回读次数内仍失败，就进入 Unknown，而不是宣布成功或重新 POST。系统保留 Request ID、目标 PR、Head 和动作，供后续人工 GET 核对。当前通用资源对账更完善，GitLink ActionPlan 自动对账仍是明确限制。

### 34. SQLite 损坏怎么办？

结论：仓库提供备份、校验和恢复流程，并测试 checksum 错误、无效 SQLite、未来 schema 和运行中恢复拒绝。它降低数据损坏后的恢复风险，但不是跨区域灾备。真实部署仍需备份目录保护、介质冗余和恢复演练。

### 35. 多实例部署是否安全？

结论：当前不支持。服务使用 `<state-db>.lock` 限制单活动进程，SQLite 和飞书长连接也按单实例设计。没有分布式锁、共享数据库和跨实例协调，因此不能横向启动多个 Gateway 来获得高可用。

## Merge Safety

### 36. Controlled Merge 检查什么？

结论：执行继承 ActionPlan 的 Installation、群、仓库、Actor、Head 和 Fingerprint，再检查 Plan/Lease、PR 仍 open、当前 Head、一致身份和显式失败 CI；POST 后回读 PR 必须为 merged。真实测试只在一次性测试基分支上进行，默认分支未改变。

### 37. CI 状态从哪里获取？

结论：Merge Writer 使用 Review Context 中的 `WorkItem.CISummary.State`。该字段来自 GitLink 读取层可获得的 CI 信息；无法获得时是 unknown，而不是 success。当前实现没有建立独立、完整的多提供方 CI 聚合系统。

### 38. CI 信息缺失怎么办？

结论：当前 Gate 只阻止明确的 failed/failure/error，unknown 会保留为未知并可能继续执行。这是已知边界，不应说成“CI 通过”。真实 merge 证据明确记录 CI=unknown，且只在用户授权的一次性测试分支中接受。

### 39. Head 更新后能否继续 Merge？

结论：不能沿用旧 Plan。执行前当前 Head 与 Expected Head 不一致会 stale，Fingerprint 变化也会 stale，Mutation 为零。用户需要重新查询并生成新 ActionPlan。Head stale 已有真实证据，Fingerprint-only stale 当前是离线回归。

## Evidence

### 40. 哪些能力在真实飞书验证？

结论：真实证据覆盖飞书卡片 create/patch/reply、PR 查询、领取/截止/释放，以及 common、批准、需要修改、拒绝并关闭、合并后的飞书结果更新。双群隔离、Base、Doc、Task 和 Webhook 在最终清单中仍为环境阻塞。

### 41. 哪些只经过 Fake Server？

结论：Auto 身份匹配/不匹配、Fingerprint-only stale、网络超时 Unknown、严格 Readback 不匹配、并发 Lease、CI 显式失败阻断和多种恢复故障主要由 httptest/Fake Server 与 SQLite 验证。答辩中应明确说“离线测试覆盖”。

### 42. 哪些只经过单元测试？

结论：部分纯规则，如 Fingerprint 规范化、Reviewer 排序、展示映射、输入解析和一些边界错误主要由单元测试覆盖；许多又被集成测试间接使用。Plan 过期缺少专门的强回归用例，是证据矩阵中的 PARTIAL 项。

### 43. 有没有真实 GitLink Mutation？

结论：有。最终证据记录 common Review、Approve、Request Changes、Reject & Close 和 Controlled Merge 各一次授权 Mutation；每项都有回读和重复确认零新增写入。测试仓库是 `muel/gitlink-feishu_agent`，Merge 使用一次性基分支，默认分支未改变。

### 44. 怎么证明不是 UI Demo？

结论：证据不仅有飞书截图，还有 ActionPlan SQLite 状态、Request ID、Review ID、GitLink 前后状态、Mutation 计数、GET Readback、重复确认结果、故障注入和目标包测试。UI 卡片只有在这些状态完成后更新，GitLink 才是最终事实源。

### 45. 如果现场要求重新演示，最小演示链是什么？

结论：优先演示零风险链：`帮助` → 查询一个公开 PR → 领取 → 设置截止 → 释放，证明协作和 GitLink 零写入。若明确授权真实写入，再在一次性测试 PR 上生成普通 Review ActionPlan、Local 确认、查看 Review ID 和 GET 回读；不现场合并默认分支。
