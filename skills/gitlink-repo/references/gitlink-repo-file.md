# repo +file

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

读取 GitLink 仓库中的任意文件内容。该命令基于 `sub_entries` API 的文件模式封装，适合查看 `go.mod`、`.gitignore`、配置文件、脚本、许可证文本和示例数据文件。

## 命令

```bash
# 读取默认分支上的文件
gitlink-cli repo +file --owner someone --repo myrepo --path go.mod

# 指定分支、标签或提交
gitlink-cli repo +file --owner someone --repo myrepo --path .gitignore --ref main

# 只输出文件内容
gitlink-cli repo +file --owner someone --repo myrepo --path README.md --content-only

# Agent 场景建议使用 JSON
gitlink-cli repo +file --owner someone --repo myrepo --path package.json --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--path, -p` | 是 | 仓库内文件路径，如 `go.mod`、`docs/guide.md` |
| `--ref, -r` | 否 | 分支、标签或提交引用，默认 `master` |
| `--content-only` | 否 | 只输出文件内容，不附带路径、SHA、大小等元数据 |
| `--owner` | 否 | 全局参数，仓库所有者，可从 git remote 自动解析 |
| `--repo` | 否 | 全局参数，仓库名称，可从 git remote 自动解析 |
| `--format` | 否 | 输出格式：`json` / `table` / `yaml` |

## 注意事项

- `repo +file` 只能读取文件；如果传入目录路径，命令会提示改用 `repo +tree`。
- GitLink 仓库常见默认分支是 `master`。如果仓库使用 `main`，请显式传入 `--ref main`。
- Agent 或脚本场景建议使用 `--format json`，方便读取 `data.content`。

## 参考

- [gitlink-repo](../SKILL.md)
- [gitlink-repo-tree](./gitlink-repo-tree.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
