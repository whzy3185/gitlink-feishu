# GitLink 飞书 Review 常驻服务架构

## 定位

`ReviewService` 是 Review Gateway 的单一生命周期所有者。它在一个进程中管理单实例锁、SQLite、管理 HTTP、飞书 Channel、GitLink Webhook、Event Processor、分类 Job Worker、Operation Planner/Worker/Reconciler、PR 校准与维护调度器。

当前是单实例可靠服务，不是多实例高可用系统。SQLite 文件和实例锁共同限定单进程所有权；多个飞书长连接也不是广播语义，因此本阶段不宣称横向扩展。

## 启动路径

1. 校验非敏感配置和 Loopback 管理地址。
2. 获取 `<state-db>.lock` 单实例锁。
3. 打开 SQLite，事务化执行 Migration 1—19。
4. 读取或建立 Configuration Revision。
5. 写入 `review_service_instances`。
6. 启动管理 Listener。
7. 按固定顺序启动必需和可选组件。
8. 必需组件全部运行后才开放 Admission 并发布 Ready。
9. 可选组件失败时进入 `degraded`；必需组件失败时进入 `failed` 并停止服务。

启动失败会逆序停止已启动组件，关闭自有数据库并释放自有实例锁。

## Listener 边界

- Admin Listener 默认 `127.0.0.1:8787`，只允许 IPv4/IPv6 Loopback。
- Public Webhook Listener 独立配置，可以部署在 TLS 反向代理后。
- `/healthz` 与 `/readyz` 不需要管理 Token。
- `/metrics` 与 `/admin/v1/*` 必须使用从 `env:VARIABLE` 加载的 Bearer Token。
- 管理 API 只提供 GET，不提供远程运维写入口。

## 数据与副作用边界

SQLite 是服务状态真源，保存事件、Job、Operation、对账、Projection、配置 Revision、实例和维护记录。飞书 Base 可以作为协作界面，但不替代本地可靠状态。

所有外部网络调用位于 SQLite 事务之外。Operation 在调用前持久化 Lease 和 Attempt；不确定副作用不会被重置为 Pending。GitLink Review 写回继续受 Installation、群来源、身份绑定、Head/Fingerprint 和显式确认约束。

## 当前未验收

本阶段仅使用临时 SQLite、Loopback、httptest 和 fake client。真实 GitLink Webhook/GET、真实双群、Card Create/Patch、Reply、Base、Doc、Task 以及完整跨平台 CI 留到阶段六。
