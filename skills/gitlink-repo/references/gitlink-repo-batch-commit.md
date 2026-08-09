# repo +batch-commit

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

在一次提交中创建、更新或删除多个文件。该命令封装 GitLink 的 batch contents API，适合 Agent 生成文档、配置或示例文件后集中提交到分支。

## 命令

```bash
# 预览多文件提交
gitlink-cli repo +batch-commit --owner someone --repo myrepo \
  --branch master \
  --message "docs: update guide" \
  --files 'update:README.md:# Updated;create:docs/demo.md:# Demo' \
  --dry-run

# 确认执行多文件提交
gitlink-cli repo +batch-commit --owner someone --repo myrepo \
  --branch master \
  --message "docs: update guide" \
  --files 'update:README.md:# Updated;delete:old.md' \
  --yes

# 创建新分支并提交
gitlink-cli repo +batch-commit --owner someone --repo myrepo \
  --branch master \
  --new-branch docs/update-guide \
  --message "docs: update guide" \
  --files 'create:docs/guide.md:# Guide' \
  --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--branch, -b` | 是 | 目标分支 |
| `--message, -m` | 是 | 提交信息 |
| `--files, -f` | 是 | 文件操作列表，格式见下文 |
| `--new-branch` | 否 | 创建并提交到新分支 |
| `--encoding` | 否 | 内容编码，`text` 或 `base64`，默认 `text` |
| `--author-name` | 否 | 作者名称 |
| `--author-email` | 否 | 作者邮箱 |
| `--committer-name` | 否 | 提交者名称 |
| `--committer-email` | 否 | 提交者邮箱 |
| `--dry-run` | 否 | 预览请求，不修改远端 |
| `--yes` | 否 | 确认执行远端写入 |

## `--files` 格式

```text
action:path[:content][;action:path[:content]...]
```

支持的 `action`：

- `create`：创建文件，必须提供 content。
- `update`：更新文件，必须提供 content。
- `delete`：删除文件，不需要 content。

## API

```text
POST /v1/{owner}/{repo}/contents/batch
```

## 安全规则

- 真实写入前必须先运行 `--dry-run`。
- 未传 `--yes` 时命令不会调用远端写入 API。
- Agent 必须向用户展示 dry-run 中的 `payload`，确认文件路径、操作类型、分支和提交信息无误后再执行。

## 参考

- [gitlink-repo](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
