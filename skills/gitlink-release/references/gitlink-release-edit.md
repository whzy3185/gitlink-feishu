# release +edit

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

获取发行版编辑接口返回的数据。该命令使用 `version_id`，通常先通过 `release +list` 获取。

## 命令

```bash
gitlink-cli release +edit --id 12345
gitlink-cli release +edit --id 12345 --owner someone --repo myrepo --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id, -i` | 是 | 发行版 ID（version_id） |
| `--owner` | 是* | 仓库所有者（可从 git remote 自动推断） |
| `--repo` | 是* | 仓库名称（可从 git remote 自动推断） |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |

> *如果在 GitLink 仓库目录下执行，`--owner` 和 `--repo` 可自动推断。

## References
- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
