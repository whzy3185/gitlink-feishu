# GitLink 接入企业微信功能评估

副标题：参考 GitHub for Slack / Microsoft Teams 的聊天室插件模式  
评估日期：2026-07-18  
本地基线：`E:\GitLinkCLI-Competition\gitlink-cli-feishu-clean`  
评估性质：复赛功能评估，不代表已经完成企业微信接入

## 1. 评估结论

GitLink 接入企业微信可行，但应明确区分三种产品层级：

| 层级 | 能力 | 可行性 | 复赛建议 |
| --- | --- | --- | --- |
| 群消息推送 | GitLink 摘要、告警、周报、模板卡片 | 高 | 立即做 |
| 事件中继 | GitLink Webhook 实时进入企业微信，支持订阅、过滤、去重 | 中高 | 作为复赛加强或紧随其后 |
| 带身份的协作应用 | 个人提及、命令、Issue/PR 动作、智能体对话 | 中，但治理复杂 | 赛后分阶段验证 |

推荐的复赛目标不是“复刻完整 GitHub for Slack”，而是：

> 用 GitHub 聊天插件的订阅和低噪声设计，完成 GitLink 到企业微信的安全群通知；同时给出事件中继和身份化应用的清晰演进路径。

关键判断：

1. 第一版企业微信群消息推送是合适的最小交付。
2. 群 Webhook 是单向出口，不具备 GitHub 插件的用户登录、聊天室命令和权限校验。
3. 实时事件应通过 GitLink Webhook 到中继服务，再由中继渲染企业微信消息。
4. 本地 GitLink Webhook 类型尚无 `wecom`。只修改 CLI 的允许枚举并不能完成原生接入，GitLink 服务端也必须支持 payload 转换和投递。
5. 任何从企业微信反向修改 GitLink 的能力，都必须先完成仓库绑定、用户身份映射、权限复核、预览确认和审计。

## 2. 本地代码现状

### 2.1 已有基础

当前分支已经具备：

- `workflow +repo-report` 仓库分析核心。
- Owner / Contributor 两种角色摘要。
- 飞书卡片、周报和 webhook 发送。
- 默认 preview、显式 `--send`。
- webhook URL 脱敏和 mock HTTP 测试。
- GitLink 仓库 Webhook 的 list/view/create/update/delete/test/tasks。
- 本地平台无关 `CollabEvent` 契约样例。
- 企业微信 markdown/template_card 离线 payload 样例。

这些能力意味着企业微信第一版不需要重新实现 GitLink 分析逻辑，只需增加 renderer、client 和投递策略。

### 2.2 GitLink 原生 Webhook 现状

`shortcuts/webhook/webhook.go` 与本地 API 参考当前允许：

```text
gitea
slack
discord
dingtalk
telegram
msteams
feishu
matrix
jianmu
softbot
```

没有 `wecom`。

当前支持的事件：

```text
push
create
delete
issues_only
issue_assign
issue_label
issue_comment
pull_request_only
pull_request_assign
pull_request_comment
```

与 GitHub 聊天插件相比还缺：

- 正式 PR review 事件。
- CI/workflow 运行与审批事件。
- release/deployment 事件。
- discussion 事件。
- 以频道为单位的 label/branch/repository subscription。

现有 API 暴露 `secret`、投递测试和 hooktasks，但本地参考文档没有说明签名算法、签名 header、delivery ID 和重放语义。事件中继实现前必须通过 GitLink 服务端文档或受控 smoke 确认，不能直接假设与 GitHub HMAC 协议相同。

### 2.3 当前发送层缺口

飞书 WebhookClient 可以作为实现参考，但不宜原样复制：

- 默认使用 `http.DefaultClient`，没有显式总超时。
- 没有统一重试、退避和 `Retry-After` 处理。
- 没有 delivery ledger。
- 没有发送速率整形。
- 没有处理“请求超时但平台可能已收到”的不确定状态。
- 业务模型、平台 payload 和投递输出仍较紧密。

企业微信实现应先抽取平台无关 delivery contract，再分别保留 FeishuClient 和 WeComClient。

## 3. GitHub 聊天室插件值得借鉴什么

