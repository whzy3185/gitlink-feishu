# GitLink 飞书协作平台当前状态与操作手册

日期：2026-08-01

适用分支：`feat/round2-feishu-platform-v2`

## 1. 一句话结论

P2.1 飞书消息 → SQLite → GitLink GET-only → 飞书回复已经真实跑通；专用 Base、每 PR Doc、Task 和 v2 多仓库绑定已在真实运行实例启用，GitLink 仍保持零写入。

当前群绑定三个真实仓库。`Gitlink/gitlink-cli` 用于既有只读验收，`Gitlink/forgeplus` 用于 GitLink 热门公共项目的跨仓库验收，`puygob236/KongMing-Job-Matching-Agent`（孔明职配）用于后续贡献者 Review 写回验收。下一步是先验证显式 `owner/repository + PR number` 路由，再为孔明职配准备一条专用测试 PR 和最小权限身份绑定。

## 2. 当前已经确认的事实

### 2.1 代码与 CI

```text
分支：feat/round2-feishu-platform-v2
已验收提交：73bd08d2a13fd7a4bc884ad12ea7de06aec8be21
相对平台 v2 基线：38 个文件，+4429/-307 行
GitHub Actions：Round 2 Review Collaboration Gate 成功
```

已实现的代码合同：

- v2 Installation、显式仓库 allowlist、一个群绑定多个仓库；
- 固定 Review 卡片创建、指纹跳过和原卡片 PATCH；
- Base WorkItem 同步、人工字段保护和重复 `unique_key` 拒绝；
- 每 PR Doc 生命周期；
- Task 创建、更新、完成和本地资源映射；
- GitLink PR Webhook 签名、大小限制、去重和路由；
- common Review 的 ActionPlan、确认、stale-head、租约和对账门禁；
- provider-neutral 外部 Agent 调用合同；
- 企业微信多仓库只读 Review Core 和持久去重；
- 飞书标识符哈希日志与不打印原始 bot/chat/message/user ID 的安全发送器。

### 2.2 真实平台验收

真实测试群已完成两次：

```text
@gitlink 查看 PR #431
→ 收到已接收回执
→ GitLink 公共 PR GET-only 成功
→ 收到最终交互卡片
→ Job completed
→ reply_status sent
→ handler_latency_ms 1
→ collection_status complete
→ GitLink writes 0
```

第二次安全回归确认 stdout 和 stderr 中没有原始 open_id、chat_id、message_id 或 event_id。详细证据见 `docs/ROUND2_FEISHU_LIVE_SMOKE_20260801.md`。

### 2.3 当前运行配置

```text
绑定 schema：feishu.review-bindings/v2
启用群绑定：1
绑定仓库：Gitlink/gitlink-cli、Gitlink/forgeplus、puygob236/KongMing-Job-Matching-Agent
Installation：1（gitlink-public）
Installation mode：collaborate
Installation credential：未配置
identity binding：0
GitLink Token：未配置
sync-feishu-resources：true
enable-gitlink-review-write：false
Webhook：未启动
Agent Provider：未配置
企业微信 Sidecar：未启动
```

因此群内现在可以通过显式仓库名查询三个仓库，并把结果同步到 Base、每 PR Doc 和 Task；多 Agent 和 GitLink 写回仍未在真实平台启用。

本机配置已由 `scripts/migrate-review-bindings-v2.ps1` 机械迁移到 `.local/review-gateway-bindings-v2.json`，该文件包含真实群绑定且受 `.gitignore` 保护，不会提交到仓库。当前 Gateway 已使用这份 v2 配置重新启动。`scripts/start-round2-feishu-gateway.ps1` 提供只读和飞书资源同步两种显式启动模式，并且有意不提供 GitLink 写入开关。

## 3. 只读诊断结果

2026-08-01 使用现有测试环境执行了只读诊断；没有创建或修改飞书资源。

| 检查 | 结果 | 结论 |
|---|---|---|
| `+app-check --remote` | 5 pass，0 fail | App ID、Secret 和 tenant token 可用 |
| `+bitable-check --remote --tables prs` | 7 pass，0 fail | 现有 Base 和 PR 表可访问，`unique_key` 查询成功 |
| `+doc-check --remote` | 5 pass，0 fail，3 skip | 已配置 Doc 和文件夹，但编辑/创建权限因只读检查未验证 |
| `+task-check --remote` | 3 pass，3 warn，1 skip | tenant token 可用；Task 创建权限未验证 |

