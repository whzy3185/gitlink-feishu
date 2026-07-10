# issue +batch-update

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

支持两种模式：
- **`--ids` 统一模式**：所有 Issue 统一更新相同字段，一次服务端批量 API 调用
- **`--from CSV` 模式**：逐条差异化更新，每条 Issue 可以有不同修改

## 命令

### --ids 统一模式

```bash
# 预览
gitlink-cli issue +batch-update --ids 10,20,30 --status closed --dry-run

# 批量关闭并指定里程碑
gitlink-cli issue +batch-update --ids 10,20,30 --status closed --milestone 5

# 批量更新标签和负责人
gitlink-cli issue +batch-update --ids 10,20,30 --labels 1,2 --assignees 100
```

### --from CSV 模式

```bash
# 预览
gitlink-cli issue +batch-update --from updates.csv --dry-run

# 确认执行
gitlink-cli issue +batch-update --from updates.csv --confirm
```

## 参数

| 参数 | 短选项 | 模式 | 必填 | 说明 |
|------|--------|------|------|------|
| `--ids` | `-i` | 统一 | 统一模式必填 | 逗号分隔的 Issue ID |
| `--status` | `-s` | 统一 | 否 | 新状态：open / closed |
| `--priority` | `-p` | 统一 | 否 | 优先级 ID |
| `--milestone` | `-m` | 统一 | 否 | 里程碑 ID |
| `--labels` | `-l` | 统一 | 否 | 逗号分隔的标签 ID |
| `--assignees` | `-a` | 统一 | 否 | 逗号分隔的负责人用户 ID |
| `--from` | | CSV | CSV 模式必填 | CSV 文件路径 |
| `--dry-run` | | 共享 | 否 | 仅预览，不实际修改 |
| `--confirm` | | CSV | 否 | CSV 模式确认执行 |
| `--max` | | CSV | 否 | 最大处理数量（默认 100） |
| `--delay` | | CSV | 否 | 请求间隔毫秒数（默认 0） |
| `--owner` | | 共享 | 否 | 仓库所有者（自动解析） |
| `--repo` | | 共享 | 否 | 仓库名称（自动解析） |
| `--format` | | 共享 | 否 | 输出格式 |
| `--debug` | | 共享 | 否 | 调试输出 |

`--ids` 和 `--from` 二选一，决定使用哪种模式。

## CSV 格式（--from 模式）

CSV 第一行为列名，后续每行为一条更新。列名支持中英文别名：

| 列名 | 别名 | API 字段 | 说明 |
|------|------|----------|------|
| title | | subject | 新标题 |
| body | | description | 新描述 |
| state | | status_id | open/closed 或数字状态 ID |
| assignee | | assigner_ids | 经办人用户名或 ID |
| milestone | milestone_id | milestone_id | 里程碑名称或 ID |
| label | labels | issue_tag_ids | 标签名称或 ID，逗号分隔 |
| priority | priority_id | priority_id | 优先级数字 ID |

**必须包含 `number` / `issue_number` / `project_issues_index` 列**。空值列表示不修改该字段。

```csv
number,title,state,assignee,milestone,label,priority
42,修复后的标题,closed,zhangsan,v2.0,bug,3
43,,open,lisi,v1.0,,
```

## API

| 模式 | API |
|------|-----|
| `--ids` 统一 | `PATCH /v1/{owner}/{repo}/issues/batch_update`（单次服务端批量调用） |
| `--from CSV` | 逐条 `PATCH /v1/{owner}/{repo}/issues/{number}` |

## Workflow

1. 与用户确认目标仓库和更新方式（`--ids` 或 `--from CSV`）。
2. 先 `--dry-run` 预览。
3. 确认后执行（CSV 模式需额外 `--confirm`）。
4. 汇报结果。

> [!CAUTION]
> 不带 `--dry-run` 的执行是 **写操作**，执行前必须确认用户意图。
