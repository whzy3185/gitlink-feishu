# issue +export

按 Issue 列表筛选条件跨页导出数据，支持 CSV、JSON 和 Markdown。该命令只读取远端数据，不会修改 Issue，适合生成周报、迁移清单、离线排查表和 AI 分析输入。

```bash
gitlink-cli issue +export --owner Gitlink --repo forgeplus --state open --export-format csv --output issues.csv
gitlink-cli issue +export --owner Gitlink --repo forgeplus --state all --keyword 登录 --fields number,title,status,assignees,updated_at,url --export-format markdown
gitlink-cli issue +export --owner Gitlink --repo forgeplus --tag-ids 1,2 --max 100 --export-format json --output issues.json
```

## 常用参数

| 参数 | 说明 |
|------|------|
| `--state` | 筛选状态：`open`、`closed`、`all` |
| `--keyword` | 按关键词搜索 |
| `--participant` | 参与范围：`all`、`aboutme`、`authoredme`、`assignedme`、`atme` |
| `--author-id` / `--assignee-id` | 按作者或负责人用户 ID 筛选 |
| `--milestone-id` / `--status-id` / `--tag-ids` | 按里程碑、状态或标签筛选 |
| `--sort-by` / `--sort-direction` | 复用 Issue 列表排序参数 |
| `--limit` | 每页数量，上限 100 |
| `--max` | 最多导出的 Issue 数量，`0` 表示不额外限制 |
| `--fields` | 逗号分隔的导出字段 |
| `--export-format` | `csv`、`json` 或 `markdown` |
| `--output` / `-o` | 写入文件；不传时输出到 stdout |

## 字段

默认字段为：

```text
number,title,state,status,priority,author,assignees,tags,updated_at,url
```

可选字段包括：`number`、`database_id`、`title`、`description`、`state`、`status`、`status_id`、`priority`、`priority_id`、`author`、`assignees`、`tags`、`milestone`、`branch`、`created_at`、`updated_at`、`closed_at`、`url`。

## 使用建议

- 需要给维护者或评委提交处理清单时，优先使用 `csv`，便于表格软件打开。
- 需要交给 AI Agent 或脚本继续处理时，使用 `json`。
- 需要直接贴到 Issue、PR 或周报时，使用 `markdown`。
