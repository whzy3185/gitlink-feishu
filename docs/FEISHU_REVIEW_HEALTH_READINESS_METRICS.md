# Health、Readiness 与 Metrics

## `/healthz`

`healthz` 只表达进程存活，返回 `200`。它不查询 SQLite，也不访问 GitLink 或飞书，适用于 systemd、反向代理和容器的 Liveness 探针。

## `/readyz`

`readyz` 表达服务是否能够安全接收新工作。只有以下条件同时成立才返回 `200`：

- 服务状态为 `ready` 或 `degraded`；
- Admission 为 `open`；
- 单实例锁仍归当前实例所有；
- SQLite 可在短超时内响应；
- Schema 已迁移到当前 Version 19；
- 配置已加载且有非空 Fingerprint；
- Admin HTTP 已监听；
- 所有必需组件处于 Running/Idle；
- 必需组件 Heartbeat 未过期。

Draining、Stopped、迁移缺失、SQLite 不可用、配置缺失或必需 Worker 停止时返回 `503`。请求不会访问任何外部平台。

## `/metrics`

Metrics 使用 Prometheus 文本格式，SQLite 查询预算为 400ms；数据库忙时返回最近一次成功缓存。固定 Histogram Bucket，不为每个仓库创建时间序列。

主要指标覆盖：服务 Ready/Uptime、组件状态与 Heartbeat、Handler/GitLink/飞书延迟、Event/Job/Operation 队列深度和最老年龄、重试/合并/拒绝、Unknown/Dead Letter/Reconciliation、Projection、SQLite/WAL 大小与 Busy、Backup/Retention/WAL 最近成功时间。

只允许低基数标签：

```text
component queue_class operation_kind operation_status error_class resource_type result
```

禁止输出 Repository、PR、Chat/Open/Message/Remote/Job/Operation/Event ID、用户名和主机名。`/metrics` 必须带管理 Bearer Token；未配置 Token 时不会退化为匿名访问。
