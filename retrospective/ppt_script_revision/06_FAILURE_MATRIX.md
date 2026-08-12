# Controlled Review Failure Matrix

本表以当前工作树 `937b7ee42db4`、定向测试 PASS 和历史真实平台证据为准。`Mutation` 表示 GitLink 业务写入，不包含飞书卡片或回复。

| 场景 | 检测机制 | 系统行为 | GitLink Mutation | 证据 |
|---|---|---|---:|---|
| Plan 过期 | `expires_at > now`；Local 显式解析 `ExpiresAt` | 拒绝 Claim/执行 | 0 | `review_action_plan.go:283`；`review_local_confirm.go:63-65`；专门测试 PARTIAL |
| Identity mismatch | Identity Binding + `/users/me` | Auto 保留为待本地确认；实际执行拒绝 | 0 | `controlled_review_actions_test.go:161`；`review_local_confirm.go:101-106` |
| Head changed | 当前 `CurrentHeadSHA` 对比 `ExpectedHeadSHA` | 执行前拒绝；用户语义为 stale，当前 Plan 持久化为 `failed` 并保留原因 | 0 | `review_local_confirm_test.go:117-131`；`real-stale.json` |
| Fingerprint changed | 当前 `SourceFingerprint` 对比 Plan | 执行前拒绝；用户语义为 stale，当前 Plan 持久化为 `failed` 并保留原因 | 0 | `review_gateway_test.go:3100`；真实平台仅 PARTIAL |
| Partial critical data | `Partial` 或 `CollectionStatus != complete` | 不生成或不执行高风险计划 | 0 | `review_write.go:31`、`:167`；`review_local_confirm.go:76` |
| Duplicate Feishu message | 稳定 `message_id` 进入 SQLite dedupe reservation | 不生成第二个 Job | 0 | `review_gateway_test.go:1042`；真实平台同事件重投未单独留证 |
| Duplicate GitLink webhook | Installation + delivery identity 进入 Event Inbox | 返回 accepted，不重复路由 | 0 | `review_event_inbox_test.go:27`；`review_webhook_ingress_test.go:234` |
| Concurrent confirm | ActionPlan Lease + SQLite 条件更新 | 只有一方取得执行权 | ≤1 | `review_local_confirm_test.go:133-155` |
| Terminal Plan 再确认 | `status` 终态检查 | 拒绝第二次执行 | 0 新增 | `controlled_review_actions_test.go:62`；真实动作 duplicate protection |
| Lease 在 pre-write 阶段过期 | `status=executing`、`reconciliation=pre_write`、Lease 到期 | 允许重新 Claim | 0 或后续 ≤1 | `review_gateway_test.go:2566` |
| Lease 在远程写边界后过期 | `remote_write_possible` / `required` | 禁止重新 Claim，进入核对 | 不新增写入 | `review_action_plan.go:319-320`；`review_shutdown.go` |
| CI 明确失败 | `CISummary.State` 为 failed/failure/error | 拒绝 Merge | 0 | `controlled_pr_action_writer.go:84-85`；对应测试 |
| CI unknown | 当前 Gate 只识别显式失败 | 显示 unknown；不会表述为通过；授权测试可继续 | 依后续门禁 | `real-merge.json`；这是当前限制 |
| Mutation 明确失败且回读不匹配 | POST 结果 + 有限 GET Readback | `failed` 或 `unknown`，取决于是否能排除远端副作用 | 按实际已发 POST 计 | Writer Fake Server 测试 |
| Mutation timeout | POST 后最多 3 次 GET Readback | 尝试核实远端 | ≤1 POST | `controlled_pr_action_writer.go:116-140` |
| Readback 仍无法判断 | Readback 未匹配预期 | `unknown_needs_reconciliation`，停止自动重试 | 不新增写入 | `controlled_review_actions_test.go:348-361` |
| Mutation 后本地完成状态保存失败 | `FinishReviewActionPlan` 返回错误 | Plan 标为 Unknown | 不新增写入 | `review_gateway_test.go:3171` |
| 进程在远程请求后退出 | `mutation_status=possible`、shutdown 状态保护 | Unknown / Needs Reconciliation | 不盲目重试 | `failure-injection.json` F13/F14；`review_shutdown_test.go` |
| 持久 Job 进程重启 | SQLite Job + 过期 Lease 回收 | 安全 Job 重新排队 | 按任务类型 | `review_gateway_test.go:990` |
| Event Processor 重启 | Event Inbox 持久化 + route identity | Inbox 保留且只路由一次 | 0（读取任务） | `failure-injection.json` F18 |
| SQLite busy | 有限 DB budget / 失败分类 | 受控失败，不把未持久化事件当成功 | 0 | `failure-injection.json` F10 |
| Base/Doc/Task 不确定副作用 | Operation 状态 + Resource Reconciler | Base 可按唯一键自动核对；Doc/Task 多数转人工 | 不影响 GitLink | `review_operation_reconciliation_test.go`；仅离线 |

## 边界结论

1. 当前不能宣称分布式 Exactly Once；准确说法是“单实例 SQLite 部署下，应用层 best-effort single mutation”。
2. Job、Operation 和 ActionPlan 的安全恢复规则不同：只有确认没有越过非幂等远程写边界的工作才可重排。
3. ActionPlan 的 Unknown 会持久化并禁止盲目重试，但当前没有通用的自动 GitLink ActionPlan Reconciler；主要依靠 Request ID、Review 或 PR 状态人工核对。
4. Merge CI Gate 只阻止显式失败。CI unknown 不能称为通过。
