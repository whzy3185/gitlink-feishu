# GitLink Review Event Replay

Replay 是本地、显式、可审计的 Inbox 操作：

```powershell
go run . feishu +review-event --action replay `
  --state-db .local/review-gateway.db `
  --event-id <event-id> `
  --request-id replay-20260804-001 `
  --reason "retry after subscription fix" `
  --yes
```

Replay 不重新执行 HMAC，不修改原 Event，不直接调用 GitLink 或飞书。它从原 `canonical_event_json` 创建 `source=manual_replay` 的新 Event，保存 `replay_of`，并按 Request ID 幂等。Processor 随后使用当前 Subscription Revision 路由。

Replay 前重新检查 Installation enabled、仓库仍在 allowlist、至少一个 enabled Binding。ignored 且没有 Canonical Event 的记录、disabled Installation 或已取消授权/绑定的仓库均不能 Replay。

检查命令默认不输出完整 Payload、原始 Delivery、Actor 或 Secret：

```powershell
go run . feishu +review-event --action list --state-db <db>
go run . feishu +review-event --action inspect --state-db <db> --event-id <id>
```
