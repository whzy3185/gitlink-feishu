# 新增 `api --paginate` 自动翻页

`gitlink-cli api GET <PATH> --paginate` 对齐 `gh api --paginate`：自动逐页抓取并把所有条目拼接成一个数组输出，免去手动传 `page`/`limit` 逐页拉取。仅支持 GET，其它方法会明确报错而不是静默退化。

配套修复了 `PaginateAll` 无法解包真实 GitLink 列表响应的问题。此前它只认顶层裸数组或 `data` 键下的数组，而 GitLink 列表接口把数组包在资源专属键下（`{"total_count":N,"pulls":[...]}`、`{"issues":[...]}`、`{"branches":[...]}` 等），这类响应会被当成单个对象直接返回、根本不翻页。现在解析顺序为：优先取 `data` 数组；否则取 map 中唯一的数组字段（覆盖 pulls/issues/branches/labels 等）；无数组字段或存在多个数组字段（歧义）时，保留“单对象作为单元素返回”的旧行为。短页终止（本页条目数小于 limit 即停止）与既有的裸数组、`data` 包裹用例保持不变。

本次变更包含 `PaginateAll` 解包逻辑修复、`--paginate` 标志与 `runAPIPaginate` 路由、中英文帮助文案，以及单元测试：client 层验证 `{total_count, issues:[...]}` 两页拼接并正确解包 `issues`；cmd 层端到端验证 `--paginate` 合并多页输出与非 GET 报错。
