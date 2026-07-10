# issue +batch-open

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

批量重新打开已关闭的 Issue。

## 命令

```bash
# 预览
gitlink-cli issue +batch-open --numbers 42,43 --dry-run

# 确认重新打开
gitlink-cli issue +batch-open --numbers 42,43 --confirm

# 从 CSV 读取
gitlink-cli issue +batch-open --from issues.csv --dry-run

# 按搜索条件
gitlink-cli issue +batch-open --search "已修复" --state closed --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--numbers, -n` | 否 | 逗号分隔的 Issue 编号 |
| `--from` | 否 | CSV 文件路径 |
| `--search` | 否 | 搜索关键词 |
| `--state` | 否 | 配合 `--search` 过滤状态 |
| `--label` | 否 | 配合 `--search` 过滤标签 |
| `--dry-run` | 否 | 仅预览 |
| `--confirm` | 否 | 确认执行 |
| `--max` | 否 | 最大处理数量（默认 100） |
| `--delay` | 否 | 请求间隔毫秒数（默认 0） |
| `--owner` | 否 | 仓库所有者（自动解析） |
| `--repo` | 否 | 仓库名称（自动解析） |
| `--format` | 否 | 输出格式 |
| `--debug` | 否 | 调试输出 |

`--numbers`、`--from`、`--search` 至少提供一个，可同时使用（结果会合并去重）。

## API

逐条 PATCH：`PATCH /v1/{owner}/{repo}/issues/{number}`，`status_id: 1`。

## Workflow

1. 与用户确认目标 Issue。
2. 先 `--dry-run` 预览。
3. 用户确认后加 `--confirm` 执行。
4. 汇报结果。

> [!CAUTION]
> `--confirm` 执行是 **写操作**，执行前必须确认用户意图。
