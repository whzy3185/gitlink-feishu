# GitLink Review 标准事件合同

## `review.event/v1`

`NormalizedReviewEvent` 包含稳定 Event ID、受控 Source、Installation、哈希化 Delivery、仓库、PR、事件类型、Action、Head SHA、Actor Hash、发生/接收时间、Payload Fingerprint、Synthetic 与 Replay 来源。

允许的 Source：

```text
gitlink_webhook
manual_replay
scheduled_reconciliation
```

允许的 EventType 为 `pull_request`、`review`、`review_thread`、`ci`。Action 采用完整名称，例如 `pull_request.synchronized`、`review.dismissed`、`review_thread.resolved`、`ci.completed`。未知 Action 不猜测为 updated，而是以 `ignored/unsupported_action` 落 Inbox；缺少仓库或 PR 目标时使用 `ignored/missing_pr_target`。

原始 Delivery ID 不持久化。Delivery Key、Delivery Hash、Actor 均为 SHA-256 派生值。Inbox 的 `canonical_event_json` 是 Replay 的唯一业务输入；`sanitized_payload_json` 仅保留仓库、PR、Action、Head 等有界定位字段，不保存 Secret、Token、Cookie、Authorization 或原始 Actor。

## Inbox 与 Route

Inbox 状态为 `normalized → routing → routed → processed`；无有效订阅时直接进入 `processed/no_active_subscription`。临时错误最多尝试 3 次，阶段三不建立通用 Dead Letter。

Route 主键为 Event + Subscription，保存当时的 Subscription Revision 和 Notification Mode。同一 Route 只创建一个只读刷新 Job。有效路由必须同时满足 Installation、仓库授权、Binding、Subscription 和事件组均为 enabled。
