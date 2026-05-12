# issue +view

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

View details of a specific issue by its number (as shown in the web URL).

## 命令

```bash
# View issue #4
gitlink-cli issue +view -n 4

# View issue with JSON output
gitlink-cli issue +view -n 4 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--number, -n` | **是** | Issue 编号（网页 URL 中的序号，如 `issues/4` 中的 `4`） |
| `--owner` | 否 | 仓库所有者（自动从 git remote 解析） |
| `--repo` | 否 | 仓库名称（自动从 git remote 解析） |
| `--format` | 否 | 输出格式: `json`/`table`/`yaml` |
| `--debug` | 否 | 开启调试输出 |

## API

```
GET /v1/{owner}/{repo}/issues/{number}
```

## References

- [gitlink-issue](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
