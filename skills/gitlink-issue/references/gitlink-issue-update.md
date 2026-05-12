# issue +update

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

Update an existing issue's title, description, or state.

The shortcut reads the current issue first and preserves existing `subject` and `description` unless the user explicitly changes them. This prevents partial updates from clearing the issue description.

## 命令

```bash
# Update issue title
gitlink-cli issue +update -n 4 -t "New title"

# Update issue description
gitlink-cli issue +update -n 4 -b "Updated description"

# Update both title and state
gitlink-cli issue +update -n 4 -t "Revised title" -s closed
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--number, -n` | **是** | Issue 编号（网页 URL 中的序号） |
| `--title, -t` | 否 | 新标题（映射为 API 字段 `subject`） |
| `--body, -b` | 否 | 新描述（映射为 API 字段 `description`） |
| `--state, -s` | 否 | 新状态: `open`、`closed`，或数字状态 ID（映射为 API 字段 `status_id`；open=1，closed=5） |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

## API

```
PATCH /v1/{owner}/{repo}/issues/{number}
Body: { "subject": current_or_new_title, "description": current_or_new_body, "status_id": state }
```

## Workflow

1. **Confirm** the fields to update and their new values with the user.
2. **Execute** `gitlink-cli issue +update -n {number} -t "..." -b "..."`.
3. **Report** the updated issue details to the user.

When using Raw API instead of the shortcut, fetch the issue first and include the current `subject` and `description` in the update payload.

> [!CAUTION]
> This is a **Write Operation** -- confirm user intent before executing.

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
