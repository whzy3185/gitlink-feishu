# SQLite 备份、恢复、保留与 WAL 维护

## Backup

本地命令：

```text
feishu +review-service --action backup --state-db <db> --output-directory <dir> --verify
```

Backup 使用 SQLite `VACUUM INTO` 生成包含已提交 WAL 数据的一致快照，不直接复制运行中的主数据库文件。临时数据库通过 Quick Check 后计算 SHA-256，Manifest 与数据库分别 fsync，再通过 Rename 发布；失败会清理临时文件。

Manifest 为 `review.backup-manifest/v1`，包含创建时间、数据库 Hash/大小、Schema 版本、Config Revision、程序版本和源路径 Hash，不包含绝对路径、Token、ChatID 或 Remote ID。相同数据库同一时刻只允许一个 Backup；Retention Count 只删除最旧且 Manifest 一致的成功备份，保留最新成功项。

## Restore

Restore 默认只验证：

```text
feishu +review-service --action restore --backup <db> --manifest <json> --verify-only
```

真正替换要求 `--replace --yes`，并拒绝存在有效实例锁的运行中服务。替换前验证 Manifest、SHA-256、SQLite Header、Integrity Check、Schema 上限、核心表和配置状态；自动为原目标建立一致备份；临时复制和校验后原子替换；重新打开执行兼容 Migration 和 Integrity Check；失败时恢复原数据库。

## Retention

Retention 默认 Dry Run，`--apply --yes` 才删除。每表每批最多 1000 行，按 Attempt→Operation、Consumer→Job 的顺序处理。

可删除旧的终态 Job/Operation、过期 Rate Limit Bucket、已解决 Dead Letter 和已解决 Reconciliation。永不删除 Pending/Running Job、Pending/Leased/Writing/Retry/Unknown/Needs Reconciliation Operation、Open Dead Letter、未解决 Reconciliation、活跃 Presentation、Resource State 和 Resource Migration Audit。

## WAL 与完整性

支持 PASSIVE、FULL、TRUNCATE Checkpoint；周期默认 PASSIVE，优雅关闭使用 TRUNCATE。Busy 是维护竞争，不等同数据损坏。Quick Check 用于周期检查，完整 Integrity Check 用于 Backup/Restore。每次任务写 `review_maintenance_runs`，只保存路径 Hash 和脱敏错误。
