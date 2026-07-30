# GitLink CLI 复赛 PR Review 协同 P1.1 门禁收口

实施日期：2026-07-30

基线：`feat/round2-review-collaboration-p1` @
`9c22c180d679eb8e49dace441f7a57b23f19840c`

实现分支：`fix/round2-review-collaboration-p1-gate`

## 1. 结论

P1.1 修复 P1 代码评估中发现的输入合同和聚合问题，使 Review Context 可以作为
后续飞书协作适配层的稳定只读输入。本轮仍不是 P2，不创建飞书资源，不从飞书
触发 GitLink 操作，也不执行任何 GitLink 写入。

已收口：

1. 同一 Reviewer 的多次当前 Review 只采用最后一条可确定顺序的有效决定。
2. 输出 Reviewer 级摘要，保留当前、过期和未知 Review 数量。
3. Review 线程保留正文、作者 ID、Review ID、路径和行位置。
4. 输出结构化采集状态、分段状态与请求错误。
5. PR 主对象读取失败时整体失败，不再生成貌似完整的成功结果。
6. 使用 patchset versions 获取当前 head SHA 和版本 ID。
7. 明确路由 `merged`、`closed`、`waiting_for_re_review` 与歧义人工复核。
8. 使用真实 GitLink 公共仓库完成 GET-only 合同验证，并保存脱敏 fixture。

## 2. Reviewer 最后有效决定

聚合键优先使用 GitLink Reviewer ID；ID 缺失时使用登录名；两者都缺失时不把
不同匿名记录合并。

每名 Reviewer 的当前有效决定按以下顺序选择：

1. 优先比较 `updated_at`。
2. 其次比较 `created_at`。
3. 时间均不可用时，只在 ID 都是数字时比较数值 ID。
4. 多条冲突决定无法可靠排序时，结果为 `unknown`，交给人工复核。

因此：

```text
同一 Reviewer: rejected -> approved  => approved
同一 Reviewer: approved -> rejected  => rejected
不同 Reviewer: approved + rejected   => rejected 仍阻断
冲突记录无法排序                       => unknown，不猜测
```

输出新增：

```text
reviewer_summaries[].reviewer_key
reviewer_summaries[].actor_id
reviewer_summaries[].actor
reviewer_summaries[].latest_effective_review
reviewer_summaries[].current_decision
reviewer_summaries[].decision_order_known
reviewer_summaries[].current_count
reviewer_summaries[].outdated_count
reviewer_summaries[].unknown_count
```

## 3. 数据完整性合同

顶层新增：

| 字段 | 语义 |
|---|---|
| `collection_status` | `complete`、`partial` 或 `failed` |
| `partial` | 只有全部必需只读证据完整、未采样且版本绑定明确时为 `false` |
| `section_statuses[]` | 每个分段的启用状态、必需性、记录数和读取上限 |
| `fetch_errors[]` | 结构化保存 section、method、path、code、HTTP 状态和可重试性 |
| `versions[]` | GitLink 返回的 patchset 版本原始只读数据 |
| `current_patchset` | 标准化的当前 patchset 摘要 |

分段状态：

```text
loaded    请求成功且有记录
empty     请求成功但没有记录
sampled   记录数达到读取上限，不能宣称全量
failed    请求失败
disabled  调用方显式关闭该可选分段
pending   尚未完成读取
```

以下分段构成完整 Review Context 的核心：

```text
pr
files
versions
reviews
threads
```

`partial=true` 时，飞书或企业微信适配层不得用本次空值覆盖上一次完整镜像；应保留
旧值、展示告警并安排重试。`fetch_errors[].message` 用于诊断，但不进入来源指纹，
避免上游自由文本变化制造无意义同步。

## 4. Patchset 与陈旧证据

真实 GitLink PR 详情响应不保证直接返回 head commit SHA。本轮改为 GET：

```text
/v1/{owner}/{repo}/pulls/{number}/versions
```

然后按更新时间、创建时间和数字 ID 选择最新版本，标准化：

```text
current_patchset.id
current_patchset.head_sha
current_patchset.base_sha
current_patchset.start_sha
current_patchset.files_count
current_patchset.commits_count
current_patchset.additions
current_patchset.deletions
```

正式 Review 和线程的 `commit_id` 与 `current_patchset.head_sha` 比较：

- 相同或安全 SHA 前缀匹配：`current`。
- 明确不同：`outdated`。
- 任一字段缺失：`unknown`。

只有过期 Review、没有当前决定时，WorkItem 进入
`waiting_for_re_review`，下一步为 `request_re_review_for_current_head`。

