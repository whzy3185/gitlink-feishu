---
name: gitlink-wiki
version: 2.0.0
description: "Wiki 页面管理：查看目录、查看、创建、更新和删除 GitLink Wiki 页面。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
---

# gitlink-wiki

**重要**: 开始操作前请先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中涵盖认证、权限、全局参数和 GitLink API 行为说明。
**重要**: 执行写入或破坏性操作（如 `+create`、`+update` 或 `+delete`）前，请先确认用户意图。
**重要**: 操作 GitLink 资源请使用 `gitlink-cli`，不要使用 `gh` 等 GitHub 专用工具。

## 快捷命令

| 快捷命令 | 说明 | 操作类型 |
|----------|------|----------|
| `wiki +list` | 列出 Wiki 页面（目录结构） | 只读 |
| `wiki +view` | 按页面名称查看 Wiki 页面详情 | 只读 |
| `wiki +create` | 创建新的 Wiki 页面 | 写入 |
| `wiki +update` | 更新 Wiki 页面标题和/或内容 | 写入 |
| `wiki +delete` | 删除 Wiki 页面 | 破坏性 |

## 使用示例

```bash
# 列出 Wiki 目录
gitlink-cli wiki +list --owner Gitlink --repo forgeplus --project-id 12345

# 查看 Wiki 页面
gitlink-cli wiki +view --owner Gitlink --repo forgeplus --project-id 12345 -n home

# 创建 Wiki 页面
gitlink-cli wiki +create --owner Gitlink --repo forgeplus --project-id 12345 \
  -n getting-started -t "快速开始" -c "# 快速开始指南\n\n这是入门文档。"

# 创建时附带提交信息
gitlink-cli wiki +create --owner Gitlink --repo forgeplus --project-id 12345 \
  -n api-guide -t "API 指南" -c "# API 指南" -m "Add API guide"

# 仅更新页面标题
gitlink-cli wiki +update --owner Gitlink --repo forgeplus --project-id 12345 -n home -t "新标题"

# 仅更新页面内容
gitlink-cli wiki +update --owner Gitlink --repo forgeplus --project-id 12345 -n home -c "# 更新后的内容"

# 同时更新标题和内容
gitlink-cli wiki +update --owner Gitlink --repo forgeplus --project-id 12345 \
  -n home -t "新标题" -c "新内容"

# 删除 Wiki 页面
gitlink-cli wiki +delete --owner Gitlink --repo forgeplus --project-id 12345 -n old-page
```

## 参数说明

| 命令 | 关键参数 |
|------|----------|
| `+list` | `--project-id` |
| `+view` | `--project-id`、`--page-name` (`-n`) |
| `+create` | `--project-id`、`--page-name` (`-n`)、`--title` (`-t`)、`--content` (`-c`)、`--message` (`-m`) |
| `+update` | `--project-id`、`--page-name` (`-n`)、`--title` (`-t`)，可选 `--content` (`-c`)、`--message` (`-m`) |
| `+delete` | `--project-id`、`--page-name` (`-n`) |

## API 说明

- 所有 Wiki 端点使用 `/api/wiki/{action}` 扁平路径结构（非 REST 嵌套路径）。
- 目录列表: `GET /api/wiki/wikiPages`（查询参数：owner, repo, projectId）
- 查看详情: `GET /api/wiki/getWiki`（查询参数：owner, repo, projectId, pageName）
- 创建页面: `POST /api/wiki/createWiki`（JSON body，content 需 base64 编码）
- 更新页面: `PUT /api/wiki/updateWiki`（JSON body，content 需 base64 编码）
- 删除页面: `DELETE /api/wiki/deleteWiki`（JSON body：owner, repo, projectId, pageName）
- Wiki 页面通过 `pageName`（slug）标识，而非数字 ID。
- 所有操作都需要 `--project-id`（GitLink 项目数字 ID）。
- 创建和更新时，内容自动进行 base64 编码后以 `content_base64` 字段发送。
- `+update` 要求必须提供 `--title` 和 `--page-name`；`--content` 为可选。
