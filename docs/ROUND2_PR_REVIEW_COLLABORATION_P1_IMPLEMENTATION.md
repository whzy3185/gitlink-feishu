# GitLink CLI 复赛 PR Review 协同 P1 实施记录

实施日期：2026-07-29

实现基线：GitHub `feat/round2-review-collaboration-p0` @ `5a3d4b0726a6c6288ec21d9128646a30b779c6b0`

实现分支：`feat/round2-review-collaboration-p1`

## 1. 本轮目标

P1 将 P0 的只读 Review 输入收敛为可供维护者、协作适配层和 Agent Host
共同消费的稳定契约：

```text
完整拉取开放 PR 队列
-> 获取单个 PR、正式 Review 与 Review 线程
-> 将证据绑定到当前 head SHA
-> 区分当前、过期和未知证据
-> 生成稳定 Review WorkItem 和来源指纹
-> 输出本地 JSON、Markdown 或表格
```

本轮仍不评论、不审批、不拒绝、不解决线程、不请求 Reviewer，也不合并 PR。

## 2. 已实现

### 2.1 稳定 Review Context

`workflow +review-context` 输出 `review.context/v1`，并保留原有原始 API
字段以兼容已有调用方。新增的稳定字段包括：

- 当前 PR head SHA 和版本 ID。
- 标准化的 `review_records`。
- 标准化的 Review `threads`。
- `review_summary` 聚合状态。
- `review.work-item/v1` 协作工作项。
- 可复现的 `source_fingerprint`。
- 无法从只读证据确认的 `unknowns`。

`ReviewRun` 和 `ReviewTask` 已建立传输层结构，但 P1 不持久化、不调度，也不
执行这些对象。

### 2.2 Review 版本新鲜度

每条正式 Review 和 Review 线程都与当前 head SHA 比较：

| 条件 | 新鲜度 |
|---|---|
| commit 与 head 完整相同，或使用不少于 7 位的安全前缀匹配 | `current` |
| commit 与 head 明确不同 | `outdated` |
| commit 或 head 缺失，或短前缀不足以确认 | `unknown` |

过期 Review 保留为审计证据，但不参与当前版本的批准或阻断判断。未知证据保守
处理，不能推导为可合并。

### 2.3 Review 与线程聚合

聚合只使用当前版本证据：

1. 当前版本存在 `rejected` Review：`blocked`。
2. 当前版本存在待响应线程或打开的问题线程：`changes_pending`。
3. 当前版本存在 `approved` Review：`approved`。
4. 当前版本只有普通评论：`commented`。
5. 没有可用当前证据：`pending`。

`approved` 只表示存在当前版本批准证据。由于分支保护、必需 Reviewer、CI 和
GitLink 合并检查并未形成完整契约，P1 不输出 `merge_ready`。

### 2.4 完整队列分页

`workflow +review-queue --all` 连续读取 PR 列表页面：

- 单页上限为 100。
- `--page` 指定起始页。
- `--max-items` 默认 1000，防止异常分页无限读取。
- 按 PR number 去重；number 缺失时使用标题和作者作为保守后备键。
- 全程只使用 GET 请求。

### 2.5 维护者协作 Skill

新增 `skills/gitlink-pr-review-warroom/SKILL.md`，用于指导 Agent Host：

- 建立完整 Review 队列。
- 获取重点 PR 的稳定上下文。
- 正确解释 Review 新鲜度和线程状态。
- 形成维护者报告。
- 遵守 P1 只读边界。

该 Skill 不包含独立 Agent 运行时，也不调用模型 API。它把 GitLink CLI 的
确定性只读能力暴露给现有 Agent Host 使用。

## 3. 命令示例

### 完整 Review 队列

```bash
gitlink-cli workflow +review-queue \
  --owner Gitlink \
  --repo gitlink-cli \
  --state open \
  --all \
  --limit 50 \
  --max-items 1000 \
  --format json
```

### 单个 PR Review Context

```bash
gitlink-cli workflow +review-context \
  --owner Gitlink \
  --repo gitlink-cli \
  --number 42 \
  --include-reviews \
  --include-threads \
  --thread-limit 100 \
  --format json
```

### 无平台凭据的离线演示

```bash
gitlink-cli workflow +review-context \
  --from shortcuts/workflow/testdata/review_context_p1_fixture.json \
  --format markdown
```

离线读取会重新计算新鲜度、聚合状态和来源指纹；已经标准化且不含原始 Review
字段的 `review.context/v1` 也能再次读取。

## 4. 测试证据

