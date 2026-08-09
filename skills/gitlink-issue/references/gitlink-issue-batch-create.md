# issue +batch-create

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

通过 CSV 文件批量创建 Issue。每行 CSV 对应一个 Issue。

## 命令

```bash
# 打印 CSV 格式模板
gitlink-cli issue +batch-create --print-schema

# 预览
gitlink-cli issue +batch-create --from issues.csv --dry-run

# 确认创建
gitlink-cli issue +batch-create --from issues.csv --confirm
```

## CSV 格式

| 列名 | 别名 | 必需 | 说明 |
|------|------|------|------|
| title | subject | 是 | Issue 标题 |
| body | description | 否 | Issue 描述 |
| assignee | assignee_id | 否 | 经办人用户名或 ID |
| milestone | fixed_version_id, milestone_id | 否 | 里程碑名称或 ID |
| label | labels | 否 | 标签名称，逗号分隔 |
| priority | priority_id | 否 | 优先级 ID（默认 2=normal） |

```csv
title,body,assignee,milestone,label,priority
登录页面崩溃,复现步骤：点击登录按钮后白屏,zhangsan,v2.0,bug,3
新增导出功能,支持 CSV 和 Excel 导出,lisi,v2.1,feature,2
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--from, -f` | 是 | CSV 文件路径 |
| `--print-schema` | 否 | 打印 CSV 列名模板并退出 |
| `--dry-run` | 否 | 仅预览，不实际创建 |
| `--confirm` | 否 | 确认执行 |
| `--max` | 否 | 最大创建数量（默认 100） |
| `--delay` | 否 | 请求间隔毫秒数（默认 0） |
| `--owner` | 否 | 仓库所有者（自动解析） |
| `--repo` | 否 | 仓库名称（自动解析） |
| `--format` | 否 | 输出格式 |
| `--debug` | 否 | 调试输出 |

## API

逐条 POST：`POST /v1/{owner}/{repo}/issues`，每条创建时自动设置 `status_id: 1`（新建）、`priority_id: 2`（正常）、`done_ratio: 0`。

## Workflow

1. 准备 CSV 文件（可用 `--print-schema` 查看格式）。
2. 先 `--dry-run` 预览。
3. 确认后加 `--confirm` 执行。
4. 汇报创建结果。

> [!CAUTION]
> `--confirm` 执行是 **写操作**，会实际创建 Issue。
