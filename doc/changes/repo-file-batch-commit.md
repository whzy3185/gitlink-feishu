# Repository File Search and Batch Commit Shortcuts

## Summary

This change adds repository file workflow shortcuts for users and automation agents that need to find files and commit multiple file changes without manually assembling Raw API calls.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli repo +files` | Search repository files by name with optional branch, tag, or commit filtering |
| `gitlink-cli repo +commit-files` | Commit one file operation or a batch JSON operation list through the `contents/batch` API |

## Examples

Search files on a branch:

```bash
gitlink-cli repo +files --owner Gitlink --repo forgeplus --search README --ref main
```

Update one text file:

```bash
gitlink-cli repo +commit-files --owner Gitlink --repo forgeplus \
  --branch main --message "docs: update README" \
  --path README.md --content "# Project"
```

Create or update a binary file by reading local bytes and encoding them as base64:

```bash
gitlink-cli repo +commit-files --owner Gitlink --repo forgeplus \
  --branch main --message "assets: update logo" \
  --action update --path assets/logo.png --from ./logo.png --encoding base64
```

Preview a multi-file commit before sending it:

```bash
gitlink-cli repo +commit-files --owner Gitlink --repo forgeplus \
  --branch main --new-branch docs/batch-update \
  --message "docs: batch update" --ops changes.json --dry-run
```

`changes.json` can be either an array of file operations:

```json
[
  {
    "action_type": "create",
    "file_path": "docs/guide.md",
    "content": "# Guide\n",
    "encoding": "text"
  },
  {
    "action_type": "delete",
    "file_path": "docs/old-guide.md"
  }
]
```

or an object with a `files` array. The CLI supplies `branch`, `message`, optional author and committer fields, and optional `new_branch` from command flags.

## Validation

- `repo +commit-files` requires `--branch` and `--message`.
- Single-file mode requires `--path`; `create` and `update` require exactly one of `--content` or `--from`.
- `delete` operations reject `content` and `encoding` so the request body matches the API intent.
- `--encoding` accepts only `text` and `base64`.
- `--ops` cannot be combined with single-file flags.
- Author and committer names must be provided together with their matching email fields.
- `--dry-run` prints the resolved request and does not call the remote API.

## Tests

Unit tests cover file search query mapping, single-file request bodies, base64 local file reading, JSON batch operation files, dry-run behavior, and validation failures that must not perform an API request.

## 中文说明

本次变更补齐了仓库文件工作流中常用的两个能力：先用 `repo +files` 按文件名和分支搜索仓库文件，再用 `repo +commit-files` 把单个或多个文件变更提交到目标分支。批量提交支持创建新分支、设置提交信息、指定作者和提交者、从本地文件读取内容、对二进制内容做 base64 编码，并提供 `--dry-run` 预览请求体，适合脚本、CI 和 AI Agent 在真正写入仓库前检查即将提交的内容。验证覆盖了端点路径、查询参数、请求体字段、JSON 批量文件、base64 编码和无效参数不触网等关键路径。
