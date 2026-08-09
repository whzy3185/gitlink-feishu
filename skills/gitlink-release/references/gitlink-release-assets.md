# release +assets

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

列出指定发布的附件和源码包下载地址。`--id` 必须使用 `version_id`，可从 `release +list` 的返回中获取。

## 命令

```bash
gitlink-cli release +assets --owner someone --repo myrepo --id 12345
gitlink-cli release +assets --id 12345 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id, -i` | 是 | 发布 `version_id` |
| `--owner` | 是* | 仓库所有者（可从 git remote 自动推断） |
| `--repo` | 是* | 仓库名称（可从 git remote 自动推断） |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |

> *如果在 GitLink 仓库目录下执行，`--owner` 和 `--repo` 可自动推断。

## References
- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
