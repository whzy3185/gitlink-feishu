# Milestone shortcut

新增 `milestone` Shortcut 组，补齐 GitLink 里程碑 OpenAPI 的常用操作封装：

- `milestone +list`
- `milestone +create`
- `milestone +view`
- `milestone +update`
- `milestone +delete`
- `milestone +close`
- `milestone +reopen`

实现要点：

- 支持列表筛选、分页、排序，以及详情页关联 Issue 过滤参数。
- 写入时将 CLI 参数 `--due-date` 映射为 API 字段 `effective_date`。
- `+update` 在没有任何变更字段时直接报错，避免发送空更新。
- `+close` 和 `+reopen` 使用 GitLink 的 milestone 状态更新接口。
- 补充单元测试覆盖各命令的 HTTP 方法、路径、查询参数和 payload。
