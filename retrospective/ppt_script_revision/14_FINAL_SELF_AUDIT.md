# PPT 逐字稿最终自审

## 12 项检查

1. **评委在前 3 分钟能否知道真正技术难点？** 能。第 1 页即给出身份、版本、并发写入和结果确认问题，第 6 页建立读写两条契约。
2. **第 12—15 页是否形成技术高潮？** 是。链路为 ActionPlan → Credential Boundary → Lease / Single Mutation → Readback / Unknown / Reconciliation。
3. **Head SHA 与 Fingerprint 是否解释清楚？** 是。Head 保护提交版本；Fingerprint 保护规范化审查上下文，并明确了排序和 SHA-256。
4. **ActionPlan、Idempotency、Lease、Single Mutation 是否区分？** 是。分别对应执行对象、事件去重、当前执行权和真实远程写边界。
5. **网络不确定状态是否正确？** 是。HTTP 超时不等于零写入；回读仍不确定则进入 Unknown，停止自动 Mutation。
6. **所有“真实完成”是否存在证据？** 是。只对 GitLink 读取、飞书卡片/回复、领取/截止/释放、common、approve、reject、close、merge 和 Head stale 使用真实验证口径。
7. **第 22 页之后是否明确属于 Backup？** 是。有独立章节和查看优先级，不计入正式时长。
8. **15 分钟版是否可控？** 是。页级预算合计 12 分 30 秒，余量用于翻页、停顿和设备延迟。
9. **普通评委是否能听懂主链？** 能。每个英文术语首次出现均附中文职责，并以“谁、哪个版本、谁执行、远程发生了什么”串联。
10. **技术评委深挖时是否有代码和测试可回答？** 有。已建立 Failure Matrix、Claim → Evidence Matrix 和 45 题深层 Q&A。
11. **飞书协作是否抢占过多主线时间？** 否。查询/协作只作为入口，可选 Projection 仅 15 秒，技术高潮完整保留 4 分钟。
12. **GitLink CLI 开源贡献与主体是否真实联系？** 是。复用 CLI 认证和 API Runtime，Review Model、Writer 与 Workflow Shortcut 位于同一 Go 仓库；稿件同时避免误称 Workflow 已直接消费 Review Model。

## 评分

| 维度 | 评分 | 理由 |
|---|---:|---|
| 完整度 | 9.3 / 10 | 29 页均有口播，正式与 Backup 已分离，另有计时和 Q&A |
| 技术深度 | 9.4 / 10 | ActionPlan、Fingerprint、Lease、Single Mutation、Readback 和 Unknown 形成完整状态链 |
| 工程可信度 | 9.2 / 10 | 强表述有代码/测试/真实证据映射，且主动公开边界 |
| 答辩抗压 | 9.1 / 10 | 45 题题库覆盖架构、并发、故障、Merge 与证据，但 Auto/CI 仍有可追问空间 |
| 证据完整度 | 8.7 / 10 | 真实 Mutation 与 Head stale 证据强；Auto、Fingerprint-only、Unknown 和可选 Projection 仍主要依赖离线证据 |

## 必须继续修复的问题

当前不存在会阻止稿件使用的 HIGH 风险，停止继续重写。如在比赛前补证，优先级最高的 5 项是：

1. 补一条能清楚区分 Auto 模式的真实端到端证据。
2. 补 Fingerprint-only stale 真实零写入证据。
3. 补真实网络不确定进入 Unknown 及人工核对的证据。
4. 补 Plan 15 分钟过期的专项自动化测试。
5. 若要在主讲中声称 Linux race 门禁，先补独立 CI 证据；否则保持 pending 口径。

