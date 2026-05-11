# issue +batch-close

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

批量关闭 Issue。支持直接传入 Issue 编号列表、从 CSV 读取 Issue 编号，以及用 `--dry-run` 安全预览。

> **Issue 编号说明：** `--numbers` 使用的是网页 URL 中可见的 Issue 编号，即 v1 API 的 `project_issues_index`，不是数据库内部 ID。

## 命令

```bash
# 预览，不修改数据
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --numbers 42,43 --dry-run

# 按 Issue 编号批量关闭
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --numbers 42,43

# 从 CSV 文件读取 Issue 编号
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --from issues.csv
```

## CSV 格式

CSV 文件可以包含 `number`、`issue_number` 或 `project_issues_index` 列：

```csv
number,title
42,stale issue
43,duplicate issue
```

如果没有表头，则默认第一列是 Issue 编号：

```csv
42,stale issue
43,duplicate issue
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--numbers, -n` | 否 | 逗号分隔的 Issue 编号，例如 `1,2,3` |
| `--from` | 否 | 包含 Issue 编号的 CSV 文件 |
| `--dry-run` | 否 | 仅预览计划操作，不关闭 Issue |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

`--numbers` 和 `--from` 至少提供一个。两者同时提供时，会按顺序合并并去重。

## 输出

命令会输出批量操作汇总：

```json
{
  "repository": "Gitlink/forgeplus",
  "dry_run": true,
  "total": 2,
  "succeeded": 2,
  "failed": 0,
  "results": [
    {"number": "42", "action": "close", "status": "planned"},
    {"number": "43", "action": "close", "status": "planned"}
  ]
}
```

## API

每个 Issue 复用 `issue +close` 的安全关闭流程：

1. **读取**当前 Issue，保留 `subject` 和 `description`：
   ```
   GET /v1/{owner}/{repo}/issues/{number}
   ```
2. **更新**Issue 状态为关闭，同时保留原有描述：
   ```
   PATCH /v1/{owner}/{repo}/issues/{number}
   Body: { "subject": <current subject>, "description": <current description>, "status_id": 5 }
   ```

## Workflow

1. 与用户确认目标仓库和要关闭的 Issue 编号。
2. 先执行 `--dry-run` 并展示计划结果。
3. 用户确认后，再执行不带 `--dry-run` 的命令。
4. 汇报成功数量、失败数量和失败原因。

> [!CAUTION]
> 不带 `--dry-run` 是 **写操作**，执行前必须确认用户意图。

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
