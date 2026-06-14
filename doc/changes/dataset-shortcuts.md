# Dataset Shortcuts

## Summary

Adds a new `dataset` shortcut group for querying GitLink research datasets,
which previously had no shortcut coverage. Datasets carry research-oriented
metadata (title, description, `paper_content`, license, owning project) that is
valuable for research/scientometric scenarios.

## Commands

| Command | Purpose | Endpoint |
|---------|---------|----------|
| `gitlink-cli dataset +list --ids <ids>` | List datasets for one or more projects | `GET /v1/project_datasets` |
| `gitlink-cli dataset +view` | View a repository's dataset | `GET /v1/project_datasets` (project ID resolved from `--owner/--repo`) |

## Behaviour

- `dataset +list --ids 1,2,3` queries datasets by comma-separated numeric
  project IDs. IDs are validated client-side before the request.
- `dataset +view --owner X --repo Y` resolves the repository's numeric project
  ID from the repository info endpoint, then queries the dataset for that
  project. Pass `--project-id` to skip resolution.

## Scope note (verified against production)

The dataset CRUD routes documented under `/api/v1/{owner}/{repo}/dataset`
(`POST`/`PUT`/`GET`) are **not deployed on the production `gitlink.org.cn`
host** — they return `404 您访问的页面不存在` even for a repository's owner. Only
the platform-wide query endpoint `GET /api/v1/project_datasets` is available in
production, so this group wraps that endpoint for both listing and per-repo
viewing. Create/update/attachment-delete can be added once the corresponding
routes are live on production.

## Tests

Unit tests cover `--ids` normalization (whitespace, ordering) and validation,
the missing/invalid `--ids` guards, project ID auto-resolution from repo info
for `+view`, explicit/invalid `--project-id`, and HTTP error handling.

## 中文说明

### 变更内容

- 新增 `dataset` 命令组：
  - `dataset +list --ids 1,2,3` 按数字项目 ID 查询数据集
  - `dataset +view --owner X --repo Y` 自动解析仓库 project_id 后查询该仓库数据集（可用 `--project-id` 跳过解析）
- 数据集含 `paper_content`、license、所属项目等科研相关元数据，服务科研数据发现场景。

### 范围说明（已对生产环境验证）

文档中 `/api/v1/{owner}/{repo}/dataset` 的增删改查路由在生产 `gitlink.org.cn`
**未部署**（即使对仓库 owner 也返回 `404 页面不存在`）。生产可用的只有平台级查询
端点 `GET /api/v1/project_datasets`，故本命令组基于该端点实现列表与按仓库查看。待
对应路由在生产上线后，可补充创建/更新/附件删除。
