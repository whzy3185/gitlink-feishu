---
name: gitlink-issue
version: 2.1.0
description: "GitLink Issue 管理：创建、查看、更新、关闭、评论，以及批量关闭、批量评论、批量更新。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-issue

> 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，确认认证方式、全局参数和安全约束。

## 适用场景

- 维护者需要快速创建、更新、关闭或评论 Issue。
- Agent 需要基于仓库内的 Issue 批量做运营动作，例如统一补评论、统一更新优先级、统一补截止日期。
- 需要从 Markdown 文件读取长文本，避免把大段内容直接塞进命令行参数。

## 常用命令

| Shortcut | 用途 | 是否写操作 |
| --- | --- | --- |
| `issue +list` | 列出 Issue | 否 |
| `issue +view` | 查看单个 Issue 详情 | 否 |
| `issue +create` | 创建 Issue，支持 `--body-file` | 是 |
| `issue +update` | 更新单个 Issue，支持 `--body-file` | 是 |
| `issue +close` | 关闭单个 Issue | 是 |
| `issue +comment` | 给单个 Issue 添加评论，支持 `--body-file` | 是 |
| `issue +batch-close` | 按编号或 CSV 批量关闭 Issue | 是 |
| `issue +batch-comment` | 按编号或 CSV 批量评论 | 是 |
| `issue +batch-update` | 按编号或 CSV 批量更新元数据 | 是 |
| `issue +assigners` | 列出可分配负责人 | 否 |
| `issue +authors` | 列出 Issue 作者 | 否 |
| `issue +priorities` | 列出优先级 | 否 |
| `issue +tags` | 列出标签 | 否 |
| `issue +statuses` | 列出状态 | 否 |

## 使用方式

```bash
# 创建一个 Issue
gitlink-cli issue +create --owner Gitlink --repo forgeplus \
  --title "Bug: 登录失败" \
  --body-file issue.md

# 更新单个 Issue 的优先级和截止日期
gitlink-cli issue +update --owner Gitlink --repo forgeplus \
  --number 42 \
  --priority-id 4 \
  --due-date 2026-06-15

# 通过文件给单个 Issue 添加长评论
gitlink-cli issue +comment --owner Gitlink --repo forgeplus \
  --number 42 \
  --body-file comment.md

# 预览批量评论
gitlink-cli issue +batch-comment --owner Gitlink --repo forgeplus \
  --numbers 42,43,44 \
  --body-file comment.md \
  --dry-run

# 从 CSV 批量更新 Issue
gitlink-cli issue +batch-update --owner Gitlink --repo forgeplus \
  --from issues.csv \
  --state closed \
  --priority-id 4 \
  --due-date 2026-06-15
```

## 批量操作约定

- `--numbers` 使用网页 URL 中可见的 Issue 编号，不是数据库内部 ID。
- `--from` 支持 CSV 文件，优先识别 `number`、`issue_number`、`project_issues_index` 列；如果没有表头，则默认第一列为 Issue 编号。
- `issue +batch-comment` 和 `issue +batch-update` 会逐条执行，并输出每条 Issue 的结果汇总。
- 批量命令支持 `--dry-run`，推荐先预览再真实执行。

## 文本输入约定

- `issue +create`、`issue +update`、`issue +comment`、`issue +batch-comment`、`issue +batch-update` 都支持 `--body-file`。
- `--body` 和 `--body-file` 互斥，避免正文来源不明确。
- 长文本优先使用 `--body-file`，便于保留换行和 Markdown 格式。

## 安全建议

- 对写操作先确认仓库、Issue 编号和目标字段。
- 批量命令先跑 `--dry-run`，确认数量和目标无误后再执行真实写入。
- `issue +update` 和 `issue +batch-update` 会先读取当前 Issue，再带上现有标题/描述/元数据发起 PATCH，避免误清空字段。

## 输出与自动化

- 所有命令都支持全局 `--format json|table|yaml`。
- 批量命令输出统一包含 `repository`、`action`、`dry_run`、`total`、`succeeded`、`failed` 和逐条 `results`，适合脚本和 Agent 继续处理。
