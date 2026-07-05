# File Content Shortcuts

## Summary

Adds a `file` shortcut group so users and AI agents can read, search, and write
repository file contents without cloning or falling back to Raw API calls.
Directory listing and README viewing remain covered by `repo +tree` and
`repo +readme`.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli file +view` | View a file's contents; `--raw` prints only the decoded content |
| `gitlink-cli file +search` | Search repository files by name |
| `gitlink-cli file +create` | Create a file and commit it to a branch |
| `gitlink-cli file +update` | Update a file and commit it to a branch |
| `gitlink-cli file +delete` | Delete a file and commit the removal to a branch |

## Validation

- `file +view` accepts `--ref` (branch, tag, or commit SHA) and `--raw`; `--raw`
  fails with a clear error when the path is a directory.
- Write commands require `--path` and `--branch`; `--message` defaults to
  `<action> <path>` when omitted.
- `file +create` / `file +update` accept exactly one of `--content` or
  `--content-file`; providing both or neither is rejected before any request.
- `--new-branch` commits the change to a new branch created from `--branch`.
- File content is transported with `text` encoding (verified against production
  gitlink.org.cn; the documented `base64` encoding is rejected there).

## Tests

Unit tests cover endpoint paths, query parameter mapping, request payload
construction, content-source validation, default commit messages, `--new-branch`
propagation, and raw content extraction from entries/README-shaped responses.

## 中文说明

### 变更内容

- 新增 `file` 快捷命令组：`+view`（查看文件内容，`--raw` 仅输出解码后的正文）、
  `+search`（按文件名搜索）、`+create` / `+update` / `+delete`（通过
  contents/batch API 直接提交文件增删改）。
- 无需克隆仓库即可读写文件，适合 AI Agent 读取 README、修改单个文件等场景
  （响应社区 issue：API 是否支持自动读取仓库内文件）。
- 内容支持 `--content` 内联或 `--content-file` 从本地文件读取（text 编码，
  已在生产环境验证，文档中的 base64 编码在生产环境会被拒绝）；支持
  `--new-branch` 提交到新分支。
- 更新 README 与 README.zh-CN 的功能表和使用示例。

### 国际化

命令与全部 flag 文案已接入 i18n（`cmd.file.*` / `flag.file.*`，含 en-US 与
zh-CN 两套 locale），`GITLINK_LANG=zh-CN` 下 `file --help` 输出中文帮助。

### 验证

- `go test ./...`
- `go vet ./...`
- `go run . file --help`
- `go run . file +view --help`
- 在生产 gitlink.org.cn 真实仓库验证 `+view --raw`、`+search`、`+create`、
  `+update`、`+delete` 全链路
