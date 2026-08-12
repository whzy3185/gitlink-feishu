# PPT 逐字稿答辩强化修订报告

## 1. 删除或压缩的内容

- 压缩了前半段对飞书功能的逐项介绍，避免听起来像通知机器人。
- 压缩 Base、DocX、Wiki、Task 到 15 秒，并明确它们只是可选协作投影。
- 删除了重复的 Local/Auto 安全门槛枚举，改为 Credential Boundary 对比。
- 压缩 i18n、Windows 和 Workflow Shortcut 的功能细节，将其定位为上游可维护性工程。
- 正式汇报在第 22 页结束；第 23—29 页统一改为 Backup / Engineering Evidence。

## 2. 提前的技术主线

- 第 1 页即提出“身份、版本、并发写入与远程结果确认”。
- 第 3 页把查询、协作和版本问题收束到真实写入风险。
- 第 4—5 页以 Before / After 说明降低的是上下文切换和旧版本操作风险，不是绕过维护者判断。
- 第 6 页建立读取侧与写入侧两条技术契约，使评委在前 3—4 分钟便知道技术重心。

## 3. 强化的核心技术

- **ActionPlan**：从“确认步骤”升级为冻结 Actor、Repository、PR、Head、Fingerprint、Action、Content、Request ID、有效期与状态的持久化执行对象。
- **Fingerprint**：明确为对规范化 Review Context 排序后计算 SHA-256，不误述为原始 diff hash。
- **Lease**：说明默认 2 分钟 TTL、条件更新获取执行权，以及只能在 `pre_write` 恢复。
- **Single Mutation**：明确为单实例 SQLite 下的应用层单次写入约束，不宣称分布式 Exactly Once。
- **Readback**：说明 POST 后有限 GET 回读，匹配远端状态、Review ID、Head、内容和 Request ID 才宣布完成。
- **Unknown / Reconciliation**：将网络不确定与明确失败分开；Unknown 后停止自动 Mutation，ActionPlan 当前主要人工核对。

## 4. 因证据不足而降级的表述

| 原强表述方向 | 修订后口径 |
|---|---|
| 分布式 Exactly Once | 单实例 SQLite 下的应用层单次写入约束 |
| 全部 Unknown 自动恢复 | ActionPlan Unknown 主要按 Request ID、Review 或 PR 状态人工核对 |
| Auto 已有独立真实全链路证据 | Auto 身份门禁和单次写入由 Fake Server 与 SQLite 测试覆盖 |
| 重启即全面自动恢复 | 只有未越过不可幂等远程写边界的安全工作可重排 |
| CI 成功才能 Merge | 当前 Gate 只阻止显式 failed；unknown 不得表述为通过 |
| Base/DocX/Wiki/Task 真实平台完成 | 代码与离线测试存在，真实平台验收受环境阻塞 |
| stale 是 ActionPlan 独立持久化状态 | stale 是用户可见的执行前校验结果；当前 Plan 持久化为 `failed` 并保留原因 |
| 高可用与性能收益 | 当前不宣称多实例 HA、QPS、P95、吞吐量或效率百分比 |

## 5. 新增的证据映射

- Review Model → `review_context_models.go` → workflow Fake API / golden tests → `real-gitlink-read.json`。
- ActionPlan → `review_action_plan.go` / SQLite Store → Plan matrix / local confirm tests → common、approve、reject、close、merge 真实证据。
- Head stale → `ExpectedHeadSHA` 预条件 → zero-write tests → `real-stale.json`。
- Lease / terminal guard → SQL 条件 Claim → concurrent confirm tests → duplicate zero additional mutation。
- GET Readback → controlled writers 最多 3 次 GET → strict mismatch tests → Review ID / closed / merged 真实回读。
- Unknown → `unknown_needs_reconciliation` 持久化 → failure injection / no-blind-retry tests → 真实断网复现尚缺。

完整矩阵见 `07_CLAIM_EVIDENCE_MATRIX.md`。

## 6. 尚未解决的问题

### 未实现

- 多实例高可用、分布式 Lease 与零停机迁移。
- 用户 OAuth、Refresh Token 与 per-user Credential Store。
- 通用的自动 GitLink ActionPlan Reconciler。

### 未真实验证

- Fingerprint-only stale、独立 Auto 真实轨迹、真实断网 Unknown。
- 双群隔离、Webhook、Base、DocX/Wiki、Task。

### 缺测试

- 15 分钟 Plan Expiration 有代码检查，但未找到独立精确的过期专项测试。
- Linux race 最终证据仍是 pending。

### 缺截图

- 真实 Unknown/Reconciliation、Fingerprint-only stale 和独立 Auto 全链路无完整公开截图。

### 缺代码证据

- 无可支撑分布式 Exactly Once、多实例 HA 或完整自动 ActionPlan Reconciliation 的实现。

### 仅文档设计

- QPS、P95、最大吞吐量和量化效率收益未实测，稿件不作数值声明。

## 7. 最终答辩风险

### HIGH

无。强断言已逐项对应代码、测试或真实证据，不足处已降级。

### MEDIUM

- 若评委追问 Auto 的真实独立轨迹，只能回答“离线集成测试覆盖，正式真实证据归档未完整区分”。
- CI Gate 只阻止显式失败；评委可能追问 unknown 为何不默认拒绝。
- ActionPlan Unknown 尚非通用自动对账，需主动说明人工核对边界。
- 当前仓库工作树存在多项未提交改动；本次证据是对当前工作树的审计，不能等同于已推送分支。

### LOW

- Expiration 专项测试、Linux race 证据和可选 Projection 真实平台证据尚待补齐。
- 878 是三个核心包中 Go `Test*` 函数的静态计数，不应说成 878 个独立业务场景。

## 8. Diff 摘要

| 项目 | 原稿 | 强化版 |
|---|---:|---:|
| 行数 | 209 | 176 |
| 字符数 | 9,935 | 8,917 |
| 正式页 | 1—22 | 1—22 |
| Backup | 结束后仍像正式讲解 | 23—29 明确标记为 Backup |
| 技术高潮 | 出现较晚 | 12—15 页完整状态链 |
| 正式时长 | 结构不够稳定 | 约 12 分 30 秒 |

