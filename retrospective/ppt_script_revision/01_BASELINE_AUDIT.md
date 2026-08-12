# PPT 逐字稿修改基线审计

## 1. 审计基线

- 源码仓库：`E:\GitLinkCLI-Competition\gitlink-feishu-m01`
- 分支：`feat/common-review-write-final`
- HEAD：`937b7ee42db4`
- 工作树：存在 31 项未提交变更；本审计只读取，不修改业务代码。
- 原逐字稿：`C:\Users\zyc\OneDrive\Desktop\kongming_editable_work\GitLink CLI 飞书协作导出集成-PPT逐字稿.md`
- 提交包：29 页 PPT、技术报告、部署与使用指南。
- 历史真实验收基线：`68c0547c4901a844d8a2d433decb5f18003c29f3`。
- 当前定向测试：`go test ./shortcuts/workflow ./shortcuts/feishu ./shortcuts/pr -count=1`，PASS。
- 静态测试规模：`workflow` 259、`feishu` 558、`pr` 61，共 878 个 Go `Test` 函数，分布在 66 个测试文件。该数字是函数静态计数，不等同于独立验收场景数。
- 离线故障注入：历史证据记录 20/20 PASS。
- 真实平台证据：GitLink 读取、飞书卡片/回复、领取/截止/释放、普通 Review、批准、需要修改、拒绝并关闭、合并、Head stale 为 PASS；双群、Base、Doc、Task、Webhook 受环境阻塞；Fingerprint-only stale 仅离线验证。

主要证据文件：

- `shortcuts/workflow/review_context_models.go`
- `shortcuts/feishu/review_action_plan.go`
- `shortcuts/feishu/review_write.go`
- `shortcuts/feishu/review_local_confirm.go`
- `shortcuts/feishu/review_gateway_command.go`
- `shortcuts/feishu/review_gateway_store.go`
- `shortcuts/pr/common_review_writer.go`
- `shortcuts/pr/controlled_pr_action_writer.go`
- `evidence/final/real-validation-manifest.json`
- `docs/FEISHU_REVIEW_FINAL_TEST_MATRIX.md`
- `docs/FEISHU_REAL_VALIDATION_GAP.md`

## A. 当前逐字稿章节结构

| 页 | 当前目的 | 当前核心结论 | 主要技术点 | 真实证据 | 主线 | Backup 建议 |
|---:|---|---|---|---|---|---|
| 1 | 开场 | Agent 时代需要新的 Review 协作方式 | 飞书、Gateway、GitLink | 有 | 是 | 否 |
| 2 | 问题背景 | 并行开发使 Review 更易成为瓶颈 | Agent 并行、Review 带宽 | 场景事实，无量化数据 | 是 | 否 |
| 3 | 痛点 | 信息、分工、版本和写入风险同时存在 | stale、身份、重复、网络不确定 | 有代码/测试 | 是 | 否 |
| 4 | 产品定位 | 飞书协作、GitLink 保持事实源 | 事实源分离 | 有 | 是 | 否 |
| 5 | 协作中枢 | 查询、分工和受控写入形成闭环 | Queue、Claim、ActionPlan | 有 | 是 | 否 |
| 6 | 总体架构 | 飞书、Gateway、GitLink 三层分工 | 分层、权限边界、回读 | 有 | 是 | 否 |
| 7 | 事件入口 | 消息快速持久化，后台处理 | Channel、去重、SQLite Job、Worker | 飞书链路真实；Webhook Inbox 离线 | 是 | 否 |
| 8 | 数据契约 | 底层接口归一化为稳定 Review Model | partial、fetch errors、Head、Fingerprint | 读取真实；其余代码/测试 | 是 | 否 |
| 9 | PR 展示 | 当前版本与当前有效 Review 可读 | freshness、Reviewer 聚合 | 有真实查询 | 是 | 否 |
| 10 | 协作分工 | Claim/Release/Deadline 不改 GitLink | Collaboration State | 有真实飞书验收 | 是 | 否 |
| 11 | 身份边界 | Binding 不授予权限 | Identity Binding、`/users/me` | 真实动作间接支持；显式身份门禁有测试 | 是 | 否 |
| 12 | 操作计划 | 聊天意图先冻结为 ActionPlan | scope、actor、head、fingerprint、TTL | 有真实动作 | 是 | 否 |
| 13 | 执行模式 | Local 与 Auto 的凭据边界不同 | Local、Auto、Credential | Local 真实；Auto 代码/测试，真实证据不完整 | 是 | 否 |
| 14 | 并发写入 | Lease 控制执行权，写边界只开放一次 | Lease、terminal guard、Single Mutation | 离线并发测试；真实重复零新增写入 | 是 | 否 |
| 15 | 结果确认 | HTTP 返回不足以证明远端结果 | GET Readback、Unknown、Reconciliation | 回读真实；Unknown 离线 | 是 | 否 |
| 16 | 协作资产 | Base/DocX/Wiki/Task 是可选投影 | Projection | 真实平台受环境阻塞 | 弱 | 可压缩 |
| 17 | 工程证据 | 代码、测试、故障注入和真实验收分层 | 测试矩阵、Failure Matrix | 有 | 是 | 否 |
| 18 | 上游维护 | i18n 与跨平台为可维护性服务 | Windows、CI、i18n | Windows/部分 CI 有；Linux race 待完成 | 次主线 | 可压缩 |
| 19 | Workflow | CLI 已有维护型工作流 | triage、health、pr-summary、repo-report | 代码/测试/只读远程 smoke | 次主线 | 可压缩 |
| 20 | 架构复用 | 不是一次性机器人 | CLI 注册、稳定契约、Gateway | 有 | 次主线 | 可压缩 |
| 21 | 总结 | 高效协作、受控执行、可复用工程 | P0/P1 总结 | 有 | 是 | 否 |
| 22 | 正式结束 | 汇报结束，证据页按需展开 | 边界收束 | 有 | 是 | 否 |
| 23 | 平台配置 | 应用发布和长连接 | Bot、Long Connection | 截图 | 否 | 是 |
| 24 | 权限配置 | 核心权限与可选权限分离 | 消息、卡片回调 | 截图 | 否 | 是 |
| 25 | 查询证据 | 帮助和 PR 查询真实可用 | PR GET、卡片/回复 | 截图/JSON | 否 | 是 |
| 26 | 协作证据 | Claim/Deadline/Release 真实可用 | SQLite Collaboration | 截图/JSON | 否 | 是 |
| 27 | Local 证据 | 本地凭据确认写入 | Local、`/users/me`、Readback | 真实 | 否 | 是 |
| 28 | Auto 证据 | Gateway 凭据匹配后直接执行 | Auto、Identity Gate | 代码/测试为主，真实证据不完整 | 否 | 是 |
| 29 | 最终写入证据 | 生命周期写入由 GitLink 状态回读确认 | close、merge、Readback | 真实 | 否 | 是 |

