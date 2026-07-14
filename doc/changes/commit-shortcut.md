# Commit shortcut

新增 `commit` Shortcut 组，封装 GitLink 仓库提交（Commit）相关 OpenAPI 的常用操作，支持查看提交列表、单条提交变更文件、提交 Diff 与文件 Blame：

- `commit +list`
- `commit +view`
- `commit +diff`
- `commit +blame`

实现要点：

- `+list` 映射 `GET /v1/{owner}/{repo}/commits`，支持 `--sha`（分支 / 标签 / 提交 SHA 过滤）、`--page`（默认 `1`）、`--limit`（默认 `20`），通过查询参数传给 API。
- `+view` 映射 `GET /v1/{owner}/{repo}/commits/{sha}/files`，返回某次提交涉及的文件清单；`--sha` 为必填，同时支持分页参数。
- `+diff` 映射 `GET /v1/{owner}/{repo}/commits/{sha}/diff`，返回指定提交的 Diff 内容；`--sha` 为必填。
- `+blame` 映射 `GET /v1/{owner}/{repo}/blame`，按文件展示逐行归属；`--path` 为必填，`--sha` 缺省为 `master`，二者以 `filepath` / `sha` 查询参数提交。
- 四个命令均通过 `ResolveOwnerRepo()` 解析 `--owner` / `--repo`（支持 `-R owner/repo` 缩写与仓库默认推断），结果统一经 `ctx.Output(env)` 输出，兼容 `--format`（table / json）等全局参数。
- 路径沿用 `/v1/` 前缀约定，与 webhook / milestone / label 等组保持一致；`.json` 后缀由底层 client 自动补全。

## Examples

```bash
# 列出 develop 分支最近 10 条提交
gitlink-cli commit +list --owner Gitlink --repo forgeplus --sha develop --limit 10

# 查看某次提交涉及的文件
gitlink-cli commit +view --owner Gitlink --repo forgeplus --sha abc123def

# 查看提交 Diff
gitlink-cli commit +diff --owner Gitlink --repo forgeplus --sha abc123def

# 查看 README.md 在 master 分支的 Blame 信息
gitlink-cli commit +blame --owner Gitlink --repo forgeplus --path README.md

# 使用 -R 缩写并以 JSON 输出
gitlink-cli commit +list -R Gitlink/forgeplus --page 2 --format json
```

## Tests

```bash
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/commit/...
go test ./...
go run . commit +list --help
```