GitHub 官方 Slack 和 Teams 集成已经形成相近的产品模型。公开的 `integrations/slack` 仓库主要是说明和问题跟踪，明确指出实际集成代码不开源，因此本项目应借鉴产品行为和安全模型，而不是假设可以复制其实现。

### 3.1 三种绑定，而不是一条 Webhook

GitHub 插件同时维护：

1. **安装绑定**：应用可以访问哪些组织和仓库。
2. **频道绑定**：某个 channel 订阅哪些仓库和事件。
3. **用户绑定**：GitHub 账号与 Slack/Teams 用户身份关联。

三者分别决定：

```text
应用能读什么
消息应该发到哪里
谁可以被提及或执行动作
```

对应 GitLink + 企业微信：

```text
GitLink token / repository scope
GitLink repository <-> WeCom destination binding
GitLink username <-> WeCom userid binding
```

第一版只建立前两项中的最小范围，不做自动用户绑定。

### 3.2 频道订阅和细粒度过滤

GitHub for Slack / Teams 支持：

- subscribe / unsubscribe repository。
- issues、pulls、commits、releases、deployments 等类别。
- reviews、comments、branches、workflows 等显式 opt-in。
- branch pattern。
- label filter。
- workflow name、actor、branch、event 过滤。

核心不是命令形式，而是“每个频道只收到与它有关的变化”。

GitLink 应借鉴为统一 `SubscriptionPolicy`：

```yaml
binding_id: owner-room
repository: Gitlink/gitlink-cli
audience: owner
events:
  - pull_request.needs_review
  - issue.missing_information
  - ci.failed
branches:
  - master
labels:
  - priority:high
delivery:
  platform: wecom
  webhook_env: WECOM_WEBHOOK_OWNER_ROOM
  mode: digest
```

配置只保存环境变量名，不保存 webhook 值。

### 3.3 一个 Issue/PR 对应一个持续上下文

GitHub 在 Slack/Teams 中把同一个 Issue 或 PR 的更新放入 thread，父卡片显示最新状态；状态变化才额外广播到频道。这比“每个事件发一条新消息”更低噪声。

企业微信群消息推送不能直接等价复制 Slack thread。建议第一版模拟其语义：

- 相同 `subject_key` 在短时间窗口内聚合。
- 只在风险升级、首次需要关注或状态完成时发主消息。
- 普通评论不逐条广播。
- 周期性摘要替代高频更新。
- 完整历史保留在 GitLink，企业微信只保留最新摘要和原始链接。

如果后续使用支持交互与更新的智能机器人模板卡片，再评估更新原卡片。

### 3.4 身份关联后才做提及

GitHub 插件只有在用户登录并完成账号关联后，才根据 assignee、review request 或 mention 提及 Slack/Teams 用户。

企业微信消息推送的 text/markdown 支持 `<@userid>`，但：

- `markdown_v2` 不支持该扩展语法。
- GitLink username 与企业微信 userid 不是同一身份域。
- 显示名、邮箱或手机号不能作为隐式高风险授权依据。

因此第一版只发送群摘要，不定向提及个人。个人提及必须基于显式、可撤销的 identity mapping。

### 3.5 链接预览是独立能力

GitHub 插件会在用户分享 Issue、PR、代码片段等链接时展示预览，并对私有仓库同时校验用户登录和应用授权。

企业微信群消息推送本身不能监听聊天内容，因此无法实现任意 GitLink 链接自动展开。要做相近能力，需要：

- 自建应用回调；或
- 智能机器人长连接接收消息；并且
- 根据发送者身份重新校验 GitLink 访问权限。

复赛不应把“机器人主动发送卡片”描述成“链接 unfurl”。

### 3.6 写动作必须重新授权

GitHub 插件可以创建、评论、关闭/重开 Issue，并能处理部分 workflow/deployment 动作，但前提是：

- 用户已登录 GitHub。
- App 获得对应仓库权限。
- 用户自身拥有动作权限。
- 动作完成后返回确认卡片。

GitLink 接入企业微信也必须遵循双重约束：

```text
WeCom 用户身份有效
AND
GitLink 用户对目标仓库/资源拥有权限
```

