# 全局 `--jq` 输出字段提取

## 动机

对标 `gh --jq`：Agent 与 shell 脚本消费 CLI 输出时经常只需要单个字段
（如最新 commit 的 sha、issue 总数），此前必须依赖外部 `jq` 或自行解析
完整 JSON 信封。新增零依赖的点分路径提取，纯 Go 标准库实现，
不引入任何第三方 JSON 查询库（避免供应链风险）。

## 行为

- 新增全局持久 flag `--jq`，对所有命令生效（不占用 `--query`，与 api 子命令的查询参数 flag 无冲突）。
- 路径语法：点分段；段为对象键，或非负整数作为数组下标。
  例：`--jq data.commits.0.sha`、`--jq data.total_issues_count`。
- 标量（字符串/数字/布尔/null）输出裸值，方便 shell 管道直接消费；
  对象与数组输出缩进 JSON。
- 错误信息可操作：键不存在时列出该层全部可用键（排序后）；
  数组下标越界/非数字段给出数组长度提示。

## 生产实测

- `issue +list --limit 2 --jq data.issues.0.subject` → 裸标题字符串
- `issue +list --jq data.total_issues_count` → `4197`
- 误键 `--jq data.issues.0.name` → 报错并列出 29 个可用键（含 `subject`）

## 测试

`internal/output/query_test.go` 6 个单测：字符串/数字标量裸值输出、
对象 JSON 输出、缺键错误含可用键列表、下标越界、数组段非数字。
