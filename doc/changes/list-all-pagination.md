# list 命令 --all 自动翻页

## 背景

`issue +list`、`pr +list`、`branch +list`、`release +list` 此前一次只能取一页，
用户或 AI Agent 想拿到全量列表必须手动循环 `--page`。代码中虽有 `PaginateAll`
翻页助手，但它只识别 `data` 包裹键；而 GitLink 生产 API 的列表响应实际用
资源名包裹数组（如 `{"total_count":N,"issues":[...]}`、`"pulls"`、`"branches"`、
`"releases"`），导致该助手在真实端点上退化为「单对象」返回，从未被任何命令使用。

## 变更内容

- `internal/client`：翻页助手对齐生产响应形状
  - 新增 `PaginateAllKey(path, params, listKey)`：按指定资源键提取数组；
    `listKey` 为空时自动探测（顶层数组 / `data` 包裹 / 唯一数组字段）。
  - 遵循 `total_count`：达到总数即停止；另设最大页数护栏，防止
    忽略 `page` 参数的端点造成死循环。
  - `PaginateAll` 保持原签名，委托给 `PaginateAllKey`。
- 九个分页 list 命令新增 `--all` 布尔参数（默认 false）：
  - `issue +list --all`（合并结果同样应用 number/database_id 规范化）
  - `pr +list --all`、`branch +list --all`、`release +list --all`
  - `milestone +list --all`、`org +list --all`、`repo +list --all`
  - `search +repos --all`、`search +users --all`
  - 对应资源键：`issues`/`pulls`/`branches`/`releases`/`milestones`/
    `organizations`/`projects`/`users`（均生产实测确认）
  - 输出与单页响应同构：`{"total_count": N, "<资源名>": [...]}`。
- 总数字段兼容 `total_count` 与 `count`（如 `/users/:login/projects`）。
- 中英文 i18n 新增 `flag.all` 文案。

## 命令示例

```bash
# 拉取仓库全部 open issue（自动翻页合并）
gitlink-cli issue +list --state open --all --format json

# 全部分支 / 全部 PR / 全部 release / 全部里程碑
gitlink-cli branch +list --all
gitlink-cli pr +list --state all --all
gitlink-cli release +list --all
gitlink-cli milestone +list --all

# 全部组织 / 某用户全部仓库 / 搜索结果全量
gitlink-cli org +list --all
gitlink-cli repo +list --user Taoyouce --all
gitlink-cli search +repos --keyword gitlink --all
```

## 测试

- `internal/client/pagination_test.go`：资源键包裹多页合并、`total_count`
  截断（模拟忽略 page 的异常端点）、`data` 包裹、唯一数组字段自动探测、
  单对象回退、指定键缺失回退，共 6 个用例。
- `shortcuts/issue`：`--all` 端到端用例验证按页请求序列与合并。
- `go test ./...`、`go vet`、`gofmt` 全部通过。
