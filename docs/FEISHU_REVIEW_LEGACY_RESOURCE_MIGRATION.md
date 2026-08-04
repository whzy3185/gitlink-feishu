# 飞书 Review 旧资源安全迁移

## 目标与边界

阶段二将旧的全局 `review_collaboration_resources` 映射改造成可发现、可验证、可审计和可重启的本地迁移流程。扫描只生成计划，不复制 Remote ID；验证与应用必须由本地管理员显式执行。旧记录始终保留，整个流程不调用 GitLink 或飞书 API。

本阶段不处理仓库事件订阅、标准事件模型、Webhook Event Inbox、Webhook Replay、定时校准、Resource Outbox、Dead Letter、Worker 隔离、限流和任务合并、服务化、健康检查及监控。

## 作用域

| 资源 | 运行时默认 | 旧资源迁移要求 |
| --- | --- | --- |
| Card | Chat，固定 | 唯一 Installation、明确 Chat、人工验证；主状态写入 `chat_pr_presentations` |
| Task | Chat，固定 | 唯一 Installation、明确 Chat、人工验证 |
| Base | Chat | 必须存在显式 `chat`、`installation` 或 `disabled` 策略，且显式开启迁移 |
| Doc | Chat | 必须存在显式 `chat`、`installation` 或 `disabled` 策略，且显式开启迁移 |

运行时默认与历史迁移默认严格分离。运行时未配置的 Base、Doc 延续 Chat 投影；历史 Base、Doc 没有显式策略时进入 `needs_reconciliation`。

## 数据结构

Schema Migration 4—6 分别建立：

- `review_resource_scope_policies`：Installation 级 Base、Doc 策略；
- `review_resource_migrations`：稳定迁移计划和条件状态；
- `review_resource_migration_audit`：只追加的脱敏审计。

迁移表不保存原始 Remote ID 或 Chat ID。Chat 作用域目标只保存不可逆目标引用；实际目标在应用事务中从保留的旧源重新解析并复核哈希。

## 状态机

正常路径：

```text
discovered → needs_reconciliation → verified → applying → migrated
```

其他终态：

```text
skipped_ambiguous
skipped_disabled
failed
superseded
```

只有 `verified` 可以进入 `applying`。应用失败会回滚目标映射，再将计划标记为 `failed`；`retry` 只能将 `failed` 恢复到 `verified`。已经 `migrated`、跳过或被替代的计划不能再次应用。

## 管理命令

仓库现有 Shortcut 结构使用单个本地命令和显式动作：

```powershell
gitlink-cli feishu +review-migration --action scan --state-db .local/review-gateway.db
gitlink-cli feishu +review-migration --action list --state-db .local/review-gateway.db
gitlink-cli feishu +review-migration --action inspect --migration-id <id> --state-db .local/review-gateway.db
gitlink-cli feishu +review-migration --action policy-list --state-db .local/review-gateway.db
```

设置显式策略：

```powershell
gitlink-cli feishu +review-migration `
  --action policy-set `
  --state-db .local/review-gateway.db `
  --installation installation-a `
  --resource feishu_doc `
  --scope installation `
  --enable-migration `
  --actor local-owner
```

验证 Chat 作用域计划：

```powershell
gitlink-cli feishu +review-migration `
  --action verify `
  --state-db .local/review-gateway.db `
  --migration-id <id> `
  --installation installation-a `
  --scope chat `
  --chat-id <expected-chat-id> `
  --method operator-confirmed `
  --actor local-owner `
  --yes
```

验证不会立即应用。确认计划后单独执行：

```powershell
gitlink-cli feishu +review-migration --action apply --state-db .local/review-gateway.db --migration-id <id> --actor local-owner
```

`fixture_verified` 仅用于 Go 测试，CLI 拒绝该方法。

## Card Adoption

可信 Card 应用时重新读取旧 Remote ID 和旧 Chat，校验两者哈希、Installation、Repository、PR 和目标作用域。目标无主卡片时，采用旧 Message ID，写入 `presentation_version=1`、`card_status=active`、空内容指纹和当前 Migration ID。下次正常 PR 查询会 PATCH 该卡片建立新指纹，而不会重新 Create。

目标 Card 或兼容映射已经存在时：相同 Remote ID 作为幂等完成；不同 Remote ID 进入 `superseded`，绝不覆盖。

## 离线验证与证据

以下脚本不调用网络。提供真实数据库时，检查脚本先复制数据库及 WAL/SHM 到临时目录，再在临时副本上打开：

```powershell
.\scripts\test-feishu-review-dual-chat.ps1
.\scripts\inspect-feishu-review-scope.ps1 -DatabasePath .local\review-gateway.db
.\scripts\verify-feishu-card-mappings.ps1 -DatabasePath .local\review-gateway.db
.\scripts\verify-feishu-review-restart.ps1
.\scripts\export-feishu-review-evidence.ps1
```

Chat ID、Message ID、Remote ID 和操作者只显示 SHA-256 前 10 位。证据文件不得包含数据库、Token、Secret、Cookie 或 Authorization。

## 准确结论

旧全局资源不再自动复制到群级作用域。Card、Base、Doc、Task 已有明确作用域和安全迁移策略；无法证明归属的资源进入对账状态；可信 Card 可以在不调用远端 API 的情况下采用为群主卡片。双群隔离和重启幂等通过离线测试，但尚未完成真实双群平台验证，也不能据此声称所有历史资源已经安全迁移。