新增测试覆盖：

- Review SHA 完整匹配、安全前缀匹配、不匹配、缺失与短前缀。
- 当前批准与过期拒绝并存时，只使用当前版本证据。
- 当前拒绝阻断。
- 打开、已解决和待响应线程聚合。
- 回复缺少父记录时的保守标记。
- 输入顺序变化时来源指纹保持稳定。
- JSON fixture 到 Markdown golden 的固定输出。
- 使用 `--from` 从 fixture 或已采集上下文完成无凭据离线演示。
- Review Queue 多页读取、重复项去除和安全上限。
- Mock Server 断言新增远端调用仍为 GET。

验收命令：

```bash
go test ./shortcuts/workflow -count=1
go test ./shortcuts/workflow ./shortcuts/pr ./shortcuts/feishu ./shortcuts
go test ./internal/i18n
go vet ./shortcuts/workflow
go build -o "$env:TEMP/gitlink-cli-round2-p1.exe" .
go run . workflow +review-context --help
go run . workflow +review-queue --help
git diff --check
```

### 2026-07-29 验收结果

- `go test ./shortcuts/workflow -count=1`：通过。
- `go test ./shortcuts/workflow ./shortcuts/pr ./shortcuts/feishu ./shortcuts`：
  通过。
- `go test ./internal/i18n`：通过。
- `go vet ./shortcuts/workflow`：通过。
- `go build -o "$env:TEMP/gitlink-cli-round2-p1.exe" .`：通过，未在仓库中
  生成或修改二进制。
- `go run . workflow +review-context --help`：通过，线程读取参数可见。
- `go run . workflow +review-queue --help`：通过，全量分页和安全上限参数可见。
- `go run . workflow +review-context --from
  shortcuts/workflow/testdata/review_context_p1_fixture.json --format table`：通过，
  离线输出识别为 `changes_pending/current`。
- `git diff --check`：通过；仅有 Windows 工作区的 LF/CRLF 提示。
- `go run ./internal/skillmeta/cmd/check`：新 Skill 没有检查问题；全仓命令仍被
  多个本轮未修改 Skill 的既有 frontmatter 漂移阻断。
- `go test ./...`：未通过，失败范围与 P0 已记录的主线基线一致，主要包括
  未定义的旧测试 helper、尚未恢复注册的 shortcut、过期 API 测试契约和既有
  Skill 元数据。
- `go vet ./...`：未通过，仍被本轮未修改的 `internal/client`、
  `shortcuts/issue`、`shortcuts/milestone`、`shortcuts/snippet` 和
  `shortcuts/user` 测试代码阻断。

因此 P1 使用涉及包的定向测试、国际化测试、Workflow 静态检查和根程序构建作为
提交门禁，不在本分支扩大修复全仓既有问题。

## 5. 安全和发布边界

本轮没有：

- 调用 GitLink 写 API。
- 调用飞书或企业微信写 API。
- 创建、审批、拒绝或合并 PR。
- 自动解决 Review 线程。
- 保存 token、cookie 或账号映射。
- 上传 GitLink。

本轮只允许将实现分支推送到 GitHub
`whzy3185/gitlink-feishu`，且不自动创建 GitHub PR。

## 6. 已知限制

- Review 和线程字段来自当前公开客户端模型与本地 Mock 证据，仍需使用真实
  GitLink 只读响应完成字段契约复核。
- Review 默认最多取 100 条；线程默认最多取 100 条并可由 `--thread-limit`
  调整。任一已加载数据达到对应上限时 `source_scope.sampled=true`。
- 必需 Reviewer、团队审批规则、分支保护和完整 CI 合并闸门尚未统一，因此
  mergeability 保持 `unknown`。
- `ReviewRun` 与 `ReviewTask` 目前只有结构定义；共享看板、租约、审计事件和
  飞书写回属于后续阶段。
- 飞书与企业微信适配层只能消费 P1 输出，不能把协作平台状态覆盖为 GitLink
  权威状态。

## 7. 下一轮建议

在继续写回能力前，先完成一个真实仓库的只读契约验证：

1. 采集脱敏后的 PR、Review、Review 线程和版本响应样例。
2. 确认正式 Review 与线程的 commit 绑定字段。
3. 确认分页总量和终止语义。
4. 补充 CI、分支保护和 Reviewer 规则的只读探测。
5. 用 `source_fingerprint + head_sha` 建立写回前的陈旧数据保护。
6. 之后再单独设计飞书多维表格、群卡片和云文档适配层的 dry-run 与显式确认。
