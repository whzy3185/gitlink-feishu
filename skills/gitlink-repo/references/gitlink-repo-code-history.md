# repo code history shortcuts

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

用于搜索仓库文件、查看提交历史、查看单个提交的变更文件和 diff。适合代码审查、科研复现性分析、Release Notes 生成和项目健康度报告。

## 命令

```bash
# 搜索仓库文件
gitlink-cli repo +files --owner someone --repo myrepo --search README --ref master

# 查看提交历史
gitlink-cli repo +commits --owner someone --repo myrepo --ref master --page 1 --limit 20

# 查看单个提交变更文件
gitlink-cli repo +commit-files --owner someone --repo myrepo --sha <commit_sha>

# 按文件路径筛选单个提交变更
gitlink-cli repo +commit-files --owner someone --repo myrepo --sha <commit_sha> --file src/main.go

# 查看单个提交 diff
gitlink-cli repo +commit-diff --owner someone --repo myrepo --sha <commit_sha>
```

## 参数

| 命令 | 参数 | 必填 | 说明 |
|------|------|------|------|
| `repo +files` | `--search, -s` | 否 | 文件名或路径关键词 |
| `repo +files` | `--ref, -r` | 否 | 分支、标签或 Commit SHA |
| `repo +commits` | `--ref, -r` | 否 | 分支、标签或 Commit SHA，映射到 API `sha` |
| `repo +commits` | `--page, -p` | 否 | 页码，默认 `1` |
| `repo +commits` | `--limit, -l` | 否 | 每页数量，默认 `20` |
| `repo +commit-files` | `--sha, -s` | 是 | Commit SHA |
| `repo +commit-files` | `--file, -f` | 否 | 只查看指定文件路径的变更 |
| `repo +commit-diff` | `--sha, -s` | 是 | Commit SHA |

## API

```text
GET /{owner}/{repo}/files
GET /v1/{owner}/{repo}/commits
GET /v1/{owner}/{repo}/commits/{sha}/files
GET /v1/{owner}/{repo}/commits/{sha}/diff
```

## 注意事项

- AI Agent 场景建议始终使用 `--format json`。
- `repo +commit-files --file` 会使用 `filepath` 查询参数；不传 `--file` 时使用分页参数。
- `repo +commits --ref` 支持分支、标签或 Commit SHA。

## 参考

- [gitlink-repo](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
