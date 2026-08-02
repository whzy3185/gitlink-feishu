# 复赛 PR Review 协作未完成项与边界

日期：2026-08-02

适用分支：`feat/round2-feishu-platform-v2`

## 1. 结论

当前项目已经完成并真实验证：

```text
飞书群消息
-> 多仓库路由
-> SQLite 持久任务
-> GitLink GET-only Review Context
-> Base + Doc + Task
-> 飞书最终回复
```

`Gitlink/gitlink-cli#431` 和 `Gitlink/forgeplus#356` 已通过真实主链路。当前剩余工作不再是证明机器人能够接收消息，而是补齐以下四个产品闭环：

1. 群内可操作的 Review 控制台；
2. 飞书身份到 GitLink 用户的可信绑定；
3. 有证据、可确认、可对账的 `common Review` 写回；
4. 真实 Agent、Webhook、企业微信和生产部署。

GitLink 写入当前保持关闭。approve、reject、merge、行级评论和 Reviewer 修改不属于当前写回范围。

## 2. 状态等级

本文统一使用以下等级，避免把测试代码当成平台上线：

| 等级 | 含义 |
|---|---|
| 真实完成 | 在真实飞书或 GitLink 环境中留有成功证据 |
| 代码完成 | 代码、合同和自动化测试存在，但未做真实平台验收 |
| 部分完成 | 主体存在，仍缺用户体验、权限或异常路径 |
| 未实现 | 尚无可执行代码或平台配置 |
| 暂不纳入 | 明确排除在当前比赛范围之外 |

## 3. 当前已完成基线

### 3.1 飞书入站与可靠性

- Channel SDK 长连接和群内 @ 机器人；
- 群、用户、管理员策略；
- `message_id` 幂等；
- SQLite Job、租约、尝试次数和结果持久化；
- 进程重启后的 queued/running 恢复合同；
- 接收回执和最终回复；
- GitLink 504 时三次重试和失败回复；
- 日志和错误信息脱敏。

### 3.2 GitLink 只读 Review

- Review Queue；
- PR 主对象、文件、patchset、Review 和线程；
- `complete`、`partial`、分段状态和结构化 fetch error；
- head SHA、source fingerprint 和 stale 判定；
- Reviewer 最后有效决定聚合；
- merged、closed、等待贡献者、等待复审等路由。

### 3.3 多仓库和公开仓库

- Installation v2；
- 群绑定多个协作仓库；
- 明确默认仓库；
- `owner/repository + PR number` 限定命令；
- 未绑定公开仓库的无凭据读取；
- 公共读取必须确认 `is_public=true`；
- 公共读取不创建 Base、Doc、Task 或协作 WorkItem。

### 3.4 飞书协作资源

- 专用 Review Base 和 14 字段 WorkItem 表；
- Base WorkItem 创建和内容指纹；
- 每 PR Review Doc 创建和追加；
- Task 创建；
- #431 的 Base 业务幂等；
- #356 的 Base、Doc、Task 首次创建。

## 4. P2 飞书协作仍未完成

### 4.1 未绑定公开仓库真实验收

状态：代码完成，真实平台待验收。

仍需使用一个不在当前三仓库绑定内、公开且存在开放 PR 的仓库，在飞书发送：

```text
@gitlink 查看 owner/public-repository PR #number
```

必须证明：

- Job 标记为 public read；
- 请求未携带 GitLink credential；
- GitLink 返回公开仓库；
- 只产生消息回复；
- Base、Doc、Task 和协作 WorkItem 新增数均为 0。

### 4.2 交互卡片真实闭环

状态：代码部分完成，真实平台待验收。

尚未证明：

- 固定卡片首次发送；
- 点击“领取”“释放”“刷新”“设置截止时间”；
- 3 秒内 ACK；
- 后台异步处理；
- 更新原卡片而不是重复发消息；
- 重复动作不产生重复协作变更；
- 长期 `message_id` 失效后的恢复策略。

当前群内主要看到文本摘要，还没有形成完整的 Review 控制台体验。

### 4.3 回复内容质量

状态：部分完成。

当前回复能够展示阶段、决策、协作状态、负责人和截止时间，但缺少比赛展示最需要的内容：

- PR 标题和作者；
- base/head 分支；
- 文件数、增删行和 patchset；
- Review 和未解决线程数量；
- 风险与 unknowns；
- source completeness；
- 建议下一步；
- GitLink、Doc、Base、Task 快捷入口。

### 4.4 Base 看板、甘特图和权限

状态：字段和数据完成，视图与权限待配置。

尚未完成：

- Review Queue 视图；
- 按负责人分组的看板；
- 截止日期甘特图；
- merged/closed 归档视图；
- Owner、Reviewer、普通成员的高级权限；
- 普通编辑者不能扩大仓库范围的约束；
- 删除、手工修改和机器人对账策略的真实验收。

### 4.5 Task 生命周期和提醒

状态：首次创建完成，完整生命周期待验收。

尚未证明：

- 认领后设置 assignee；
- follower；
- 全天截止日期；
- 原生提醒；
- PR merged/closed 后完成 Task；
- 重复同步不重复创建；
- Task 被人工删除后的恢复策略。

### 4.6 真实崩溃恢复演示

状态：代码和测试完成，真实演示待补。

需要在人为控制的测试中：

1. 保存 Job 后杀死进程；
2. 重启 Gateway；
3. 回收超时 running lease；
4. 继续执行；
5. 最终只回复一次。

