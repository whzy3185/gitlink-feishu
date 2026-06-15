# 表格输出确定性排序与错误状态码展示

## 背景

`--format table` 的渲染依赖 Go map 的遍历顺序，而 Go 的 map 遍历是**随机的**：

- `printMapTable`（单对象 KEY/VALUE 表）每次运行的行顺序都不一样；
- `collectKeys` 在补全非优先列时直接 `for k := range m`，导致 `printSliceTable`（列表表）优先列之后的**列顺序**也随机。

这会让同一条命令两次运行的表格输出不一致，难以肉眼对比、`diff`、截图或在脚本/测试中稳定断言。

此外，table 模式的错误输出只显示 `Error: <message>`，**不显示状态码**，用户难以快速区分 404 / 422 / 500。

## 变更

- `collectKeys`：优先列（id/name/login/title/status/state/created_at/updated_at）之后的剩余列改为 `sort.Strings` 排序，列顺序稳定且可预期。
- `printMapTable`：改为按 `collectKeys` 的顺序输出，行顺序确定，并与列表表的列顺序保持一致。
- 错误输出：当存在错误码时显示 `Error [<code>]: <message>`（无错误码时保持 `Error: <message>`），便于快速识别状态码。

不影响 json / yaml 输出，也不改变成功数据的内容，仅稳定其呈现顺序与丰富错误提示。

## 测试

`internal/output/formatter_test.go` 新增：
- `collectKeys` 非优先列按字典序排序；
- 单对象表渲染 25 次输出完全一致（确定性）；
- 列表表表头顺序 25 次渲染一致；
- 错误输出包含状态码 `Error [500]: ...`，无错误码时回退为 `Error: ...`。

`go build ./...`、`go vet ./...`、`go test ./...`、`gofmt -s` 全部通过。
