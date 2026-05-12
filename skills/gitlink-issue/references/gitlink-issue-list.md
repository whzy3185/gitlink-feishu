# issue +list

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

List issues for the current repository, with optional state filter and pagination. Uses the v1 API which returns `project_issues_index` (the issue number visible in the web URL).

## 命令

```bash
# List open issues (default)
gitlink-cli issue +list

# List closed issues
gitlink-cli issue +list -s closed

# List all issues, page 2, 10 per page
gitlink-cli issue +list -s all -p 2 -l 10

# Output as JSON
gitlink-cli issue +list --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--state, -s` | 否 | 按状态过滤: `open`、`closed`、`all`（默认 `open`） |
| `--page, -p` | 否 | 页码（默认 `1`） |
| `--limit, -l` | 否 | 每页数量（默认 `20`） |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

## API

```
GET /v1/{owner}/{repo}/issues?state={state}&page={page}&limit={limit}
```

## 返回字段（v1）

每个 Issue 包含以下关键字段：

| 字段 | 说明 |
|------|------|
| `project_issues_index` | Issue 编号（网页 URL 中的序号） |
| `id` | 数据库内部 ID |
| `subject` | 标题 |
| `status_name` / `status_id` | 状态名 / 状态 ID |
| `author.login` | 作者 |
| `assigners` | 负责人列表 |
| `created_at` / `updated_at` | 创建 / 更新时间 |

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
