# Review 服务优雅关闭

## 触发来源

服务响应 SIGINT、SIGTERM、Windows Service Stop 和父级 Context Cancel。Shutdown 通过互斥状态保证只执行一次，重复调用安全返回。

## 顺序

1. 服务进入 `draining`，Admission 进入 `draining`，`readyz` 立即为 503。
2. SQLite Admission Gate 拒绝新群消息和 Webhook Job。
3. Claim Gate 停止新 Job 与 Operation Claim。
4. 关闭 Public Webhook HTTP。
5. 立即释放 `mutation_status=not_started` 的 Operation Lease。
6. 在 Read Drain Timeout 内等待正在运行的 GitLink GET。
7. 在 Operation Drain Timeout 内等待正在执行的 Operation。
8. 对剩余工作执行安全状态转换。
9. 关闭 Admin HTTP，取消组件 Context，逆序停止组件和飞书 Channel。
10. 执行 `TRUNCATE` WAL Checkpoint。
11. 写 Service Instance Stopped，关闭自有 SQLite，释放自有实例锁。

总 Shutdown Timeout 到达后仍会尽力执行状态保护、数据库关闭和锁释放，并向调用者返回超时错误。

## 安全状态转换

- GitLink GET/Collaboration 未完成 Job 可重新排队。
- 未开始 Operation 可回到 Pending。
- `request_started + idempotent` 可进入 Retry Scheduled。
- `request_started + reconcilable` 进入 Needs Reconciliation。
- `request_started + non_idempotent` 进入 Unknown，并设置 `requires_reconciliation=1`。
- Sending Reply 进入 Unknown，不被旧的过期 Lease 恢复逻辑盲目重发。
- GitLink ActionPlan 在 `pre_write` 前可释放；`remote_write_possible` 后进入 Unknown Needs Reconciliation。

因此，进程停止不会把可能已经产生远端副作用的请求伪装成“尚未执行”。
