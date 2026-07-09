# Fix: Windows Git Bash 下 `gitlink-cli api` 路径被 MSYS2 污染导致 404

## 问题

在 Windows Git Bash（MSYS2）环境下执行：

```bash
gitlink-cli api GET /v1/owner/repo
```

路径参数 `/v1/owner/repo` 会被 MSYS2 自动改写为类似 `C:/Program Files/Git/v1/owner/repo` 的 Windows 路径——MSYS2 把以 `/` 开头的命令行参数当成 Unix 路径，转换为 Git 安装目录。结果 API 请求路径错误，返回 404。影响所有 Windows Git Bash 用户。

## 根因

MSYS2 的 POSIX→Windows 路径转换会对命令行参数中以 `/` 开头的字符串生效，且无法通过 shell 转义稳定规避（`MSYS_NO_PATHCONV` 等环境变量依赖用户配置，不可靠）。

## 修复

在 `cmd/api` 的 `runAPI` 中，对取到的 `path` 调用 `restoreAPIPath` 还原：

- 检测首部是否为盘符（正则 `^[A-Za-z]:/`）；
- 若是，按常见 API 前缀（`/v1/` `/v2/` `/api/` `/users/` `/projects/`）在污染后的路径里定位原始起点并截取；
- 无盘符或无匹配前缀时原样返回，不影响其他平台与正常路径。

`restoreAPIPath` 为纯函数，便于单元测试。

## 影响

仅 Windows 受益，其他平台行为不变。改动集中在 `cmd/api/api.go`（约 +25 行，含函数与注释）。

## Tests

```bash
go test ./cmd/api/... -run TestRestoreAPIPath -v
```
