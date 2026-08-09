# Wiki Shortcuts

## Summary

Adds a new `wiki` shortcut group so maintainers and agents can manage a
repository's wiki without falling back to Raw API calls. Wiki was listed in the
competition guide as a desired capability and previously had no shortcut
coverage. The commands wrap GitLink's `/api/wiki/open/*` and `/api/wikiExport/*`
endpoints, which require the numeric GitLink project ID in addition to
owner/repo.

## Commands

| Command | Purpose | Endpoint |
|---------|---------|----------|
| `gitlink-cli wiki +list` | List wiki pages | `GET /wiki/open/wikiPages` |
| `gitlink-cli wiki +view` | View a wiki page | `GET /wiki/open/getWiki` |
| `gitlink-cli wiki +create` | Create a wiki page | `POST /wiki/open/createWiki` |
| `gitlink-cli wiki +update` | Update a wiki page | `PUT /wiki/open/updateWiki` |
| `gitlink-cli wiki +delete` | Delete a wiki page | `DELETE /wiki/open/deleteWiki` |
| `gitlink-cli wiki +export` | Export the wiki | `GET /wikiExport/wikiExport-wrapper` |

## Behaviour

- `--project-id` selects the GitLink project ID. When omitted, it is resolved
  from `--owner/--repo` via the repository info endpoint, matching the
  convention used by the `repo` interaction commands.
- Page content for `+create`/`+update` is provided with `--content` (plain text,
  base64-encoded automatically), `--content-file` (read from a file, also
  base64-encoded), or `--content-base64` (already encoded). Precedence is
  `--content-base64` > `--content` > `--content-file`.
- `+create` requires content; `+update` treats content as optional so callers
  can change only the title/message.
- `--title` defaults to the page name when omitted.
- `+create`, `+update`, and `+delete` support `--dry-run` to preview the request
  body without changing remote state.
- `+export` accepts `--type` (`markdown` (default), `pdf`, or `html`) and an
  optional `--project-name` (defaults to the repository name).

## Tests

Unit tests cover endpoint paths, project ID auto-resolution from repo info,
base64 encoding from `--content` and `--content-file`, the create
content-required guard, update without content, dry-run previews, export
defaults and invalid `--type`, and invalid `--project-id`.

## 中文说明

### 变更内容

- 新增 `wiki` 命令组：`+list`、`+view`、`+create`、`+update`、`+delete`、`+export`。
- `--project-id` 省略时自动从 `--owner/--repo` 解析（与 `repo` 互动命令一致）。
- `+create`/`+update` 内容支持 `--content`（纯文本自动 base64）、`--content-file`
  （读文件自动 base64）、`--content-base64`（已编码）；`+create` 必填内容，
  `+update` 内容可选。
- `+create`/`+update`/`+delete` 支持 `--dry-run` 预览请求体。
- `+export` 支持 `--type`（markdown/pdf/html）和 `--project-name`。

### 价值

Wiki 此前无任何 shortcut 封装，是参赛指南点名的能力方向。该命令组让人与 AI Agent
都能直接管理仓库 Wiki，为科研项目文档沉淀与导出等场景提供支撑。