### 3.1 已发现的配置问题

#### 现有 Base 表不是 Review Queue 表

当前 `FEISHU_PR_TABLE_ID` 对应旧 PR 汇总表，字段合同是：

```text
unique_key
repository
pr_group
risk_level
count
review_focus
recommended_action
gitlink_url
```

平台 v2 Review Publisher 需要一个专用 WorkItem 表。不要直接对旧表打开同步，否则会遇到字段不存在或字段类型不匹配。

#### 环境变量名称不一致

现有环境文件提供：

```text
FEISHU_PR_TABLE_ID
FEISHU_DOCUMENT_ID
FEISHU_FOLDER_TOKEN
```

Review Gateway 读取：

```text
FEISHU_REVIEW_TABLE_ID
FEISHU_REVIEW_DOCUMENT_ID
FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN
```

启用前必须通过新环境变量或命令参数明确映射，不能假定 Gateway 会自动读取旧变量。

#### Doc 同时配置了两种目标

现有环境同时有 Document ID 和 Folder Token。Review Gateway 只允许二选一：

- `review-document-id`：所有快照追加到一个现有文档；
- `review-document-folder-token`：每个 PR 创建并维护一份独立文档。

比赛 Review 协作推荐选择文件夹模式。

#### Task 项目和清单不是当前阻塞项

`FEISHU_TASK_PROJECT_ID` 和 `FEISHU_TASK_SECTION_ID` 当前为空，但 Gateway 的 Task v2 创建请求尚未使用这两个字段。真正需要验证的是 `task:task:write` 是否生效，以及测试租户是否允许应用身份创建 Task。

## 4. 你现在需要完成的操作

以下操作都在飞书或 GitLink 后台完成。不要把 App Secret、GitLink Token、Webhook Secret 或完整凭据粘贴到群聊、Issue、PR 或文档。

### A. 检查飞书应用权限是否已经生效

进入：

```text
飞书开放平台
→ 当前企业自建应用
→ 开发配置
→ 权限管理
```

确认以下能力对应的权限状态为“已开通”，不是“待发布”或“审核中”：

1. 接收群内 @ 机器人消息；
2. 以应用身份发送或回复消息；
3. Base/Bitable 记录读取与写入；
4. DocX 文档读取、创建和编辑；
5. Drive 文件夹访问与在文件夹内创建文档；
6. 如果本轮启用 Task：`task:task:write`。

`im.message.receive_v1` 是事件类型，不是 Scope。事件订阅和 API 权限需要分别检查。

如果新增权限仍显示“待发布”，执行：

```text
应用发布
→ 版本管理与发布
→ 创建版本
→ 确认新增权限和可用范围
→ 申请发布
→ 企业管理员审核通过
```

如果权限已经是“已开通”，不要为了本轮服务端代码变化重复发布应用。

### B. 创建专用 Base Review Queue

新建一个仅用于复赛测试的多维表格，例如：

```text
GitLink Review Queue - Round 2 Test
```

新建一张数据表，例如 `Review WorkItems`。第一轮全部按简单类型创建，避免人员/日期字段格式阻塞 smoke：

| 字段 | 第一轮字段类型 | 用途 |
|---|---|---|
| `unique_key` | 单行文本 | 必需；Publisher 查询键 |
| `pr_key` | 单行文本 | `owner/repo#number` |
| `repository` | 单行文本 | 仓库全名 |
| `pr_number` | 数字 | PR 编号 |
| `review_stage` | 单行文本 | Review 阶段 |
| `decision` | 单行文本 | GitLink 当前决定 |
| `collection_status` | 单行文本 | 数据完整性 |
| `head_sha` | 单行文本 | 当前 head |
| `source_fingerprint` | 单行文本 | stale 判断指纹 |
| `collaboration_status` | 单行文本 | 人工协作状态 |
| `assigned_to` | 单行文本 | 第一轮先保存显示值 |
| `due_at` | 单行文本 | `YYYY-MM-DD` |
| `archived` | 复选框 | merged/closed 归档 |
| `updated_at` | 单行文本 | RFC3339 更新时间 |