## B. 强技术主张列表

`YES` 表示存在直接证据；`PARTIAL` 表示范围有限或证据只覆盖部分路径；`NO` 表示本轮未找到对应证据。

| 技术主张 | 代码支持 | 单元测试 | 集成测试 | 真实平台验证 | 审计结论 |
|---|---|---|---|---|---|
| Review Model | YES | YES | YES | YES | `review.context/v1` 已形成稳定读取契约 |
| partial collection | YES | YES | YES | PARTIAL | 真实读取为 complete；partial 主要由 Fake Server 验证 |
| structured fetch errors | YES | YES | YES | NO | 结构存在，错误排序和脱敏有测试；无真实故障证据 |
| Head SHA | YES | YES | YES | YES | 真实 Head stale 产生零 Mutation |
| Source Fingerprint | YES | YES | YES | PARTIAL | 算法与排序有测试；Fingerprint-only stale 未真实验收 |
| ActionPlan | YES | YES | YES | YES | 多种真实动作均保存并完成计划 |
| Credential Identity | YES | YES | YES | PARTIAL | `/users/me` 门禁有测试；真实证据未独立记录调用明细 |
| Plan TTL / Expiration | YES | PARTIAL | PARTIAL | NO | 15 分钟 TTL 和执行检查存在；未找到专门的过期回归测试 |
| Lease | YES | YES | YES | PARTIAL | 并发测试和真实重复零新增写入；未做真实并发竞争验收 |
| Single Mutation | YES | YES | YES | YES | 应用层单 POST 边界；不得宣称分布式 Exactly Once |
| GET Readback | YES | YES | YES | YES | Review、close、merge 均有真实回读证据 |
| Unknown | YES | YES | YES | NO | 超时/持久化故障有注入测试；未故意破坏真实网络复现 |
| Reconciliation | PARTIAL | YES | YES | NO | 资源 Operation 有自动/人工对账；ActionPlan Unknown 主要要求人工核对，未形成完整自动闭环 |
| Inbox | YES | YES | YES | NO | `review_event_inbox` 用于 GitLink Webhook；飞书长连接消息走持久 Job admission，不走同一 Inbox 表 |
| Event Idempotency | YES | YES | YES | PARTIAL | 消息 ID/事件投递去重有测试；真实重复命令不等同同一事件重投 |
| SQLite Recovery | YES | YES | YES | PARTIAL | Job/Operation/卡片状态重启恢复有测试；正式证据未覆盖所有恢复路径 |
| Local | YES | YES | YES | YES | 普通 Review 和高风险动作存在真实验收 |
| Auto | YES | YES | YES | PARTIAL | Fake Server 验证身份匹配与单次写入；正式 JSON 未完整标明真实 Auto 链路 |
| CI Gate | PARTIAL | YES | YES | PARTIAL | 只阻止“明确失败”的 CI；真实 merge 时 CI 为 unknown 并在授权测试中放行 |

## C. 必须在强化稿中纠正的表述

1. 不把 Feishu 长连接消息描述为先进入 `review_event_inbox`；真实路径是消息归一化/策略检查后，原子去重并保存持久 Job。Event Inbox 属于 GitLink Webhook 路径。
2. 不把 Source Fingerprint 描述成完整代码 Diff 的 Hash。它是规范化 Review Context 的 SHA-256，覆盖 Head、Patchset、Review、Reviewer Summary、Thread、采集状态和结构化错误。
3. 不宣称 Exactly Once。当前是单实例 SQLite 下的应用层 best-effort single mutation：事件去重、Lease、终态保护和 POST 前持久化边界共同约束；网络不确定时依赖 Readback/Unknown。
4. 不宣称 ActionPlan Unknown 能自动恢复。当前会停止自动重试并提示按 Request ID 或 PR 状态核对；通用 Operation Reconciler 主要覆盖飞书资源操作。
5. 不宣称 CI 必须成功才允许 Merge。当前只阻断显式失败；unknown 不被表述为通过。
6. 不把 Workflow Shortcut 说成直接消费当前 Review Model。`triage`、`health`、`pr-summary`、`repo-report` 有独立实现，它们体现相同的数据归一化和 CLI 自动化方向。