群机器人 webhook 不满足这个条件，不能承载 GitLink 写动作。

### 3.7 智能体可以用聊天上下文，但上下文不是授权

GitHub Copilot cloud agent 可以从 Slack/Teams thread 发起任务。官方同时提醒：整段 thread 可能成为 Agent 上下文，Agent 可能代表关联用户创建 Issue 或 PR。

对 GitLink 的启示：

- 默认只读。
- 展示智能体准备使用的仓库和分支。
- 在写入前显示 action preview。
- 不把群成员能看到某段对话等同于其能修改仓库。
- 对话内容进入 GitLink Issue/PR 前需要明确提示和必要的脱敏。

## 4. GitHub 模式与企业微信能力映射

| GitHub 聊天插件能力 | 企业微信群消息推送 | 企业微信自建应用 | 企业微信智能机器人长连接 | GitLink 建议 |
| --- | --- | --- | --- | --- |
| 仓库通知 | 支持 | 支持 | 支持 | P0 |
| 模板卡片 | 支持 | 支持 | 支持且可交互/更新 | P0 先静态 |
| channel/repo subscription | 只能在外部配置 | 可通过应用配置 | 可通过对话配置 | P0 先 CLI 配置 |
| 事件/标签/分支过滤 | 发送前过滤 | 发送前过滤 | 发送前过滤 | P0 |
| Thread 聚合 | 无直接等价 | 能力有限 | 可用更新卡片模拟 | P0 用 digest/ledger |
| 用户提及 | 可 @userid，但需映射 | 支持定向 | 支持单聊/群聊 | P2 |
| 账号登录/绑定 | 不支持 | 可设计 OAuth/绑定流程 | 可设计绑定流程 | P2 |
| 链接自动展开 | 不支持监听 | 可接收消息后处理 | 可接收消息后处理 | P2 |
| 聊天命令 | 不支持 | 回调可实现 | 适合 | P2/P3 |
| 创建/管理 Issue | 不应支持 | 技术可行 | 技术可行 | P3，先预览确认 |
| Agent thread context | 不支持 | 可做 | 最适合 | P3 独立 bridge |

## 5. 三条技术路径

### 路径 A：gitlink-cli 直接发送企业微信群消息

```text
GitLink read-only fetch / local fixture
-> workflow
-> CollabEvent
-> SubscriptionPolicy
-> Digest
-> WeComRenderer
-> WeComClient
-> group message push webhook
```

适合：

- Owner 日报/周报。
- 高风险 PR/Issue 摘要。
- Contributor 群级下一步。
- 复赛双平台现场演示。

优点：

- 改动局限在 CLI。
- 可以复用现有 workflow 和 digest。
- 默认 preview，安全边界清楚。
- 无需部署常驻公网服务。

局限：

- 不是实时 GitLink 事件流。
- 不能在群内 subscribe/unsubscribe。
- 不能读聊天消息、展开链接或执行回调。
- 不能做可靠的个人身份和权限链路。

结论：**P0，立即实现的最佳路径。**

### 路径 B：GitLink Webhook 到中继服务，再到企业微信

```text
GitLink repository webhook
-> HTTPS receiver
-> verify signature and delivery identity
-> queue
-> normalize CollabEvent
-> subscription/filter/dedupe
-> WeCom renderer/client
-> delivery ledger
```

适合：

- 接近 GitHub 插件的实时通知。
- 多仓库、多群绑定。
- label/branch/event 过滤。
- 失败投递、重放和统一监控。

必要条件：

- 核实 GitLink webhook 的签名算法和 header。
- 核实投递 ID、重试和 redelivery 语义。
- 接收端快速返回 2xx，异步处理。
- 保存 source delivery ID 和目标 delivery 状态。
- 不在 URL 中暴露中继认证凭证。

如果 GitLink 暂无稳定签名协议，中继只能在受控环境技术验证，不能标记为生产安全路径。

结论：**P1，价值高，但需要 bridge 和 GitLink 服务端契约。**

### 路径 C：GitLink 服务端原生增加 `wecom` Webhook 类型

理想调用：

