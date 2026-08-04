# GitLink Review 群级事件订阅

## 定位

仓库授权与事件订阅是两件事：`installation_repositories` 决定 Installation 可访问哪些仓库，`chat_repository_bindings` 决定群可操作哪些仓库，`review_chat_subscriptions` 才决定群是否接收事件。已有 Binding 不会隐式开启 Subscription。

订阅作用域为 `installation_id + chat_id + repository`，支持 `pulls`、`reviews`、`threads`、`merge`、`ci`。重复设置同一组配置是幂等操作；实际变更递增 `revision`，条件更新避免并发覆盖。取消最后一个事件组只将订阅设为 disabled，不删除历史记录。

## 群命令

```text
订阅 owner/repo pulls reviews merge
取消订阅 owner/repo reviews
查看订阅 [owner/repo]
设置通知 owner/repo canonical_only
设置通知 owner/repo canonical_and_notice
设置通知 owner/repo silent_refresh
设置默认仓库 owner/repo
```

只有 Binding 中明确配置的群管理员或本地超级管理员可以修改订阅、通知模式和默认仓库。管理员列表为空时默认拒绝，不创建 Job、不改变数据库。

## 通知模式

- `canonical_only`：创建或更新唯一主卡片，不另发事件通知。
- `canonical_and_notice`：维护唯一主卡片，并允许一条轻量通知。
- `silent_refresh`：只执行 GitLink GET-only 刷新，不产生飞书消息写入。

所有事件生成的 GitLink Job 均为 `refresh_review_context`、`mode=preview`、`mutates_gitlink=false`。
