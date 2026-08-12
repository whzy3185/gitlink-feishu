# 能力优先版逐字稿修订报告

## 1. 修订原则

原“答辩强化版”保持不变，本轮另建副本。新版的沟通任务是：

> 让评委先相信这套系统能解决真实 PR Review 协作问题，再用少量工程机制证明其可信。

正式 1—22 页优先使用“场景 → 用户能做什么 → 解决什么 → 一句可信机制”。连续底层机制解释已移入 23—29 页 Backup，并继续由 Failure Matrix、Claim → Evidence Matrix 和 Deep Q&A 承载。

## 2. 正式稿的新价值结构

| 内容 | 目标占比 | 主要页面 |
|---|---:|---|
| 场景与痛点 | 约 25% | 1—5 |
| 产品能力与使用方式 | 约 40% | 6—13 |
| 实际效果与真实证据 | 约 25% | 14—21 |
| 底层实现 | 约 10% | 正式稿中只保留可信性结论；细节在 Backup |

## 3. 逐页转换说明

| 页码 | 原表达重心 | 能力优先表达 | 用户获得的价值 | 保留的可信证明 |
|---:|---|---|---|---|
| 1 | 身份、版本、并发写入 | 在飞书查看、协作并受控推动 PR | 一开场就知道产品能做什么 | GitLink 仍是事实与权限判断方 |
| 2 | Review 压力与安全门槛 | Agent 并行开发使 Review 压力更集中 | 视角从系统问题转为维护者工作压力 | 不虚构 PR 增长比例 |
| 3 | 四类工程瓶颈 | 状态散、分工乱、结论过期、写操作谨慎 | 评委可以对照真实 Review 工作 | 便利性不代替安全边界 |
| 4 | Review Model 与 ActionPlan 概念 | 飞书中可查询、分工和发起 Review 操作 | 知道飞书不再只是通知入口 | GitLink 保存最终仓库事实 |
| 5 | 抽象 Before / After 链路 | 用户一次 Review 的切换步骤前后对比 | 减少切换和状态割裂 | 不替代维护者代码判断 |
| 6 | 读取侧/写入侧技术契约 | 看得见、分得清、推得动、控得住 | 一页获得完整能力全景 | 身份、版本、重复写与结果受控 |
| 7 | Long Connection → SQLite → Worker | 群里一条命令查询任意公开 PR | 查询入口留在团队协作场景 | 重复消息去重，安全持久任务可继续处理 |
| 8 | Review Model 四个职责 | 一张视图看清变更、Review 和下一步 | 不必先打开 GitLink 才理解 PR | 数据不完整时明确提示并阻止高风险操作 |
| 9 | Head SHA / Fingerprint 算法 | 当前版本与历史 Review 分开 | 不把旧批准当成新版本结论 | 执行前重新确认 PR 与 Review 上下文 |
| 10 | SQLite Collaboration Model | 领取、释放和截止日期组成团队 Review 队列 | 共享负责人与处理进度 | 真实飞书验收已覆盖 |
| 11 | Binding → Credential → Permission | 协作负责不等于仓库授权 | 享受协作便利时不混淆权限 | 执行前核对真实 GitLink 身份，服务端最终判权 |
| 12 | ActionPlan 字段列表 | 聊天文本不会绕过明确操作上下文和执行门禁 | Local 先查看并本地确认；Auto 只在显式启用且身份匹配时继续 | 意图先被固化为可校验操作 |
| 13 | Local / Auto Credential Boundary | 两种方式适配个人凭据与固定 Gateway 凭据场景 | 团队可在安全边界和便利之间选择 | 两者执行前均核对真实 GitLink 身份 |
| 14 | ActionPlan / Idempotency / Lease / Mutation 区分 | PR 变化、重复点击和多人并发时自动停止或限制 | 旧操作和重复确认不轻易越过写边界 | 执行前重读 PR，并发只让一个执行过程继续 |
| 15 | Readback / Unknown / Reconciliation 状态 | 网络超时后重读 GitLink，不盲目重试 | 用户看到远端真实结果 | 已完成/失败/暂时无法判断三种结果分开处理 |
| 16 | Projection 实现边界 | 可选协作资产 | 核心 Review 不被外部资产配置绑架 | 真实平台验收阻塞保持原证据等级 |
| 17 | 测试数与 Failure Matrix | 真实查询、协作、Review、Approve、Request Changes、Close、Merge | 证明不是 UI Demo | 真实平台证据 + 878 个 Test 函数 + 20 个故障场景 |
| 18 | i18n / Windows 门禁 | 中文命令与跨平台使用体验 | 降低上游用户和维护者门槛 | Windows 证据已有，Linux race 仍 pending |
| 19 | Workflow Shortcut 实现列表 | triage、health、pr-summary、repo-report 对应维护自动化 | 展示项目进入 GitLink CLI 的实际延展方向 | 已注册、有测试，triage/health 有只读 smoke |
| 20 | 核心技术资产列表 | 新协作入口和 Agent 可继续复用同一 GitLink 边界 | 项目不停留在一次性机器人 | CLI 读取、Review 视图和受控写入可独立复用 |
| 21 | Controlled Execution 技术名词总结 | 维护者、贡献者、团队、GitLink 四个角色的价值 | 结尾落回人和平台的真实收益 | GitLink 始终保留权限和最终事实 |
| 22 | 技术流程式收尾 | 看见、分工、推动 Review，以 GitLink 事实为准 | 评委离场时能复述产品价值 | 明确 Backup 存在完整技术证据 |
| 23 | 飞书应用与长连接 | 保留为部署与持久任务 Backup | 被问部署时可深入 | 说明去重、持久和有限恢复边界 |
| 24 | 权限与卡片回调 | 保留最小权限与 partial 读取 Backup | 被问飞书权限时可深入 | 关键数据 partial 时零高风险写入 |
| 25 | 真实 PR 查询 | 补充 Head / Fingerprint 技术解释 | 主稿不被算法占用 | 真实读取证据与 SHA-256 实现口径 |
| 26 | 真实协作状态 | 补充 Identity Binding 与 `/users/me` 边界 | 主稿只讲用户能感知的权限区别 | Claim 零 GitLink Mutation |
| 27 | Local 真实证据 | 完整 ActionPlan 和执行链路留在 Backup | 满足技术评委深挖 | Local 真实证据最完整 |
| 28 | Auto 真实链路表述 | 加强 Lease / Single Mutation / Exactly Once 边界 | 展示并发设计但不阻塞主稿 | Auto 仅离线证据，不扩大宣称 |
| 29 | Mutation + Readback | 补充 Unknown 与人工 Reconciliation 边界 | 回答网络不确定和重复写入问题 | 真实 Mutation / duplicate zero / GET 回读证据 |