然后：

1. 把该 Base 授权给当前自建应用；
2. 保留你自己和 Owner 团队的编辑权限；
3. 不要创建两条相同 `unique_key`；
4. 保存 Base app token 和新表 table ID 到本地安全环境；
5. 使用变量名 `FEISHU_REVIEW_TABLE_ID`，不要覆盖旧的 `FEISHU_PR_TABLE_ID`。

### C. 准备专用 Review Doc 文件夹

1. 在测试空间建立文件夹 `GitLink Review Evidence - Round 2 Test`；
2. 将该文件夹授权给当前自建应用；
3. 确认 Owner 团队可以查看和编辑其中的文档；
4. 将 folder token 保存为 `FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN`；
5. 本轮不要同时设置 `FEISHU_REVIEW_DOCUMENT_ID`。

### D. 决定本轮是否启用 Task

如果启用：

1. 确认 `task:task:write` 已开通并在需要时随版本发布；
2. 第一轮不填写负责人，只验证创建、重复同步不重复创建、PR 关闭后完成；
3. 后续完成飞书 open_id 与 Reviewer 的映射后，再验证 assignee/follower；
4. Task 项目和清单 ID 暂时不是 Gateway 的硬依赖。

如果暂时不启用，不影响 Base 和 Doc 验收。

### E. 选择第二个 GitLink 仓库

为了真正展示多仓库能力，需要给出一个明确的第二仓库：

```text
owner/repository
```

要求：

- 你有权把它加入测试群绑定；
- 至少存在一个可读 PR；
- 第一轮可以是公开仓库，不需要 GitLink Token；
- 不使用文档中的 `Gitlink/example-repository` 占位符。

### F. GitLink 写回暂时保持关闭

只有准备测试 common Review 时再执行：

1. 使用专用测试账号创建最小权限 GitLink Token；
2. Token 只写入本地环境变量，例如 `GITLINK_INSTALL_GITLINK_TOKEN`；
3. 不在对话中发送 Token；
4. 明确一个允许产生普通 Review 的专用测试 PR；
5. 提供该测试账号的 GitLink login；
6. 由 Owner 管理员确认对应飞书用户身份绑定；
7. 最后才把 Installation 从 `collaborate` 改为 `write` 并显式打开写入开关。

批准、拒绝、行级评论、Reviewer 变更和合并仍不启用。

## 5. 你完成后只需要告诉我这些非敏感信息

```text
1. 新增飞书权限：已开通 / 待发布 / 已发布待审核
2. 专用 Review Base：已创建 / 未创建
3. 专用 Review Doc 文件夹：已授权应用 / 未授权
4. Task：本轮启用 / 暂不启用
5. 第二个 GitLink 仓库：owner/repository
6. GitLink 写回：本轮暂不启用 / 已准备专用测试 PR
```

资源 token、table ID、open_id 和 Secret 可以只保存在本机配置文件中，不需要发到聊天里。

## 6. 后续由实现侧执行的步骤

收到上面的状态后，按以下顺序推进。

### R1：迁移 v2 多仓库绑定，不产生外部写入

1. [x] 从当前 v1 文件迁移到 `feishu.review-bindings/v2`；
2. [x] 保留当前测试群和 `Gitlink/gitlink-cli`；
3. [ ] 加入第二个明确授权仓库；
4. [x] Installation 保持 `operation_mode=collaborate`；
5. [x] 不配置 GitLink Token；
6. [ ] 验证未限定仓库时的默认/歧义行为和两个限定仓库查询。

门禁：GitLink 写入 0，飞书资源写入 0。

### R2：Base 首次写入

1. 先跑只读 App/Base 诊断；
2. 只启用 Base，不启用 Doc 和 Task；
3. 对 PR #431 创建一条 WorkItem；
4. 重复执行，确认不创建第二条；
5. 人工修改 `assigned_to`、`collaboration_status`、`due_at`；
6. 再同步，确认人工字段保留；
7. 保存脱敏 record ID 哈希和前后状态。

门禁：只有专用 Base 发生写入，GitLink 写入 0。

