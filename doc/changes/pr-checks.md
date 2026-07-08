# pr +checks 合并请求 CI 构建状态命令

## 背景

`gitlink-cli pr` 已覆盖合并请求的列表、详情、评审、评论等操作，`gitlink-cli ci +builds` 可列出仓库的 CI 构建，但两者相互独立。用户或 AI Agent 想确认「某个合并请求的最新提交是否通过了 CI」，此前需要手动读取 PR 的源分支/源提交，再逐条比对构建列表。

本次变更新增 `pr +checks`，对齐 `gh pr checks` 的语义：解析合并请求 head，自动关联并汇总对应的 CI 构建状态。

## 变更内容

- 新增 `gitlink-cli pr +checks --id N` Shortcut。
- 先 `GET /{owner}/{repo}/pulls/{id}` 读取合并请求详情，取源分支 `head` 与源提交 `head_commit_sha`（字段依据 API 文档「获取一个合并请求」章节）。
- 再 `GET /{owner}/{repo}/builds` 拉取 CI 构建列表，在客户端按 head 提交/分支筛选。
- 输出规范化的状态摘要：`matched_by`、`total_builds` 以及每条构建的 `id / stage / status / conclusion / branch / sha`。
- 复用现有仓库上下文解析、API 调用与统一输出封装；新增中英文 i18n 文案。

## 匹配策略

| 优先级 | 条件 | `matched_by` |
|--------|------|--------------|
| 1 | 构建提交 SHA 与 head 提交一致（支持缩写前缀比对） | `sha` |
| 2 | 无 SHA 命中，但构建分支等于 head 分支 | `branch` |
| 3 | 构建未暴露任何分支/提交字段，无法建立关联 | `unlinkable` |
| 4 | 构建暴露了分支/提交字段但均不匹配 | `none` |

## 命令示例

```bash
gitlink-cli pr +checks --owner Gitlink --repo forgeplus --id 42
gitlink-cli pr +checks --id 42 --format json
```

## 已知限制

GitLink 的 `/{owner}/{repo}/builds` 端点未纳入官方 OpenAPI 参考文档，构建对象中承载分支与提交的字段名无法从文档确证。为避免臆造字段：

- 分支字段按 `branch / head_branch / source_branch / ref` 依次探测（`ref` 会去除 `refs/heads/` 前缀）。
- 提交字段按 `head_commit_sha / commit_sha / commit_id / sha / after / revision` 依次探测。
- 若某次构建两类字段均缺失，则判定为无法关联（`matched_by = unlinkable`），此时**降级返回全部最近构建并附带说明**，由使用者依据 `head_sha` 手动核对，而非丢弃结果或猜测字段。

后续若 `/builds` 响应结构被官方文档化，可据实收敛探测键集合。

## 测试覆盖

- 表驱动 httptest：先 mock `GET pulls/{id}`、再 mock `GET builds`，断言四种 `matched_by` 分支（sha 优先于 branch、缩写 SHA 命中、branch 回退、unlinkable 全量降级、none 无命中）与选中的构建 id。
- head 字段缺失、builds 请求 HTTP 失败的错误路径。
- 纯函数单测：`extractPullRequestHead`、`buildsFromEnvelope`（含客户端把顶层数组作为字符串返回的情形）、`commitMatches`、仅有分支的 PR。

验证命令：

```bash
go build ./...
go test ./shortcuts/pr/ ./shortcuts/ci/
```

## 交付要求核对

- 功能代码：`shortcuts/pr/pr.go`、`shortcuts/pr/checks.go`
- 单元测试：`shortcuts/pr/checks_test.go`
- i18n 文案：`internal/i18n/locales/en-US.json`、`internal/i18n/locales/zh-CN.json`
- 变更说明文档：`doc/changes/pr-checks.md`

## 兼容性

该变更只新增 Shortcut、辅助函数、单元测试、i18n 文案与文档，不修改任何已有命令的参数或输出结构。
