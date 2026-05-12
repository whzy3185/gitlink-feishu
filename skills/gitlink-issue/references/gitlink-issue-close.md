# issue +close

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

Close an issue. Automatically fetches the current issue subject and description, then sets `status_id=5` (closed) without clearing the description.

## 命令

```bash
# Close issue #4
gitlink-cli issue +close -n 4
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--number, -n` | **是** | Issue 编号（网页 URL 中的序号） |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

## API

The command performs two API calls:

1. **Fetch** the issue to get the current `subject` and `description`:
   ```
   GET /v1/{owner}/{repo}/issues/{number}
   ```
2. **Update** the issue with `status_id=5`, preserving the current description:
   ```
   PATCH /v1/{owner}/{repo}/issues/{number}
   Body: { "subject": <current subject>, "description": <current description>, "status_id": 5 }
   ```

## Workflow

1. **Confirm** with the user which issue to close (by number).
2. **Execute** `gitlink-cli issue +close -n {number}`.
3. **Report** that the issue has been closed successfully.

> [!CAUTION]
> This is a **Write Operation** -- confirm user intent before executing.

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
