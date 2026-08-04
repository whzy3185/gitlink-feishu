# Review 配置 Revision

## 全局 Revision

Version 17 新增 `review_configuration_state` 和只追加的 `review_configuration_revisions`。每次批量应用执行：

1. 读取并校验 `ExpectedRevision`；
2. 规范化 Installation、Repository Allowlist、群绑定和身份绑定；
3. 读取当前 Subscription 与 Resource Scope Policy；
4. 计算非敏感、确定性的 SHA-256 Fingerprint；
5. Fingerprint 相同则不增加 Revision；
6. Fingerprint 变化时在单个事务内写全部实体、追加 Revision、更新 State 和旧兼容 Audit；
7. 任一步失败则全部回滚。

启动命令可以使用 `--expected-revision`，过期值返回 `configuration revision conflict`，防止后写覆盖先写。

## Fingerprint 边界

Fingerprint 覆盖非敏感 Installation 字段、明确仓库列表、群绑定、身份绑定 Hash、Subscription、Resource Policy、Worker 并发、Queue/Rate Limit 与服务非敏感配置。

它不包含 Secret 值、Token、Cookie、Authorization、App/Webhook/Admin Secret 或文件绝对路径。Credential/Webhook 只保留现有 `env:VARIABLE` 引用语义；Revision JSON 不保存引用对应的值。

## Entity Revision

Version 19 为以下表补齐 `revision`、`created_at`、`updated_by_hash`：

- `gitlink_installations`
- `installation_repositories`
- `chat_repository_bindings`
- `review_identity_bindings`

未变化实体保留 Revision，变化实体递增，新实体从 1 开始。Subscription 和 Resource Scope Policy 继续使用各自已有 Revision；资源策略现也支持基于旧 Revision 的比较更新。

Revision 审计有 Trigger 禁止 UPDATE 和 DELETE。管理员 API 和本地 CLI 只读展示，不回显敏感字段。