## 5. 真实 GitLink GET-only 合同验证

验证日期：2026-07-30。

验证条件：

- 未登录状态下读取公共仓库 `Gitlink/gitlink-cli`。
- 所有请求均为 GET。
- 未评论、审批、拒绝、解决线程、请求 Reviewer 或合并。
- 未调用飞书或企业微信 API。
- 未保存 token、cookie、Webhook 或账号映射。

### 5.1 当前 patchset

对公共 PR #431 的只读观察确认：

- PR 详情使用 `create_user`、`base`、`head`、`pull_request_staus`、
  `merged` 等字段。
- PR 详情未返回可靠 head SHA。
- versions 返回多个 patchset，并包含 `head_commit_sha`、
  `base_commit_sha`、`files_count`、`commits_count`、增删行和 Unix 时间。
- `workflow +review-context` 能从最新版本得到 head SHA、版本 ID 和 55 个文件。
- PR、files、versions、reviews、threads 均成功读取时，输出
  `collection_status=complete`。

### 5.2 Review 和线程

对公共 PR #14 与 #339 的只读观察确认：

- 正式 Review 包含 `id`、`status`、`content`、`created_at`、
  `commit_id` 和嵌套 `reviewer`。
- Review journal 包含 `id`、`note`、`state`、`need_respond`、`path`、
  `line_code`、`parent_id`、`review`、`commit_id` 和嵌套 `user`。
- 观察到的正式 Review 与部分 journal 的 `commit_id` 可以为 `null`。
- `commit_id=null` 时实现输出 `freshness=unknown`、
  `collection_status=partial`，不会把批准推导为当前版本批准。
- journal 正文能够进入标准化 `threads[].content` 并出现在 Markdown。

脱敏后的标准化观察样例保存在：

```text
shortcuts/workflow/testdata/review_context_p11_observed_fixture.json
shortcuts/workflow/testdata/review_context_p11_observed.golden.md
```

样例不包含真实用户 ID、真实正文或有效提交标识。

### 5.3 额外观察

以下是后续独立任务，不在 P1.1 扩大修改：

- PR 队列公共接口对 `state=open` 的过滤行为需要单独合同测试；观察结果中出现
  非开放条目。
- PR commits 读取曾返回 HTML 包装内容，需要在独立 PR 合同任务中确认端点。
- PR 详情未提供完整标题等队列元数据；P2 看板应关联 Review Queue，而不是
  在 Review Context 中猜测。

## 6. 状态路由

WorkItem 先尊重 GitLink 终态：

| GitLink / Review 条件 | `review_stage` | `recommended_next_step` |
|---|---|---|
| 已合并 | `merged` | `none` |
| 已关闭未合并 | `closed` | `none` |
| 当前 Reviewer 决定顺序冲突 | `human_reviewing` | `resolve_ambiguous_reviewer_decision` |
| 只有过期 Review | `waiting_for_re_review` | `request_re_review_for_current_head` |
| 当前拒绝 | `waiting_for_contributor` | `address_rejected_review` |
| 当前待响应线程 | `waiting_for_contributor` | `resolve_pending_review_threads` |
| 当前批准且无阻塞线程 | `ready_for_decision` | 维护者作最终决定 |

`ready_for_decision` 不等于 `merge_ready`。CI、分支保护、必需 Reviewer 和
GitLink 合并检查仍保持 `unknown`。

## 7. 测试矩阵

新增或扩展测试覆盖：

- 同一 Reviewer 拒绝后批准、批准后拒绝。
- 时间缺失时的数字 ID 顺序。
- 不同 Reviewer 的拒绝仍阻断。
- 决定顺序无法确定时保守为 unknown。
- ReviewerSummary 计数和最后有效决定。
- 线程正文、嵌套 Review ID 和孤儿回复。
- PR observed 字段形态与 merged 状态。
- 最新 patchset 选择与 head SHA。
- PR 主对象失败时命令失败。
- 单分段失败时结构化 partial/error。
- 所有分段失败时 `collection_status=failed`。
- merged、closed、waiting_for_re_review 路由。
- 本地完整 fixture 与真实观察脱敏 fixture 的 golden 输出。
- Mock Server 断言远端路径均为 GET。

## 8. P1.1 退出边界

本轮可以确认：

- 真实 PR 当前 patchset、文件、Review 和线程可只读获取。
- Review Context 能结构化表达完整、部分和失败采集。
- 当前与过期证据的离线合同和测试成立。
- 同一 Reviewer 的最后有效决定不会被历史拒绝永久阻断。
- P2 可以把该模型作为只读输入开始单独设计。

