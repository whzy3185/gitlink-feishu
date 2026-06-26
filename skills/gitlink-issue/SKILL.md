---
name: gitlink-issue
version: 2.0.0
description: "Issue 管理：创建、查看、更新、关闭/批量关闭/批量更新/批量删除/批量重开/批量标签/批量指派/批量评论/批量导出/批量导入 Issue，添加评论。当用户需要操作 GitLink Issue 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-issue（Issue 操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `issue +list` | Issue 列表 | 否（公开项目） |
| `issue +create` | 创建 Issue | 是 |
| `issue +view` | Issue 详情 | 否（公开项目） |
| `issue +update` | 更新 Issue | 是 |
| `issue +close` | 关闭 Issue | 是 |
| `issue +batch-close` | 批量关闭 Issue，支持 `--dry-run` 预览 | 是（dry-run 不写入） |
| `issue +batch-reopen` | 批量重开已关闭 Issue，支持 `--dry-run` 预览 | 是（dry-run 不写入） |
| `issue +batch-update` | 按 API issue id 批量更新状态、优先级、里程碑、标签、负责人 | 是（dry-run 不写入） |
| `issue +batch-delete` | 按 API issue id 批量删除 Issue；真实删除必须 `--yes` | 是（dry-run 不写入） |
| `issue +batch-label` | 按 API issue id 批量添加/移除标签 | 是（dry-run 不写入） |
| `issue +batch-assign` | 按 API issue id 批量指派/取消指派负责人 | 是（dry-run 不写入） |
| `issue +batch-comment` | 批量添加评论到多个 Issue | 是（dry-run 不写入） |
| `issue +batch-export` | 导出 Issue 到 CSV 或 JSON 格式 | 否（公开项目） |
| `issue +batch-import` | 从 CSV 文件批量创建 Issue | 是（dry-run 不写入） |
| `issue +comment` | 添加评论 | 是 |
| `issue +assigners` | 查询 Issue 负责人列表 | 否（公开项目） |
| `issue +authors` | 查询 Issue 发布人列表 | 否（公开项目） |
| `issue +statuses` | 查询 Issue 状态列表 | 否（公开项目） |
| `issue +tags` | 查询 Issue 标签列表 | 否（公开项目） |
| `issue +priorities` | 查询 Issue 优先级列表 | 否（公开项目） |

## 使用示例

```bash
# 列出 Issue
gitlink-cli issue +list --owner Gitlink --repo forgeplus --state open

# 搜索并排序 Issue
gitlink-cli issue +list --owner Gitlink --repo forgeplus --state open --keyword 登录 --sort-by issues.updated_on --sort-direction desc

# 创建 Issue
gitlink-cli issue +create --owner myuser --repo myrepo --title "Bug: 登录失败" --body "复现步骤：..."

# 查看 Issue 详情（使用网页可见的 Issue 编号）
gitlink-cli issue +view --owner Gitlink --repo forgeplus --number 4

# 更新 Issue
gitlink-cli issue +update --number 4 --title "新标题" --body "更新描述"

# 关闭 Issue
gitlink-cli issue +close --number 4

# 预览批量关闭 Issue，不修改数据
gitlink-cli issue +batch-close --owner myuser --repo myrepo --numbers 123,124 --dry-run

# 从 CSV 文件批量关闭 Issue
gitlink-cli issue +batch-close --owner myuser --repo myrepo --from issues.csv

# 按 API issue id 预览批量更新元数据（注意不是网页 Issue 编号）
gitlink-cli issue +batch-update --owner myuser --repo myrepo --ids 101,102 --status-id 3 --priority-id 2 --dry-run

# 危险批量删除：必须先 dry-run，真实执行还要 --yes
gitlink-cli issue +batch-delete --owner myuser --repo myrepo --ids 101,102 --dry-run
gitlink-cli issue +batch-delete --owner myuser --repo myrepo --ids 101,102 --yes

# 批量重开已关闭的 Issue
gitlink-cli issue +batch-reopen --owner myuser --repo myrepo --numbers 123,124 --dry-run

# 批量添加标签（使用 API issue id 和标签 id）
gitlink-cli issue +batch-label --owner myuser --repo myrepo --ids 101,102 --add 1,2 --dry-run

# 批量移除标签
gitlink-cli issue +batch-label --owner myuser --repo myrepo --ids 101,102 --remove 3,4

# 批量指派负责人（使用 API issue id 和用户 id）
gitlink-cli issue +batch-assign --owner myuser --repo myrepo --ids 101,102 --add 5,6 --dry-run

# 批量取消指派
gitlink-cli issue +batch-assign --owner myuser --repo myrepo --ids 101,102 --remove 5,6

# 批量添加评论
gitlink-cli issue +batch-comment --owner myuser --repo myrepo --numbers 123,124 --message "已修复，请验证" --dry-run

# 导出 Issue 到 CSV
gitlink-cli issue +batch-export --owner myuser --repo myrepo --output issues.csv

# 导出 Issue 到 JSON（带过滤条件）
gitlink-cli issue +batch-export --owner myuser --repo myrepo --format json --status-id 5 --output closed_issues.json

# 从 CSV 文件批量创建 Issue
gitlink-cli issue +batch-import --owner myuser --repo myrepo --file issues.csv --dry-run

# 添加评论
gitlink-cli issue +comment --number 4 --body "已修复，请验证"

# 查询 Issue 负责人
gitlink-cli issue +assigners --owner Gitlink --repo forgeplus --keyword alice

# 查询 Issue 发布人
gitlink-cli issue +authors --owner Gitlink --repo forgeplus --keyword bob
```

