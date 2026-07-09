# File shortcut

新增 `file` Shortcut 组，补齐 GitLink 仓库文件与目录内容操作的常用封装：

- `file +list`   列出仓库文件（`--ref` 指定分支/标签/commit，`--search` 关键词过滤）
- `file +tree`   列出文件树（`--sha` 默认 master，`--recursive` 递归，支持分页）
- `file +get`    获取文件或目录内容（`--path` 必填，`--ref` 默认 master）
- `file +create` 创建文件（`--path`/`--content`/`--message` 必填，content 自动 Base64 编码）
- `file +delete` 删除文件（`--path`/`--sha`/`--message` 必填，SHA 取自 `file +list`）

实现要点：

- `+tree` 走 `/v1/{owner}/{repo}/git/trees/{sha}`，与 git 树对象语义一致，支持 `--recursive` 与分页。
- `+get` / `+list` 经 `/sub_entries`、`/files` 等接口读取文件或目录内容。
- `+create` 调用 `/create_file`，文件内容 Base64 编码后提交；`+delete` 调用 `/delete_file`，需先从 `file +list` 取得文件 blob SHA。
- 路径统一使用 `/v1/{owner}/{repo}/` 前缀，与现有 Shortcut 组保持一致。

补充单元测试 `shortcuts/file/file_test.go`，覆盖各命令的参数解析与路径构造。

## Examples

```bash
gitlink-cli file +list --owner Gitlink --repo gitlink-cli
gitlink-cli file +tree --owner Gitlink --repo gitlink-cli --recursive
gitlink-cli file +get  --owner Gitlink --repo gitlink-cli --path README.md
gitlink-cli file +create --owner Gitlink --repo gitlink-cli --path docs/note.md --content "hello" --message "add note"
```

## Tests

```bash
go test ./shortcuts/file/...
```