## 5. P3 受控 Review 写回仍未完成

### 5.1 飞书账号与 GitLink 身份绑定

状态：数据模型完成，真实绑定为 0。

已有 `ReviewIdentityBinding`，但尚未实现：

- 自助绑定入口；
- GitLink OAuth、App Installation 或一次性绑定码；
- code 回调；
- 用户撤销；
- Token 刷新、过期和吊销；
- 用户与 App/Installation 权限交集检查。

比赛阶段可以先使用管理员核验的手工映射，但必须记录验证方式和时间，不能把飞书 `open_id` 直接当作 GitLink 用户。

### 5.2 GitLink 凭据管理

状态：仅有 `credential_ref=env:VARIABLE` 合同。

尚未完成：

- 平台级短期 Installation token；
- 安全密钥服务；
- Token 轮换；
- 每用户最小权限；
- Token 使用审计；
- 生产环境避免长期个人 Token 的方案。

### 5.3 孔明职配专用测试 PR

状态：仓库已绑定，当前没有开放 PR。

仓库：

```text
puygob236/KongMing-Job-Matching-Agent
```

仍需由协作者创建一条只修改测试文档的 PR，用于真实 common Review 验收。

### 5.4 common Review 真实写回

状态：代码门禁完成，真实平台未执行。

必须保存：

- 写入前 Review 列表；
- ActionPlan；
- actor identity；
- expected head SHA；
- source fingerprint；
- 二次确认消息；
- POST 次数；
- 写入后的 Review ID；
- 再次读取的 Review；
- 对账状态；
- 写入开关关闭记录。

任何 head 或 fingerprint 变化都必须使计划失效且 POST 为 0。网络不确定路径必须进入 `unknown_needs_reconciliation`，禁止自动重试。

### 5.5 当前明确不做的 GitLink 动作

状态：暂不纳入。

```text
approve
reject
merge
close
创建或解决行级评论
请求或移除 Reviewer
修改仓库权限
```

## 6. P4 企业微信仍未完成

状态：适配代码和合同测试完成，真实平台未验收。

已有：

- 企业微信智能机器人入站归一化；
- 多仓库只读命令；
- fail-closed chat/user allowlist；
- 本机 Review Core；
- 持久去重；
- 写类命令拒绝；
- 流式回复合同。

仍缺：

- 真实企业微信应用或智能机器人；
- 长连接或回调真实消息；
- 流式回复实际效果；
- GitLink GET；
- 同一事件去重；
- 重启恢复；
- 生产部署与权限清单。

## 7. P5 多 Agent 仍未完成

状态：协议、验证和离线编排完成，真实 Agent 未接入。

已有：

- provider-neutral Agent Invocation；
- specialist 任务拆分；
- run、task、role、head 匹配；
- Assessment 完整性校验；
- 并发、超时和确定性汇总；
- Owner 保留最终决定权。

仍缺：

- 真实 Agent Provider endpoint；
- 鉴权和凭据轮换；
- 至少一个代码 Review Agent；
- 实际 diff 证据和 findings；
- 成本、延迟和失败降级；
- Agent 输出在飞书卡片和 Doc 中的展示；
- 用户触发、取消和重新运行；
- prompt injection 与仓库不可信内容隔离验收。

## 8. Webhook 与主动刷新仍未完成

状态：签名、大小限制、去重和路由代码完成，未部署。

仍缺：

- GitLink 可配置的 Webhook；
- HTTPS 公网入口或可信反向代理；
- secret 配置；
- PR 更新、合并、关闭事件真实投递；
- 同一 delivery 去重；
- 新 patchset 自动标记旧结果 stale；
- 卡片、Base、Doc、Task 自动刷新和归档。

## 9. 生产部署与运维仍未完成

当前 Gateway 是本机进程，不是正式服务。尚未完成：

- Windows Service、计划任务、systemd 或容器；
- 开机自启；
- 健康检查；
- 自动重启；
- 结构化指标和告警；
- 日志轮转；
- SQLite 备份和恢复；
- 多实例共享队列或 leader election；
- Channel SDK 多连接随机投递下的实例协调；
- 版本升级和回滚脚本。

## 10. 比赛展示仍缺的证据

| 证据 | 当前 |
|---|---|
| 单仓库真实只读 | 已完成 |
| 第二仓库真实只读 | 已完成 |
| Base、Doc、Task 首次创建 | 已完成 |
| Base 重复执行幂等 | #431 已完成 |
| #356 Base、Doc、Task 重复同步 | 待完成 |
| 未绑定公开仓库真实读取 | 待完成 |
| 卡片动作和原卡片更新 | 待完成 |
| 真实杀进程恢复 | 待完成 |
| 账号绑定 | 待完成 |
| common Review before/after/Review ID | 待完成 |
| 真实 Agent assessment | 待完成 |
| 企业微信真实消息 | 待完成 |
| Webhook 主动刷新 | 待完成 |
| 常驻部署 | 待完成 |

## 11. 当前比赛定位

当前准确介绍应为：

> GitLink 飞书 Review Gateway 已经完成真实多仓库 PR 读取、可靠异步执行以及 Base、Doc、Task 协作投影；公开仓库可免绑定查看。受控 Review 写回、多 Agent、企业微信、Webhook 和生产部署已有代码框架，但仍需真实平台验收。

不应表述为：

```text
已经能够通过飞书批准或合并 PR
已经上线自动 AI Review
已经完成企业微信生产接入
已经支持 GitLink OAuth
已经具备多实例生产能力
```
