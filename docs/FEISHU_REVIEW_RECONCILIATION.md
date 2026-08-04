# GitLink Review 主卡片定时校准

Webhook 可能因网络、平台配置或服务停机而缺失。Reconciliation 仅扫描已有活跃 Canonical Card，并周期性创建 GitLink GET-only 刷新 Job，不扫描全部仓库或全部历史 PR。

候选必须同时满足：`chat_pr_presentations.card_status=active`、Canonical Message 存在、PR 非 merged/closed、Subscription enabled、Binding enabled、Installation enabled、仓库仍获授权。

运行参数：

```text
--reconciliation-interval 0    # 默认关闭
--reconciliation-interval 20m  # 推荐
```

允许区间为 5 分钟到 24 小时。同一 Installation、群、仓库、PR、时间桶只创建一个 Job；Job 使用 `refresh_review_context`、`NotifyChat=false`、`mutates_gitlink=false`。数据未变化时不 PATCH、不通知；变化时复用唯一主卡片 PATCH 状态机，不创建第二张完整卡片。

本地检查：

```powershell
go run . feishu +review-reconciliation --action list --state-db <db>
go run . feishu +review-reconciliation --action run --dry-run --state-db <db>
go run . feishu +review-reconciliation --action run --once --bindings <bindings.json> --state-db <db>
```

Cursor 持久化最近检查、入队时间、下一次检查和 Source Fingerprint。阶段三尚未实现 Resource Outbox、Dead Letter、通用 Worker 隔离、服务化健康检查、指标和备份恢复。