### R3：每 PR Doc

1. 启用文件夹模式；
2. PR #431 首次同步创建一个文档；
3. 相同指纹重复执行不重复追加；
4. 新 patchset 才追加新快照；
5. 重启后继续使用同一 document ID。

门禁：不创建重复文档，GitLink 写入 0。

### R4：Task

1. 只在你选择启用 Task 时执行；
2. 首次创建一个 Task；
3. 立即重复执行，不创建第二个 Task；
4. 修改截止日期后更新同一 Task；
5. merged/closed 后完成同一 Task。

门禁：Task GUID 稳定，GitLink 写入 0。

### R5：固定卡片和多实例恢复

1. 保存同一 WorkItem 的飞书 message ID；
2. 相同指纹跳过；
3. 状态变化 PATCH 原卡片；
4. 进程重启后仍更新原卡片；
5. 验证 stdout/stderr 继续不含原始飞书 ID。

### R6：Webhook、Agent、企业微信和 GitLink 写回

这些能力分别需要公网入口、外部 Agent Provider、企业微信应用或 GitLink 写凭据，不与 R1–R5 混在一次验收中。每项单独建立证据和回滚点。

## 7. 回滚与停止条件

任何一步出现下列情况立即停止该资源同步：

- 飞书返回权限或资源不可见错误；
- Base 出现重复 `unique_key`；
- 人工字段被覆盖；
- Doc 或 Task 重复创建；
- partial GitLink 快照准备覆盖 complete 快照；
- 日志出现原始用户、群、消息或凭据；
- GitLink POST 计数不为 0；
- 远端可能成功但本地状态未知。

回滚时关闭对应启动开关并保留 SQLite 对账数据，不删除远端资源掩盖问题。

## 8. 当前推荐决策

本轮应先完成 R1–R3：

```text
多仓库只读
→ 专用 Base Review Queue
→ 每 PR 审计 Doc
```

Task、Webhook、Agent 和 GitLink common Review 放到前述证据通过之后。这样比赛演示已经能清楚展示“飞书团队协作影响 GitLink Review 效率”，同时避免把真实写回风险和平台配置问题混在一起。

## 9. 使用命令行创建专用 Review Base

不再要求手工逐列创建 `Review WorkItems` 表。当前分支提供：

```powershell
# 只生成计划，不访问飞书、不产生写入
.\gitlink-cli.exe feishu +review-base-bootstrap

# 明确确认后才创建 Base 和数据表
.\gitlink-cli.exe feishu +review-base-bootstrap --send

# 将创建结果加载到当前 PowerShell 会话
. .\.local\feishu-review-resources.env.ps1
```

默认计划创建：

```text
Base: GitLink Review Queue - Round 2 Test
Table: Review WorkItems
View: Review Queue
Fields: 14
```

安全边界：

- 默认 preview，飞书写入 0、GitLink 写入 0；
- `--send` 只创建或复用专用 Base/表，不删除默认空表或旧表；
- 创建 Base 后立即保存本地恢复文件；创建表失败时可以加载恢复文件后重跑；
- 重跑会按精确表名复用 `Review WorkItems`，避免重复建表；
- 控制台只显示资源标识符哈希，完整 app token 和 table ID 只保存在被 `.gitignore` 保护的 `.local` 文件；
- Gateway 优先读取 `FEISHU_REVIEW_BASE_APP_TOKEN`，不会覆盖历史报表使用的 `FEISHU_BASE_APP_TOKEN`。

### 9.1 2026-08-01 命令行真实验收结果

```text
专用 Review Base：创建成功
Review WorkItems 表：14 字段创建成功
PR #431 WorkItem：complete、partial=false，创建成功
相同 unique_key 再查询：恰好 1 条，第二次写入 0
Review Evidence Doc：创建成功，11 个内容块，revision=1
Task：单条低风险验收任务创建成功
Gateway：Base + 每 PR Doc + Task 同步已启用
Gateway 资源标识符：只通过子进程环境变量传递，不出现在进程参数
GitLink 写入：0
```

真实资源 ID 仅保存在本地受忽略文件和 SQLite 映射中；文档只记录脱敏哈希或数量。下一条群内 `@gitlink 查看 PR #431` 会由已启用资源同步的 Gateway 执行完整 Publisher 链路。

