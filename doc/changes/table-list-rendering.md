# `--format table` 支持资源包裹列表响应

## 动机

平台所有分页列表端点的响应形状都是资源包裹 map：
`{"total_count": N, "<资源名>": [...]}`。此前 `printTable` 遇到含嵌套
结构的 map 一律回落 JSON，导致 **所有 list 命令的 `--format table`
实际上从不渲染表格**，与 flag 文案承诺不符。

## 行为

- 检测「恰好一个数组值 + 其余全是标量」的 map（列表响应形状）：
  先打印标量摘要行（如 `total_count: 7`，键排序），再把包裹数组
  渲染为表格。
- 含嵌套对象的 map（如 commit 详情）与既有行为一致回落 JSON。

## 生产实测

`branch +list --format table`（forgeplus）：输出 `total_count: 7`
摘要行 + 7 行分支表格（此前输出整段 JSON）。

## 测试

formatter_test.go 新增 2 个单测：资源包裹列表渲染表格（含摘要行、
不回落 JSON）；嵌套对象 map 仍回落 JSON。
