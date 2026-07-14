# Browse Shortcut

新增 `browse` Shortcut 组，在浏览器中打开仓库的各类页面，对标 `gh browse`：

- `browse +repo`
- `browse +issue --number <n>`
- `browse +pr --number <n>`
- `browse +commit --sha <sha>`
- `browse +branch [--name <branch>]`
- `browse +file --path <path> [--ref <branch>]`
- `browse +releases`
- `browse +wiki`

说明：

- Web 地址由配置的 `base_url` 推导（去掉 `/api` 后缀），因此自建实例同样适用。
- `-n/--no-browser` 只打印地址而不打开浏览器，便于脚本取用与 CI 环境。
- 浏览器启动优先使用 `$BROWSER`，否则回退到平台默认（Windows/macOS/Linux）。

同时补充了 `internal/browser` 跨平台启动器与 URL 构造的单元测试。
