# Notification Shortcut

新增 `notification` Shortcut 组，补齐 GitLink OpenAPI 中用户消息与消息设置相关接口的高层封装：

- `notification +list`：查看用户消息列表，支持 `notification` / `atme` 与已读状态过滤。
- `notification +mark-read`：按消息 ID 标记已读，支持 `--all-unread`。
- `notification +delete`：按消息 ID 删除消息。
- `notification +create-atme`：基于 Issue、PullRequest 或 Journal 创建 @我通知。
- `notification +platform-settings`：查看平台消息设置模板。
- `notification +settings`：查看用户消息设置。
- `notification +settings-update`：更新用户消息/邮件设置。

实现要点：

- 写操作均支持 `--dry-run`，可先输出 method/path/body 供用户或 Agent 确认。
- `settings-update` 会先读取当前用户设置，再合并 CLI 指定的 key，避免未指定配置被覆盖。
- 参数校验覆盖消息类型、已读状态、@我对象类型、消息 ID、布尔配置项等常见误用场景。
- 补充单元测试覆盖 HTTP method/path/query/payload、dry-run 不触发 API、设置合并保留原值等场景。
- README / README.zh-CN / Skills 文档同步补充通知消息管理示例。
