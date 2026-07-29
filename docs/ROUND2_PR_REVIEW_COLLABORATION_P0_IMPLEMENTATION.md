# GitLink CLI 复赛 PR Review 协同 P0 实施记录

实施日期：2026-07-29

设计基线：`docs/round2-pr-review-collaboration-v2` @ `c31ad677e1865998edba5a54e6b434fd6b586c78`

实现分支：`feat/round2-review-collaboration-p0`

后续进展：[P1 实施记录](./ROUND2_PR_REVIEW_COLLABORATION_P1_IMPLEMENTATION.md)

## 1. 本轮目标

本轮只恢复主线可调用性并建立只读协作输入，不进行真实 GitLink、飞书或企业微信写入：

```text
恢复命令注册
-> 修复批量合并后的契约漂移
-> 暴露只读 Review 评论查询
-> 复用 Review Context 和 Review Queue
-> 建立自动测试证据
```

## 2. 已实现

### 根命令

- 恢复 `feishu` 命令组注册。
- 将根命令测试从脆弱的固定数量改为必需命令组检查，同时保留重复、空描述和空子命令检查。

### PR

- 恢复 `pr +commits`。
- 恢复 `pr +branches`。
- 恢复 `pr +check-merge`。
- 恢复 `pr +checks`。
- 恢复 `pr +checkout`。
- 恢复 `pr +diff --file` 查询参数。
- 暴露只读 `pr +review-comments`。
- 保持 `pr +review-comment`、`pr +review-comment-update` 和 `pr +review-comment-delete` 未注册。
- 修复 `pr +review` 测试使用的参数契约。
- 移除 `pr +review` 成功后的隐式 journal 重复写入；现在一次调用只创建一条正式 Review。

### Workflow

- 恢复以下已有只读 Workflow 的注册：
  - `+review-queue`
  - `+release-notes`
  - `+dependency-audit`
  - `+issue-dedupe`
  - `+release-readiness`
  - `+stale`
- 保留并验证 `+review-context`。
- 修复 GitLink 响应中的 `issue_tags`、`issue_journals`、`status_name`、`comment_journals_count` 和分钟级时间格式归一化。

## 3. 安全边界

当前允许：

```text
PR、Review、Review 评论读取
本地规则分析
Review 队列排序
飞书命令本地 preview
用户显式调用原有 CLI 写命令
```

当前不允许从协作入口触发：

```text
自动 Review
行级评论创建、回复、解决或删除
Reviewer 请求或移除
approved/rejected 自动写入
自动合并
任意 Raw API
```

本轮没有执行真实远端写请求，也没有验证 Reviewer 管理端点。

## 4. 新增合同测试

`pr +review-comments` 新增测试覆盖：

- 命令注册边界。
- 写命令保持不可达。
- GET method 和 `/v1/{owner}/{repo}/pulls/{index}/journals` path。
- keyword、review ID、need respond、state、parent、中文 path、full thread 和排序 query。
- 非法布尔值在发起请求前失败。

现有测试同时验证：

- PR commits、branches、check-merge、checks。
- `pr +review` 一次调用只发送一次正式 Review 请求，并携带可选 commit SHA。
- `pr +review --dry-run` 不发送任何 HTTP 请求。
- Workflow 命令注册完整性。
- GitLink 多种字段和 envelope 归一化。
- 飞书命令包和根命令组注册。

## 5. P0 验收命令

```bash
go test ./shortcuts/pr ./shortcuts/workflow ./shortcuts/feishu ./shortcuts
go test ./...
go vet ./...
go build .
go run . feishu --help
go run . workflow --help
go run . pr --help
git diff --check
```

### 2026-07-29 验收结果

- `go test ./shortcuts/pr ./shortcuts/workflow ./shortcuts/feishu ./shortcuts`：通过。
- `go test ./internal/i18n`：通过。
- `go build .`：通过；构建产生的本地二进制变化未纳入提交。
- `go run . feishu --help`：通过，15 个飞书协作命令可见。
- `go run . workflow --help`：通过，11 个只读工作流命令可见。
- `go run . pr --help` 与 `go run . pr +review-comments --help`：通过。
- `git diff --check`：通过；仅报告 Windows 工作区的 LF/CRLF 提示。
- `go test ./...`：未通过。失败集中在本轮未修改的主线基线包，包括
  `internal/client`、`internal/capability`、`internal/output`、`internal/skillmeta`，
  以及若干尚未恢复注册或测试契约漂移的 shortcuts 包。
- `go vet ./...`：未通过。当前被主线测试代码中的未定义符号和重复测试函数阻断，
  包括 `internal/client`、`shortcuts/issue`、`shortcuts/milestone`、
  `shortcuts/snippet` 和 `shortcuts/user`。

因此本轮以 P0 涉及包的定向测试、国际化测试和根程序构建作为提交门禁；全仓基线
修复应单独规划，避免将 PR Review 协同实现扩张为无边界的主线清理。

## 6. 下一步

进入 P1 前继续保持只读：

1. 为 `review-context` 增加当前 head SHA、Review commit 和线程数据。
2. 定义稳定的 `PullRequestReviewContext` JSON schema。
3. 实现 Review 新鲜度：`current/outdated/unknown`。
4. 实现多人 Review 和待响应线程的只读聚合状态。
5. 增加固定 fixture、golden test 和 Markdown 报告。
6. 在专用测试仓库完成只读 API smoke 后，再讨论任何写入功能。