## 4. 底层技术语言如何转换为用户语言

| 底层表达 | 正式口播表达 |
|---|---|
| Review Model | 群里直接获得统一、清晰的 PR Review 视图 |
| partial / fetch errors | 数据没读完时明确提示，不用残缺数据做高风险操作 |
| Head SHA / Source Fingerprint | PR 代码或 Review 上下文变化后，旧操作失效 |
| ActionPlan | 高风险操作先看清内容并确认，再真正执行 |
| Identity Binding + `/users/me` | 执行前确认飞书用户对应的真实 GitLink 身份 |
| Idempotency / Lease / Single Mutation | 重复消息、连续点击和多人并发不会轻易产生重复写入 |
| GET Readback | 写完后重新读取 GitLink，确认最终真实状态 |
| Unknown / Reconciliation | 结果无法确认时停止继续操作，转入后续核对 |
| SQLite Job / restart recovery | 协作任务可持续处理，但只恢复未越过不可幂等写边界的安全工作 |

## 5. 证据等级保留情况

### 可使用“真实平台证据已覆盖”

- GitLink PR 读取。
- 飞书卡片 create / patch / reply。
- Claim / Deadline / Release。
- 普通 Review、Approve、Request Changes、Reject & Close、Merge。
- Head changed 时旧计划零写入。

### 只能使用“离线测试覆盖”或“受环境阻塞”

- Auto 独立真实链路。
- Fingerprint-only stale。
- 真实网络故障进入 Unknown 及后续核对。
- Webhook、双群隔离、Base、DocX、Wiki、Task。
- Linux race 最终门禁。

## 6. 正式口播检查

- 第 1—6 页：场景、产品定位、Before/After 和四项能力。
- 第 7—11 页：从用户查询一个 PR，到看懂 Review、分工、明确权限边界。
- 第 12—15 页：四个真实操作场景，不再是底层协议教学。
- 第 16 页：可选投影压缩到 15 秒。
- 第 17 页：集中售卖“真实能力有证据”。
- 第 18—20 页：工程成熟度、GitLink CLI 贡献和复用方向。
- 第 21 页：按维护者、贡献者、团队、GitLink 四个角色收束。
- 第 22 页：正式结束，之后全部为 Backup。

## 7. 结论

本轮没有删除工程深度，而是改变它的出场位置：

- 正式答辩负责说清“为什么值得用”。
- Backup 和 Deep Q&A 负责回答“为什么它可信”。
- Failure Matrix 和 Claim → Evidence Matrix 继续作为强表述的后台证据索引。