```powershell
gitlink-cli webhook +create `
  --owner Gitlink `
  --repo gitlink-cli `
  --type wecom `
  --url-env WECOM_WEBHOOK_URL `
  --events pull_request_only,issue_comment
```

但当前 CLI 的 `webhook +create` 会把请求交给 GitLink 服务端。要真正支持 `wecom`，至少需要：

1. GitLink 服务端允许 `wecom` 类型。
2. 服务端把 GitLink 事件转换成企业微信合法 payload。
3. 安全保存和脱敏企业微信 webhook。
4. 处理企业微信业务返回码和频率限制。
5. 提供投递记录、失败原因和测试事件。
6. CLI、API 文档和服务端同时更新。

只在 `allowedWebhookTypes` 增加字符串会产生“客户端接受、服务端拒绝或错误投递”的假集成。

结论：**P2，需要 GitLink 平台侧协作；可作为官方长期方向。**

## 6. 推荐的复赛功能范围

### 6.1 P0 必做

建议新增命令：

```text
gitlink-cli wecom +check
gitlink-cli wecom +bot-test
gitlink-cli wecom +notify
gitlink-cli wecom +owner-digest
gitlink-cli wecom +contributor-digest
```

共同规则：

- 默认 preview。
- 真实发送必须显式 `--send`。
- 支持 `markdown` 和 `template_card`。
- 使用 `WECOM_WEBHOOK_URL`，输出中删除完整 query/token。
- 默认只允许 HTTPS。
- 显式 HTTP 总超时。
- 识别 HTTP 状态和企业微信业务返回码。
- 每个 destination 不超过 20 条/分钟。
- 同一 `event_id + destination + render_version` 不重复发送。
- 不盲目重试结果未知的 POST。
- 没有 identity mapping 时不 @ 个人。

### 6.2 P0 功能加强

在现有 `CollabEvent` 上增加：

```text
subject_key
source_delivery_id
source_fingerprint
audience
data_scope
unknowns
render_version
delivery_policy
```

增加：

- 本地 subscription fixture。
- 本地 delivery ledger。
- 事件合并窗口。
- Owner Top 3-5。
- Contributor 状态机预览。
- 飞书/企业微信语义一致性 golden test。

### 6.3 P1 事件中继原型

只读原型应做到：

- 接收固定 GitLink webhook fixture。
- 校验已明确的签名协议。
- 记录 delivery ID。
- 立即 ack，异步处理。
- 重复投递只产生一次 CollabEvent。
- 只发送到测试群。
- 支持失败队列和手工 replay preview。

不做：

- GitLink 写入。
- 自动用户绑定。
- 生产多租户。
- 无签名公网接收。

### 6.4 P2/P3 后续

- 自建应用安装和可见范围管理。
- GitLink username 与 WeCom userid 显式绑定/撤销。
- PR review request 定向提醒。
- GitLink 链接预览。
- 群内 subscribe/list/unsubscribe。
- Issue 创建、评论、关闭的 preview/confirm/audit。
- 智能机器人长连接 Agent。

## 7. 推荐目录边界

```text
internal/collab/
  event.go
  subscription.go
  digest.go
  ledger.go
  policy.go

shortcuts/wecom/
  wecom.go
  options.go
  markdown.go
  template_card.go
  client.go
  render.go
  diagnostics.go
  *_test.go

cmd/gitlink-wecom-bridge/       # P1，独立进程
  receiver.go
  verify.go
  queue.go
  normalize.go
  delivery.go

skills/gitlink-wecom/
  SKILL.md
