# issue +comment

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

Add a comment to an existing issue.

## 命令

```bash
# Add a comment to issue #4
gitlink-cli issue +comment -n 4 -b "This has been fixed in commit abc123"

# Add a multi-line comment
gitlink-cli issue +comment -n 4 -b "Investigation results:
- Root cause: null pointer in auth module
- Fix: add nil check before dereference"
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--number, -n` | **是** | Issue 编号（网页 URL 中的序号） |
| `--body, -b` | **是** | 评论内容（映射为 v1 API 字段 `notes`） |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

## API

```
POST /v1/{owner}/{repo}/issues/{number}/journals
Body: { "notes": body }
```

## Workflow

1. **Confirm** the comment content with the user before posting.
2. **Execute** `gitlink-cli issue +comment -n {number} -b "..."`.
3. **Report** that the comment was added successfully.

> [!CAUTION]
> This is a **Write Operation** -- confirm user intent before executing.

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
