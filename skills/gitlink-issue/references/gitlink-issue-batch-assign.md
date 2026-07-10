# issue +batch-assign

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

批量分配 Issue 经办人。支持两种模式：统一模式（所有 Issue 分配同一人）、CSV 模式（每行指定不同经办人）。

## 命令

```bash
# 统一模式：所有 Issue 分配给同一人
gitlink-cli issue +batch-assign --numbers 42,43 --assignee zhangsan --dry-run

# CSV 模式：每行指定不同经办人
gitlink-cli issue +batch-assign --from assign.csv --dry-run

# 确认执行
gitlink-cli issue +batch-assign --numbers 42,43 --assignee zhangsan --confirm
```

## CSV 格式（CSV 模式）

必须包含 `number`（或别名）列和 `assignee`（或 `assignee_id`、`assigned_to_id`）列：

```csv
number,assignee
42,zhangsan
43,lisi
44,wangwu
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--numbers, -n` | 否 | 逗号分隔的 Issue 编号 |
| `--from` | 否 | CSV 文件路径（CSV 模式） |
| `--search` | 否 | 搜索关键词 |
| `--state` | 否 | 配合 `--search` 过滤状态 |
| `--assignee, -a` | 统一模式必填 | 经办人用户名或 ID |
| `--dry-run` | 否 | 仅预览 |
| `--confirm` | 否 | 确认执行 |
| `--max` | 否 | 最大处理数量（默认 100） |
| `--delay` | 否 | 请求间隔毫秒数（默认 0） |
| `--owner` | 否 | 仓库所有者（自动解析） |
| `--repo` | 否 | 仓库名称（自动解析） |
| `--format` | 否 | 输出格式 |
| `--debug` | 否 | 调试输出 |

## 经办人解析

支持用户名和数字 ID。用户名通过 `GET /users/search` API 自动解析为 ID，结果有进程级缓存。

## API

逐条 PATCH：`PATCH /v1/{owner}/{repo}/issues/{number}`，合并 `assigner_ids` 字段。

## Workflow

1. 确认目标 Issue 和经办人。
2. 先 `--dry-run` 预览。
3. 确认后加 `--confirm` 执行。

> [!CAUTION]
> `--confirm` 执行是 **写操作**。