```

边界：

- `internal/collab` 不引用飞书或企业微信 payload 类型。
- renderer 不读取 GitLink API。
- WeComClient 不判断业务风险。
- bridge 不进入普通 CLI 命令的生命周期。
- GitLink 写动作不进入 P0/P1。

## 8. 功能优先级评估

评分：价值 1-5；工作量 S/M/L/XL；风险 1-5。

| 功能 | 价值 | 工作量 | 风险 | 建议 |
| --- | ---: | :---: | ---: | --- |
| 企业微信 preview + bot test | 4 | S | 1 | P0 |
| Owner / Contributor 群摘要 | 5 | S-M | 1 | P0 |
| markdown/template_card 双 renderer | 4 | S-M | 1 | P0 |
| destination/repo subscription | 5 | M | 2 | P0 |
| event ledger、合并和限速 | 5 | M | 2 | P0 |
| 定时 PR review reminder | 4 | M | 2 | P1 |
| GitLink webhook bridge | 5 | L | 3 | P1 |
| GitLink 服务端原生 wecom type | 4 | L | 3 | P2 |
| 自建应用个人消息 | 4 | L | 4 | P2 |
| 链接自动预览 | 3 | L | 4 | P2 |
| 群内 Issue/PR 写动作 | 4 | XL | 5 | P3 |
| 长连接智能体 | 4 | L-XL | 4 | P3 spike |

## 9. 安全与可靠性评估

### 9.1 企业微信消息推送

官方当前说明：

- 支持 text、markdown、markdown_v2、image、news、file、voice、template_card。
- 每个消息推送不超过 20 条/分钟。
- text/markdown 可以使用 `<@userid>`；markdown_v2 不支持。
- webhook URL 本身是敏感凭证。

要求：

- 日志不能输出 query 中的 key。
- `WECOM_WEBHOOK_URL` 不进入仓库配置或命令历史示例。
- 非测试环境默认 `--send=false`。
- 单群限速和全局并发限制分别实现。

### 9.2 自建应用与回调

自建应用消息使用 `access_token`、应用 `agentid` 和接收用户/部门范围。回调配置需要开发者提供 URL、Token、EncodingAESKey。

要求：

- token 缓存和过期处理。
- 回调验签与消息解密。
- 应用可见范围不能代替 GitLink 仓库权限。
- 删除/撤销身份映射后立即停止个人消息。

### 9.3 智能机器人长连接

当前官方文档说明：

- 使用 BotID 和长连接专用 Secret。
- 无需固定公网 IP，也无需自行处理回调加解密。
- 可以接收单聊、群聊 @、卡片事件并主动推送。
- 建议 30 秒心跳。
- 同一机器人同时只能保持一个有效连接，新连接会踢掉旧连接。
- 同一会话回复和主动推送合计 30 条/分钟、1000 条/小时。
- 流式消息从首次发送起有 10 分钟结束限制。

要求：

- 独立 bridge 进程。
- 主备切换，不做多实例同时消费。
- msgid/req_id 幂等。
- 断线重连不重复执行 GitLink 动作。
- Agent 全程只读，直到动作网关单独通过评审。

### 9.4 Webhook 中继

参考 GitHub 官方 Webhook 安全最佳实践，但不要假设 GitLink 协议与 GitHub 相同：

- 只订阅必要事件。
- HTTPS。
- secret/HMAC 验证。
- 固定时间比较签名。
- 快速 ack，队列异步处理。
- delivery ID 防重放。
- 支持失败投递检查和受控重放。

GitHub 官方建议 Webhook 接收端在 10 秒内返回 2xx，并使用 `X-GitHub-Delivery` 去重；GitLink 中继必须寻找 GitLink 自身的等价契约，没有时需要平台侧补齐。

## 10. 验收清单

### P0

- [ ] 同一 fixture 可渲染飞书卡片和企业微信模板卡片。
- [ ] 两个平台的仓库、风险数、证据、未知项和下一步语义一致。
- [ ] 默认运行不发消息。
- [ ] `--send --dry-run` 直接报错。
- [ ] webhook URL/query 完全脱敏。
- [ ] HTTP timeout 有测试。
- [ ] HTTP 429/5xx 和企业微信非零业务码有测试。
- [ ] 20 条/分钟限制有确定性测试。
- [ ] 相同 event_id 重复输入不重复发送。
- [ ] Contributor 未映射时不生成个人 @。
- [ ] 所有卡片保留 GitLink 原始链接和数据范围。

### P1 bridge

- [ ] GitLink 签名协议有文档或真实 smoke 证据。
- [ ] 无效签名拒绝。
- [ ] ack 与业务处理分离。
- [ ] 重复 source delivery 只生成一次事件。
- [ ] 失败投递可查询。
- [ ] replay 默认 preview。
- [ ] GitLink/WeCom 凭证不进入日志和队列明文。
- [ ] 测试群有独立 kill switch。

### P2/P3

- [ ] 账号绑定可撤销。
- [ ] 每次写动作重新校验 GitLink 权限。
- [ ] 写入前显示目标仓库、资源、动作和内容。
- [ ] 用户明确确认。
- [ ] 操作结果和失败原因可审计。
- [ ] 群聊上下文进入 Issue/PR 前提示并脱敏。

## 11. Go / No-Go

### 可以进入复赛实现

- `wecom` 第一版只做消息推送。
- 复用平台无关事件和现有 workflow。
- 默认 preview。
- 有测试群和消息额度控制。
- 不做个人身份猜测和 GitLink 写入。

### 应暂停

- 计划只在 `allowedWebhookTypes` 添加 `wecom`，没有 GitLink 服务端方案。
- 无法说明 webhook 凭证存放位置。
- 需要把真实 webhook 写入命令、fixture 或仓库。
- 没有幂等和限速就接实时事件。
- 把群成员身份直接映射成 GitLink 用户。
- 把普通评论或每次 push 全量广播到群。
- 用长连接智能体替代 P0 基础能力。

## 12. 最终建议

复赛建议采用：

```text
P0: gitlink-cli wecom 消息推送
  + 统一 CollabEvent
  + Owner/Contributor 群摘要
  + subscription fixture
  + event/delivery ledger
  + 双平台 golden test

