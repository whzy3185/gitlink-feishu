# Dataset Shortcuts

## Summary

Adds a new `dataset` shortcut group for managing and querying GitLink research
datasets, which previously had no shortcut coverage. Datasets carry
research-oriented metadata (title, description, `paper_content`, license, owning
project) that is valuable for research/scientometric scenarios.

## Commands

| Command | Purpose | Endpoint |
|---------|---------|----------|
| `gitlink-cli dataset +view` | View a repository's dataset and attachments | `GET /v1/{owner}/{repo}/dataset` |
| `gitlink-cli dataset +list --ids <ids>` | List datasets for one or more projects | `GET /v1/project_datasets` |
| `gitlink-cli dataset +create` | Create a repository's dataset | `POST /v1/{owner}/{repo}/dataset` |
| `gitlink-cli dataset +update` | Update a repository's dataset | `PUT /v1/{owner}/{repo}/dataset` |
| `gitlink-cli dataset +delete-attachment --uuid <uuid>` | Delete a dataset attachment | `DELETE /attachments/{uuid}` |

## Behaviour

- `+view` paginates attachments via `--page`/`--limit`.
- `+list --ids 1,2,3` queries datasets by comma-separated numeric project IDs;
  IDs are validated client-side before the request.
- `+create`/`+update` send `title`, `description`, optional `license-id`
  (validated as a positive integer) and `paper-content`. Both support
  `--dry-run` to preview the request body without writing.
- `+delete-attachment` is destructive: it requires `--dry-run` preview or an
  explicit `--yes` confirmation before issuing the DELETE.

## Production status (verified)

Verified against production `gitlink.org.cn`:

- `GET /v1/project_datasets` (`+list`) — **available and verified** (e.g.
  `--ids 5988` returns the forgeplus dataset).
- The per-repository routes `/v1/{owner}/{repo}/dataset`
  (`+view`/`+create`/`+update`) currently return `404` on production www
  (confirmed even for a repository's own owner; not reachable on the gateway
  host either). They follow the documented contract and are expected to work
  once the platform deploys these routes. `+delete-attachment` targets the
  generic attachments endpoint.

The commands and request shapes match the published OpenAPI spec, so they are
ready the moment the routes go live; unit tests exercise every command against a
mock server.

## Tests

Unit tests cover the view path with pagination, `--ids` normalization and
validation, create/update request bodies and `license-id` validation, dry-run
previews, and the destructive-delete confirmation guard (`--yes`).

## 中文说明

### 变更内容

- 新增 `dataset` 命令组：`+view`、`+list`、`+create`、`+update`、`+delete-attachment`。
- `+view` 支持 `--page`/`--limit` 对附件分页；`+list --ids` 按项目 ID 查询。
- `+create`/`+update` 发送 `title`/`description`/可选 `license-id`/`paper-content`，均支持 `--dry-run` 预览。
- `+delete-attachment` 为破坏性操作，需 `--dry-run` 预览或显式 `--yes` 确认。

### 生产状态（已验证）

- `GET /v1/project_datasets`（`+list`）在生产**可用并已验证**（如 `--ids 5988` 返回 forgeplus 数据集）。
- `/v1/{owner}/{repo}/dataset` 的 `+view`/`+create`/`+update` 当前在生产 www 返回 `404`（即使对仓库 owner 也如此，gateway 也未托管）。实现严格遵循已发布的 OpenAPI 契约，待平台部署后即可生效；单测以 mock 覆盖全部命令。

### 相对文档契约的增强

双语 i18n 帮助文案、写操作 `--dry-run` 预览、破坏性删除 `--yes` 二次确认、`license-id` 正整数校验。
