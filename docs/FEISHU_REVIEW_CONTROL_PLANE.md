# 只读控制平面

## 安全模型

控制平面只绑定 Loopback，使用 `Authorization: Bearer <token>`。Token 通过 `env:VARIABLE` Secret Reference 加载，不进入 SQLite、日志、URL Query 或响应，比较使用 `subtle.ConstantTimeCompare`。

除 `/healthz`、`/readyz` 外，`/metrics` 和 `/admin/v1/*` 均要求 Token。所有非 GET 管理请求返回 `405`。本阶段不提供从 Admin HTTP 重试、重放、改绑定或确认写入的能力。

## 路径

```text
/admin/v1/summary
/admin/v1/service-instances
/admin/v1/components
/admin/v1/installations
/admin/v1/bindings
/admin/v1/subscriptions
/admin/v1/config-revisions
/admin/v1/events
/admin/v1/event-routes
/admin/v1/jobs
/admin/v1/job-consumers
/admin/v1/operations
/admin/v1/operation-attempts
/admin/v1/dead-letters
/admin/v1/operation-reconciliations
/admin/v1/resource-migrations
/admin/v1/projection-status
/admin/v1/maintenance-runs
/admin/v1/audit
```

列表默认 `limit=50`，最大 200，Cursor 是不透明分页位置。每个资源只接受声明的 `status`、`installation`、`repository`、`resource_type`、`queue_class` 过滤器，不接受自由 SQL 或排序表达式。

## 脱敏

Chat/Open/Message/Remote/Operation 等标识只返回截断 Hash。Operation 不返回 `desired_json`；Event 不返回完整 Payload；Installation 不返回 Credential 值；错误统一为稳定错误码，不回显原始 SQL 错误。

## 本地 CLI

`feishu +review-service --action status|doctor` 和 `feishu +review-config --action status|revisions|inspect` 提供脱敏只读视图。维护动作是显式本地 CLI，不通过 HTTP 暴露。
