# release +download

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

下载指定发布的附件或源码包。`--asset` 可传附件 ID 或文件名，`--archive` 可传 `zip` 或 `tar`；两者不能同时使用。

## 命令

```bash
# 下载附件
gitlink-cli release +download --owner someone --repo myrepo --id 12345 --asset app.zip --output dist/

# 下载源码 zip
gitlink-cli release +download --owner someone --repo myrepo --id 12345 --archive zip --output dist/source.zip

# 允许覆盖已有文件
gitlink-cli release +download --id 12345 --asset 441826 --output app.zip --force
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id, -i` | 是 | 发布 `version_id` |
| `--asset, -a` | 否 | 要下载的附件 ID 或文件名 |
| `--archive` | 否 | 下载源码包，取值 `zip` 或 `tar` |
| `--output, -o` | 否 | 输出文件或目录路径，默认当前目录 |
| `--force` | 否 | 允许覆盖已存在的输出文件 |
| `--owner` | 是* | 仓库所有者（可从 git remote 自动推断） |
| `--repo` | 是* | 仓库名称（可从 git remote 自动推断） |

> *如果在 GitLink 仓库目录下执行，`--owner` 和 `--repo` 可自动推断。

## 注意事项

- 如果发布只有一个附件，不传 `--asset` 时会默认下载该附件。
- 如果发布有多个附件，必须传 `--asset` 指定 ID 或文件名，避免下载错文件。
- 默认不会覆盖已存在文件，需要覆盖时传 `--force`。

## References
- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
