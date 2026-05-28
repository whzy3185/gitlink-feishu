# Issue and PR Journal Shortcuts

补齐 Issue 评论/操作记录与 Pull Request Review 评论相关 OpenAPI 封装。

## Issue comments / journals

新增或增强：

- `issue +comment`：添加评论，新增 `--parent-id`、`--reply-id`、`--attachment-ids`、`--receivers`、`--dry-run`。
- `issue +comments`：查看 Issue 评论和操作记录，支持 `category`、关键字、排序、分页。
- `issue +comment-update`：更新 Issue 评论，支持附件、@接收人和 `--dry-run`。
- `issue +comment-delete`：删除 Issue 评论，支持 `--dry-run`。
- `issue +comment-children`：查看指定评论的子评论。

覆盖 OpenAPI：

- `GET /api/v1/{owner}/{repo}/issues/{index}/journals.json`
- `POST /api/v1/{owner}/{repo}/issues/{index}/journals.json`
- `PATCH /api/v1/{owner}/{repo}/issues/{index}/journals/{id}.json`
- `DELETE /api/v1/{owner}/{repo}/issues/{index}/journals/{id}.json`
- `GET /api/v1/{owner}/{repo}/issues/{index}/journals/{id}/children_journals.json`

## PR review comments

新增：

- `pr +review-comments`：查看 PR Review 行评论，支持 review/state/path/parent/keyword 等过滤。
- `pr +review-comment`：创建 PR Review 行评论，支持 `comment` / `problem` 类型和 `--dry-run`。
- `pr +review-comment-update`：更新 Review 评论内容、commit 或状态，支持 `--dry-run`。
- `pr +review-comment-delete`：删除 Review 评论，支持 `--dry-run`。

覆盖 OpenAPI：

- `GET /api/v1/{owner}/{repo}/pulls/{index}/journals.json`
- `POST /api/v1/{owner}/{repo}/pulls/{index}/journals.json`
- `PUT /api/v1/{owner}/{repo}/pulls/{index}/journals/{id}.json`
- `DELETE /api/v1/{owner}/{repo}/pulls/{index}/journals/{id}.json`

## 安全设计

- 所有写入/删除评论的命令支持 `--dry-run`，先输出 method/path/body，不直接改远端数据。
- PR Review 行评论创建支持 `--diff-json` / `--diff-file`，复杂 diff payload 由用户或 Agent 明确传入，避免 CLI 猜测 line diff。
- 参数校验覆盖 Issue journal category、PR review comment type/state、布尔查询参数、正整数 ID 和 JSON diff。

## 测试

新增单元测试覆盖：

- Issue comments list/create/update/delete/children 的 method、path、query、payload。
- PR review comments list/create/update/delete 的 method、path、query、payload。
- dry-run 不触发 API。
- 参数校验失败不触发 API。
