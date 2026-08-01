# 复赛 P2–P5 P0 收口记录

日期：2026-07-31

## 结论

针对提交 `9d17980fe3ec1ad3e8d62611b04749a4fa32ab13` 的代码复核指出的 P0
问题，本轮完成了代码与离线合同收口。准确边界仍然是：

> P2–P5 定向代码门禁完成。飞书消息、Gateway、GitLink GET-only 和飞书最终回复主链路
> 已于 2026-08-01 通过；其余外部资源与写入验收仍需单独保存脱敏证据。

## 已关闭问题

### P3 SourceFingerprint

确认 `common Review` 前同时要求：

```text
collection_status = complete
partial = false
current_head_sha = expected_head_sha
source_fingerprint = action_plan.source_fingerprint
```

新增测试证明：head 不变但 Review 事实发生变化时，ActionPlan 进入 `stale`，GitLink
POST 次数为 0。

### P3 执行租约和恢复边界

`review_action_plans` 新增：

```text
lease_owner
lease_expires_at
attempt_count
max_attempts
reconciliation_status
```

迁移函数会为既有 SQLite 数据库补列。

状态边界：

```text
pending_confirmation
-> executing / pre_write
-> remote_write_possible
-> completed / verified
   unknown / required
```

只有 `pre_write` 阶段的过期租约允许恢复。进入 `remote_write_possible` 后，即使租约
过期也禁止重新执行 POST，必须先对账。

### P3 远端成功、本地状态失败

GitLink POST 成功但本地完成状态无法保存时：

```text
result.status = unknown_needs_reconciliation
mutated = true
automatic retry = disabled
```

系统会尽力把计划持久化为：

```text
status = unknown
reconciliation_status = required
```

新增故障注入测试证明该路径不会产生第二次 POST。

### P5 Assessment 校验

只有满足下列条件的 assessment 才计入 `AssessmentsReceived`：

```text
status = completed
completed_at 为 RFC3339
severity 属于 critical/high/medium/low
finding.summary 非空
finding.evidence 非空
finding.confidence 非空
```

失败状态或字段不完整的结果会产生 conflict，Synthesis 保持 `incomplete` 和
`human_decision_required`。

### P5 Warroom 与 P2 协作状态

`workflow +review-warroom` 增加：

```text
--collaboration <canonical-work-item-or-bundle.json,...>
```

同一 `pr_key` 会保留 P2 中的：

```text
assigned_to
collaboration_status
due_at
archived
next_step
```

截止日期同时改为 `YYYY-MM-DD` 日期语义，不再保存 UTC 零点时间。

### 企业微信 fail-closed

空 chat/user allowlist 不再表示允许全部。生产启动必须满足：

```text
至少配置一个 WECOM_ALLOWED_CHAT_IDS / WECOM_ALLOWED_USER_IDS
```

或仅在明确的开发场景设置：

```text
WECOM_ALLOW_ALL=true
```

## 自动验收

本轮门禁：

```powershell
.\scripts\verify-round2-p5.ps1
```

覆盖：

```text
飞书 Review 协作测试
ActionPlan 租约、迁移、fingerprint 和对账故障注入
企业微信 Go 适配测试
企业微信 Node sidecar 合同测试
P5 Agent assessment 与 Warroom 测试
全仓生产代码构建
P2–P5 vet
```

## 仍需真实环境完成

```text
Base / Doc / Task 首次写入和重复执行
飞书同一 message_id 去重、执行中重启和 reply 恢复故障演练
企业微信真实消息 -> sidecar -> Review Core -> GitLink -> 流式回复
GitLink 测试 PR 一次 common Review 的 before/after 和 Review ID
```

未取得测试租户、测试群和测试 PR 的明确授权前，不执行上述外部写入。