## 批量维护安全约束

### ID 类型说明

- **网页 Issue 编号**（`project_issues_index`）：用于 `--numbers` 参数
  - `issue +batch-close --numbers`
  - `issue +batch-reopen --numbers`
  - `issue +batch-comment --numbers`

- **API Issue ID**（数据库内部 ID）：用于 `--ids` 参数
  - `issue +batch-update --ids`
  - `issue +batch-delete --ids`
  - `issue +batch-label --ids`
  - `issue +batch-assign --ids`

### 安全操作流程

- 执行 `batch-update` / `batch-delete` / `batch-label` / `batch-assign` 前，先用 `issue +list` 或 `issue +view` 确认 API issue id 来源。
- 写操作先执行 `--dry-run`，展示 `method`、`path`、`body` 给用户确认。
- `batch-delete` 是破坏性操作，真实执行必须显式传 `--yes`。
- `batch-label` 和 `batch-assign` 会保留现有标签/负责人，仅添加/移除指定的项。

### CSV 文件格式

`batch-import` 支持的 CSV 列：
- `subject`（必需）：Issue 标题
- `description`（可选）：Issue 描述
- `priority_id`（可选）：优先级 ID
- `status_id`（可选）：状态 ID（默认为 1）
- `assigner_ids`（可选）：负责人 ID 列表（逗号分隔）
- `issue_tag_ids`（可选）：标签 ID 列表（逗号分隔）

## Raw API 补充

```bash
# 获取 Issue 评论列表（使用 v1 API，按 issue number 查询）
gitlink-cli api GET /v1/:owner/:repo/issues/:number/journals

# 批量更新 Issue（仍使用旧版 API，需传数据库 ID）
gitlink-cli api POST /:owner/:repo/issues/series_update --body '{"ids":[1,2,3],"status_id":"closed"}'

# 批量添加评论（使用 v1 API）
gitlink-cli api POST /v1/:owner/:repo/issues/:number/journals --body '{"notes":"评论内容"}'

# 导出 Issue 列表（使用 v1 API）
gitlink-cli api GET /v1/:owner/:repo/issues.json
```

## GitLink Issue 字段映射

| gitlink-cli 参数 | GitLink API 字段 | 说明 |
|------------------|-----------------|------|
| `--number` / `-n` | `project_issues_index` | Issue 编号（网页 URL 中的序号） |
| `--id` / `-i` | `project_issues_index` | `--number` 的兼容别名，不是数据库内部 ID |
| `--title` | `subject` | Issue 标题 |
| `--body` | `description` | Issue 描述 |
| `--assignee` | `assigned_to_id` | 指派人 ID |
| `--milestone` | `fixed_version_id` | 里程碑 ID |
| `--state` | `status_id` | 状态（open=1，closed=5，也可直接传数字 ID） |
| `--priority-id` | `priority_id` | 优先级 ID |
| `--tag-ids` / `--label` | `issue_tag_ids` | Issue 标签 ID 数组 |
| `--assigner-ids` | `assigner_ids` | 负责人 ID 数组 |
| `--branch` | `branch_name` | 关联分支 |
| `--start-date` | `start_date` | 开始日期 |
| `--due-date` | `due_date` | 截止日期 |

## API 注意事项

- **Issue 编号（`--number`）是网页 URL 中看到的序号**（如 `issues/4` 中的 `4`），不是数据库内部 ID
- `--id` / `-i` 仅作为 `--number` / `-n` 的兼容别名，传入的仍然是网页 URL 中的 Issue 编号
- **批量关闭使用 `--numbers`，同样传网页 URL 中的 Issue 编号**，不是数据库内部 ID
- Issue 操作使用 v1 API（`/api/v1/`），支持按 Issue 编号查询和操作
- **创建 Issue 时 CLI 会自动设置 `status_id: 1`（新增）和 `priority_id: 2`（正常）**
- **更新/关闭 Issue 时必须保留当前 `subject` 和 `description`**，即使只修改状态（CLI 会先读取当前 Issue 并自动带回）
- v1 API 写操作必须使用 `access_token`（非 `token`）认证，CLI 已自动处理

## Issue 状态映射（status_id）

| status_id | 名称 | 说明 |
|-----------|------|------|
| 1 | 新增 | 新建 Issue 的默认状态 |
| 2 | 正在解决 | 处理中 |
| 3 | 已解决 | 已修复 |
| 5 | 关闭 | 关闭（`+close` 命令使用此值） |
