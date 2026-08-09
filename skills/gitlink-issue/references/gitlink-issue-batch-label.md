# issue +batch-label

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

批量操作 Issue 标签，支持三种模式：
- **add**：在现有标签基础上追加
- **remove**：从现有标签中移除
- **set**：替换为指定标签（覆盖现有）

## 命令

```bash
# 添加标签
gitlink-cli issue +batch-label --numbers 42,43 --action add --labels bug,urgent --dry-run

# 移除标签
gitlink-cli issue +batch-label --numbers 42,43 --action remove --labels deprecated --dry-run

# 替换标签
gitlink-cli issue +batch-label --numbers 42,43 --action set --labels bug,feature --dry-run

# 使用数字标签 ID
gitlink-cli issue +batch-label --numbers 42,43 --action add --label-ids 1,2 --confirm
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--numbers, -n` | 否 | 逗号分隔的 Issue 编号 |
| `--from` | 否 | CSV 文件路径 |
| `--search` | 否 | 搜索关键词 |
| `--state` | 否 | 配合 `--search` 过滤状态 |
| `--action, -a` | 是 | 操作类型：add / remove / set |
| `--labels, -l` | 否 | 标签名称列表，逗号分隔 |
| `--label-ids` | 否 | 标签 ID 列表，逗号分隔（与 `--labels` 互斥） |
| `--dry-run` | 否 | 仅预览 |
| `--confirm` | 否 | 确认执行 |
| `--max` | 否 | 最大处理数量（默认 100） |
| `--delay` | 否 | 请求间隔毫秒数（默认 0） |
| `--owner` | 否 | 仓库所有者（自动解析） |
| `--repo` | 否 | 仓库名称（自动解析） |
| `--format` | 否 | 输出格式 |
| `--debug` | 否 | 调试输出 |

`--labels` 和 `--label-ids` 必须提供其一，不可同时使用。

## 标签解析

标签名称通过 `GET /{owner}/{repo}/labels` API 自动解析为 ID，结果按仓库维度缓存。

## API

逐条 PATCH：`PATCH /v1/{owner}/{repo}/issues/{number}`，先读取当前标签列表，再根据 action 合并/移除/替换 `issue_tag_ids`。

## Workflow

1. 确认目标 Issue、操作类型和标签。
2. 先 `--dry-run` 预览。
3. 确认后加 `--confirm` 执行。

> [!CAUTION]
> `--confirm` 执行是 **写操作**。