仍不能宣称：

- 已通过写入制造一个新 patchset 并现场观察旧 Review 由 current 变 stale。
- 已验证完整 Review/线程分页总量语义。
- 已完成飞书 Base、Doc、卡片、任务或长连接。
- 已完成飞书账号与 GitLink 账号绑定。
- 已从飞书执行任何 GitLink 操作。

由于本轮明确禁止远端写入，未主动推送新 patchset 以制造状态变化。该项保留为
专用测试仓库的人工只读回读验收：由维护者先产生版本变化，再运行本命令观察，
不由 P1.1 自动写入。

## 9. P2 输入规则

后续飞书适配层必须：

1. 以 `pr_key + head_sha + source_fingerprint` 作为事实镜像键。
2. 展示 `reviewer_summaries`，不自行重新解释原始 Review 顺序。
3. 展示线程正文，同时保留 GitLink 链接供维护者回到权威页面。
4. `partial=true` 时不覆盖上一份完整事实。
5. `merged` 和 `closed` 工作项从活动队列归档。
6. `unknown` 只生成核查任务，不生成批准或可合并结论。
7. 飞书只保存认领人、截止时间、协作备注和同步审计等协作字段。
8. GitLink 的 PR、Review、线程和合并状态始终是权威事实。

P2 应从通过本门禁后的提交创建新分支；不得把实验性 Channel SDK spike 宣称为
完整 P2。

## 10. 2026-07-30 验收结果

通过：

```bash
go test ./shortcuts/workflow -count=1
go test ./shortcuts/workflow ./shortcuts/pr ./shortcuts/feishu ./shortcuts -count=1
go test ./internal/i18n -count=1
go vet ./shortcuts/workflow
go build -o "$env:TEMP/gitlink-cli-round2-p11.exe" .
gitlink-cli workflow +review-context --help
gitlink-cli workflow +review-queue --help
gitlink-cli workflow +review-context --from \
  shortcuts/workflow/testdata/review_context_p1_fixture.json --format table
gitlink-cli workflow +review-context --from \
  shortcuts/workflow/testdata/review_context_p11_observed_fixture.json --format table
git diff --check
```

真实 GET-only smoke：

```text
Gitlink/gitlink-cli #431
state=open
stage=triaged
collection=complete
head=5b40a9088726134829e68ca656b85cfcaac1a8a2
files=55
reviews=0
threads=0
errors=0

Gitlink/gitlink-cli #14
state=closed
stage=closed
collection=partial
head=2aaa7e55fd6bea18644aaeb7d6b9017650c01439
reviews=1
threads=1
freshness=unknown
errors=0
```

第二个结果表明：HTTP 请求全部成功也不必然等于证据完整。平台返回
`commit_id=null` 时，`collection_status` 仍正确保持 `partial`。

全仓 `go test ./...` 仍失败，但 `shortcuts/workflow` 通过。失败来自本轮未修改
的既有区域，包括：

- `internal/client` 测试引用未定义 helper。
- `shortcuts/issue`、`milestone`、`snippet`、`user` 测试无法编译。
- 多个历史 shortcut 的注册表与旧测试期望不一致。
- alias、API、label、notification、release、repo 等旧合同测试漂移。
- 贡献者图表的国际化测试环境缺少预期文案。

全仓 Skill 元数据检查也仍被多个既有 Skill 的旧 frontmatter 字段阻断。本轮更新
的 `gitlink-pr-review-warroom` 保持既有合法结构，并只提升到 `1.1.0`。

这些历史基线不在 P1.1 扩展修复；本轮修改涉及的目标包、根 shortcuts、i18n、
静态检查与构建均已通过。

## 11. 分支与发布边界

2026-07-30 重新 fetch 后：

- GitLink `origin/master` 仍指向
  `d3bcbae82a20963c70a20407f8dc578e2127a754`。
- P1.1 直接建立在已上传 GitHub 的 P1 顶端
  `9c22c180d679eb8e49dace441f7a57b23f19840c`。
- P1/P0 是迁移后的完整提交链，与 GitLink `origin/master` 已形成不同提交
  身份；此时再次 rebase 到 `origin/master` 会重复或改写迁移历史。

因此本轮不再次改写 P0/P1 历史，只在 P1 顶端追加一个 P1.1 收口提交。发布只允许：

```text
GitHub: whzy3185/gitlink-feishu
Branch: fix/round2-review-collaboration-p1-gate
```

不推送 GitLink，不创建 PR。