P1: GitLink webhook bridge 只读原型
  + 签名和 delivery contract 验证
  + 队列、过滤、去重、重放预览

P2/P3: 自建应用、身份、链接预览和智能体
  + 独立授权与审计评审
```

对外定位：

> GitLink 企业微信集成不是另一个流水消息机器人。它借鉴 GitHub 聊天插件的订阅、聚合和身份边界，把真正需要处理的 GitLink 协作变化带到企业微信；GitLink 始终是事实和权限源。

## 13. 官方资料

### GitHub 聊天集成

- [GitHub for Slack 安装与能力](https://docs.github.com/en/integrations/how-tos/slack/integrate-github-with-slack)
- [GitHub for Slack 使用、提及、thread 与链接预览](https://docs.github.com/en/enterprise-cloud@latest/integrations/how-tos/slack/use-github-in-slack)
- [GitHub for Slack 公开说明仓库](https://github.com/integrations/slack)
- [GitHub for Teams 安装与能力](https://docs.github.com/en/integrations/how-tos/teams/integrate-github-with-teams)
- [GitHub for Teams 使用与功能](https://docs.github.com/en/enterprise-cloud@latest/integrations/how-tos/teams/use-github-in-teams)
- [GitHub for Teams 通知过滤](https://docs.github.com/en/integrations/how-tos/teams/customize-notifications)
- [GitHub for Teams 创建 Issue](https://docs.github.com/en/integrations/tutorials/teams/create-issues)
- [GitHub for Teams 管理 Issue/PR](https://docs.github.com/en/integrations/tutorials/teams/manage-issues)
- [GitHub for Teams PR review reminder](https://docs.github.com/en/integrations/how-tos/teams/schedule-reminders)
- [GitHub App 最小权限](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app)
- [GitHub Webhook 最佳实践](https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks)
- [GitHub Webhook 签名验证](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)
- [Copilot cloud agent 接入 Slack](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/integrate-cloud-agent-with-slack)

### 企业微信

- [消息推送配置说明](https://developer.work.weixin.qq.com/document/path/91770)
- [发送应用消息](https://developer.work.weixin.qq.com/document/path/90236)
- [接收消息和事件](https://developer.work.weixin.qq.com/document/path/94670)
- [回调配置](https://developer.work.weixin.qq.com/document/path/90930)
- [智能机器人长连接](https://developer.work.weixin.qq.com/document/path/101463)

### GitLink

- [GitLink 平台](https://www.gitlink.org.cn/Gitlink/forgeplus)
- [gitlink-cli](https://www.gitlink.org.cn/Gitlink/gitlink-cli)

