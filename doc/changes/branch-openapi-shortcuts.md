# Branch OpenAPI Shortcuts

补齐 GitLink 分支 OpenAPI 的生命周期操作，并增强现有 branch shortcut 的安全性和参数能力。

## 新增 / 增强命令

- `branch +list`：新增 `--keyword` 和 `--state all|deleted`，对齐 OpenAPI 查询参数。
- `branch +all`：调用无分页分支列表接口。
- `branch +create`：新增 `--dry-run`，预览创建分支请求。
- `branch +delete`：切换到 OpenAPI 文档中的 `DELETE /api/v1/{owner}/{repo}/branches/{branch}.json`，并新增 `--dry-run`。
- `branch +set-default`：设置仓库默认分支，支持 `--dry-run`。
- `branch +restore`：恢复已删除分支，支持 `--dry-run`。

## OpenAPI 对齐

- `GET /api/v1/{owner}/{repo}/branches.json`
- `POST /api/v1/{owner}/{repo}/branches.json`
- `GET /api/v1/{owner}/{repo}/branches/all.json`
- `DELETE /api/v1/{owner}/{repo}/branches/{branch}.json`
- `PATCH /api/v1/{owner}/{repo}/branches/update_default_branch.json`
- `POST /api/v1/{owner}/{repo}/branches/restore.json`

## 安全设计

- `branch +create`、`branch +delete`、`branch +set-default`、`branch +restore` 都支持 `--dry-run`。
- `branch +delete` 会对包含 `/` 的分支名进行路径转义，避免把 `feature/foo` 误解析为多级路径。
- `branch +restore` 校验 `--branch-id` 必须为正整数。
- `branch +list --state` 仅允许 `all` 或 `deleted`，避免无效状态参数。

## 测试

新增单元测试覆盖：

- list 查询参数。
- all endpoint。
- create payload 与 dry-run。
- delete v1 endpoint 与路径转义。
- set-default query 参数。
- restore payload。
- 无效 state / branch-id 不触发 API。
