# repo +tree

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

列出 GitLink 仓库根目录或指定目录下的文件和子目录。该命令封装 `sub_entries` API，适合项目结构检查、文档检查、科研复现性分析和 Agent 自动化报告。

## 命令

```bash
# 列出仓库根目录
gitlink-cli repo +tree --owner someone --repo myrepo

# 指定分支、标签或提交
gitlink-cli repo +tree --owner someone --repo myrepo --ref main

# 列出指定目录
gitlink-cli repo +tree --owner someone --repo myrepo --path src --ref main

# 输出为 JSON
gitlink-cli repo +tree --owner someone --repo myrepo --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--path, -p` | 否 | 要列出的目录路径，默认为仓库根目录 |
| `--ref, -r` | 否 | 分支、标签或提交引用，默认 `master` |
| `--owner` | 否 | 全局参数 - 仓库所有者，可从 git remote 自动解析 |
| `--repo` | 否 | 全局参数 - 仓库名称，可从 git remote 自动解析 |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |
| `--debug` | 否 | 启用调试输出 |

## 注意事项

- GitLink 仓库常见默认分支是 `master`，镜像仓库也可能使用 `main`。如果根目录返回不存在，请显式指定 `--ref main` 或从 `repo +info` 的 `default_branch` 字段确认。
- AI Agent 场景建议使用 `--format json`，便于读取 `data.entries` 中的文件名、路径、类型和 SHA。

## 参考

- [gitlink-repo](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
