# Claim → Evidence Matrix

Confidence 取值：`VERIFIED`（代码、测试充分）、`TESTED`（离线集成/故障测试充分）、`REAL PLATFORM VERIFIED`、`CODE ONLY`、`PARTIAL`、`NOT IMPLEMENTED`。

| Claim | Implementation | Unit Test | Integration Test | Real Evidence | Confidence |
|---|---|---|---|---|---|
| 飞书事件接收 | `review_gateway_command.go:516`、Channel SDK | Gateway tests | 真实 Channel + Job/Reply | 群聊截图、`real-card-reply.json` | REAL PLATFORM VERIFIED |
| duplicate Feishu message handling | `ReserveAndSaveJob`、stable message ID | queue tests | SQLite reopen/dedupe | 无同一事件重投专证 | TESTED |
| GitLink Event Inbox | `review_event_inbox.go`、Webhook ingress | inbox tests | processor/replay tests | Webhook blocked | TESTED |
| worker processing | 分类 Job Worker、Operation Worker | queue/worker tests | service tests | 实际 Gateway 已运行 | VERIFIED |
| restart recovery | SQLite Job/Operation/Lease 恢复 | 多个 restart tests | fault F11/F18 | 无完整真实重启验收 JSON | TESTED |
| Review Model | `review_context_models.go` | 259 个 workflow 测试中的相关用例 | Fake API + golden | `real-gitlink-read.json` | REAL PLATFORM VERIFIED |
| partial collection | `CollectionStatus`、`Partial`、`FetchErrors` | partial tests | Fake API partial | 无真实失败平台证据 | TESTED |
| Head SHA | `CurrentHeadSHA`、`ExpectedHeadSHA` | stale tests | local/gateway confirmation tests | `real-stale.json` | REAL PLATFORM VERIFIED |
| Source Fingerprint | canonical Review Context SHA-256 | order-independent tests | changed fingerprint zero-write | Fingerprint-only real test 缺失 | TESTED |
| repository binding | `ReviewGatewayBindings`、Installation scope | scope tests | multi-repo tests | 多仓库查询真实；双群 blocked | PARTIAL |
| identity binding | `ReviewIdentityBinding` | binding tests | local/auto identity tests | 真实动作由绑定登录执行，但无独立身份 trace | VERIFIED |
| ActionPlan | `review_action_plans`、v3 model | plan matrix tests | SQLite + writer tests | 多动作真实证据 | REAL PLATFORM VERIFIED |
| expiration | `ExpiresAt=now+15m`、执行时检查 | 无专门过期测试 | Claim SQL 间接覆盖 | 无 | PARTIAL |
| stale protection | Head + Fingerprint precondition | stale tests | confirmation zero-write | Head stale real PASS | REAL PLATFORM VERIFIED |
| credential identity | `/users/me` 对比 `GitLinkLogin` | writer tests | Local/Auto tests | 真实动作结果支持，但未独立记录接口 | VERIFIED |
| Local execution | `+review-confirm-local` | local tests | SQLite + Fake Server | common/approve/reject/close/merge | REAL PLATFORM VERIFIED |
| Auto execution | `controlled-action-mode=auto` + identity match | auto tests | Fake Server single write | 正式证据未完整区分 Auto | TESTED |
| Lease | SQL conditional Claim、2 分钟 TTL | lease tests | concurrent local confirm | duplicate real zero additional mutation | VERIFIED |
| Single Mutation | writer one POST + terminal/lease guard | writer tests | concurrent confirmation | 所有真实动作 actual=1，duplicate=0 | REAL PLATFORM VERIFIED |
| Review write | controlled Review writer | writer tests | Fake Server + SQLite | Review #130 | REAL PLATFORM VERIFIED |
| PR comment write | controlled journal writer（当前未提交工作树） | comment writer tests | Gateway auto comment test | 最近 PR #14 评论 #492270/#492271，不在历史 final manifest | PARTIAL |
| Approve | controlled Review status approved | tests | Fake Server | Review #131 | REAL PLATFORM VERIFIED |
| Request Changes | controlled Review status rejected | tests | Fake Server | Review #132，PR 保持 open | REAL PLATFORM VERIFIED |
| Reject & Close | native `refuse_merge` | tests | Fake Server | `real-refuse.json` | REAL PLATFORM VERIFIED |
| Controlled Merge | native `pr_merge` | tests | Fake Server | `real-merge.json` | REAL PLATFORM VERIFIED |
| GET Readback | Review/PR bounded GET | readback tests | strict mismatch tests | common/approve/reject/close/merge | REAL PLATFORM VERIFIED |
| Unknown | persisted mutation possible + no blind retry | unknown tests | failure injection | 未破坏真实网络复现 | TESTED |
| ActionPlan Reconciliation | Request ID/PR 状态人工核对提示 | terminal guards | unknown tests | 无 | PARTIAL |
| Resource Reconciliation | Operation Reconciler | reconciler tests | failure injection | Base/Doc/Task blocked | TESTED |
| CI Gate | blocks explicit failed CI only | merge tests | Fake Server | real merge CI=unknown | PARTIAL |
| Base projection | Bitable publisher/outbox | tests | failure injection | blocked | TESTED |
| DocX/Wiki projection | document publisher | tests | failure injection | blocked | TESTED |
| Task projection | task publisher | tests | failure injection | blocked | TESTED |
| Workflow triage | `triage_*` | workflow tests | remote read-only command | manual smoke in report | VERIFIED |
| Workflow health | `health_*` | workflow tests | remote read-only command | manual smoke in report | VERIFIED |
| Workflow pr-summary | `pr_summary*` | workflow tests | Fake fetch | no current real evidence file | TESTED |
| Workflow repo-report | `repo_report*` | workflow tests | partial aggregation | no current real evidence file | TESTED |
| Windows compatibility | PowerShell scripts/Go build | local tests | Windows full baseline | `full-repository-tests.json` | VERIFIED |
| i18n | i18n resolver + Chinese UX presentation | tests | package/full tests | real Chinese Feishu UX | REAL PLATFORM VERIFIED |
| Multi-instance HA | none；explicitly unsupported | NO | NO | NO | NOT IMPLEMENTED |
| User OAuth/Credential Store | none；explicitly unsupported | NO | NO | NO | NOT IMPLEMENTED |

## 强表述审查规则

- 可使用“真实平台已验证”：GitLink 读取、飞书卡片/回复、领取/截止/释放、common、批准、需要修改、拒绝并关闭、合并、Head stale。
- 只能使用“离线测试覆盖”：Fingerprint-only stale、Unknown 故障、Auto 身份路径、重启恢复、Base/Doc/Task、Webhook、资源自动对账。
- 只能使用“当前边界”：CI Gate、ActionPlan Reconciliation、PR comment（当前工作树存在但不属于历史 final manifest）。
- 禁止使用“高可用”“分布式 Exactly Once”“全部自动恢复”“所有用户 OAuth”。