## 10. 多仓库真实验收与成熟集成范式

### 10.1 仓库选择

2026-08-01 使用 GitLink 官方项目列表按 `praises_count` 排序，`Gitlink/forgeplus` 以 577 个点赞、77 个复刻位于点赞榜首。其 PR #356 为开放状态，Review Context 读取结果为 `complete`、`partial=false`，可作为不属于 gitlink-cli 的公共只读样本。

孔明职配公开页面的旧标识会重定向，真实 Git 仓库标识为：

```text
puygob236/KongMing-Job-Matching-Agent
```

该仓库可以读取，但当前开放 PR 数为 0。要验证贡献者 Review 写回，必须先由仓库协作者创建一条专用测试 PR；不能用 contributor 身份替代 Review 权限验证。

### 10.2 当前群内命令

多仓库查询必须带完整仓库名，避免同一个群中的 PR 编号冲突：

```text
@gitlink 查看 Gitlink/forgeplus PR #356
@gitlink 查看 puygob236/KongMing-Job-Matching-Agent PR #<真实测试编号>
```

所有仓库均为平等作用域；PR 级命令必须显式携带 `owner/repository`，不再从群绑定推断默认仓库。当前绑定保留三个协作仓库；此外已实现 GitHub App 式公共仓库发现：Installation 和群同时显式启用 `allow_public_read` 后，任意显式 `owner/repository` 可以进行无凭据 `read_review_context`。公开状态无法由 GitLink 返回验证时拒绝，私有仓库和任何高级协作或写操作继续要求 Installation 授权。

当前暂定产品边界：

| 场景 | 是否需要仓库绑定 | 能力 |
|---|---:|---|
| 显式公开 PR 查看 | 否 | 无凭据 GET，回复消息 |
| 未限定仓库的 PR 查看 | 不适用 | 受控拒绝，要求显式 `owner/repository` |
| Review Queue、认领、截止时间 | 是 | 更新协作状态 |
| Base、Doc、Task 投影 | 是 | 创建或幂等更新飞书资源 |
| Agent、草稿、Review ActionPlan | 是 | 只对协作仓库开放 |
| GitLink Review 写回 | 是 | 还需身份、凭据、head/fingerprint 和二次确认 |

“不绑定也能查看”只免除仓库绑定，不免除群准入、完整仓库名、公开状态验证、消息幂等和频率控制。

### 10.3 参考 GitHub、Slack 与飞书的接入结构

成熟代码托管聊天集成的共同结构应用到本项目如下：

1. Installation 是授权和审计边界，不把机器人进程视为全局超级账号；
2. 群聊绑定 Installation，并保存一组平等的可管理仓库范围；PR 命令始终显式选择仓库；
3. 公共仓库允许显式路径只读发现，私有仓库必须在 Installation 范围内；
4. 飞书消息和卡片只负责触发、展示与确认，耗时 GitLink 调用进入持久异步 Job；
5. `message_id`、业务命令和目标 patchset 共同形成幂等与 stale 门禁；
6. 飞书 `open_id` 可直接用于飞书成员 mention；只有 GitLink 身份动作才必须绑定到 GitLink 用户，Token 只保存在安全凭据存储；
7. Review 写回使用“预览 ActionPlan → Owner/Reviewer 二次确认 → 再校验 head SHA 与 fingerprint → 单次写入 → 对账”；
8. Base、Doc、Task 是协作投影和审计证据，不是 GitLink Review 状态真源。

当前真实运行实例启用了绑定仓库协作读取、无凭据公共仓库发现和飞书资源同步；公共发现不会创建 Base、Doc、Task，GitLink 写回仍为 0。

GitHub App 安装、用户授权、仓库选择、最小权限和短期令牌的官方参考与本项目映射见：`docs/GITLINK_FEISHU_GITHUB_APP_REFERENCE_20260801.md`。

2026-08-02 之后的剩余差距和执行顺序分别见：

- `docs/ROUND2_REMAINING_GAPS_20260802.md`
- `docs/ROUND2_REMAINING_IMPLEMENTATION_PLAN_20260802.md`
