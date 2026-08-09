# api 单次调用模板变量与预演能力

这次改动把 `gitlink-cli api` 的单次调用模式和 batch 模式拉齐了。

- 单次调用现在支持 `--var key=value`，可以在路径、查询参数和 JSON 请求体里复用 `{{var}}` 模板变量。
- 路径里的 `:owner` 和 `:repo` 会自动使用当前 `--owner` / `--repo` 或 git remote 上下文渲染，修复了单次调用不替换占位符的问题。
- `--dry-run` 不再只属于 batch 模式，单次调用也可以先预览渲染后的 method、path、query、body 和 variables，再决定是否真正发请求。

这样做的目的不是单纯补一个 bug，而是让 Raw API 更适合脚本和 Agent 复用：同一份模板写法既能用在 `api --batch-file`，也能平滑退化成一次性的单条请求。

本地验证：

```bash
go test ./cmd/api
go test ./...
go build ./...
git diff --check
go run . api --help
```
