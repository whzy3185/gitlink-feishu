# repo tag shortcuts

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

用于查看仓库标签列表、标签详情，以及在明确确认后删除远端标签。

## 命令

```bash
# 分页列出标签
gitlink-cli repo +tags --owner someone --repo myrepo --page 1 --limit 20

# 按名称搜索标签，仅返回名称
gitlink-cli repo +tags --owner someone --repo myrepo --name v1 --only-name true

# 查看标签详情
gitlink-cli repo +tag --owner someone --repo myrepo --name v1.0.0

# 预览删除标签
gitlink-cli repo +delete-tag --owner someone --repo myrepo --name v1.0.0 --dry-run

# 确认删除标签
gitlink-cli repo +delete-tag --owner someone --repo myrepo --name v1.0.0 --yes
```

## 参数

| 命令 | 参数 | 必填 | 说明 |
|------|------|------|------|
| `repo +tags` | `--page, -p` | 否 | 页码，默认 `1` |
| `repo +tags` | `--limit, -l` | 否 | 每页数量，默认 `20` |
| `repo +tags` | `--name, -n` | 否 | 标签搜索关键词 |
| `repo +tags` | `--only-name` | 否 | 只返回标签名称，通常与 `--name` 搭配 |
| `repo +tag` | `--name, -n` | 是 | 标签名称 |
| `repo +delete-tag` | `--name, -n` | 是 | 标签名称 |
| `repo +delete-tag` | `--dry-run` | 否 | 预览删除请求，不修改远端 |
| `repo +delete-tag` | `--yes` | 否 | 确认执行删除 |

## API

```text
GET /v1/{owner}/{repo}/tags
GET /{owner}/{repo}/tags
GET /v1/{owner}/{repo}/tags/{name}
DELETE /v1/{owner}/{repo}/tags/{tag}
```

## 注意事项

- `repo +delete-tag` 是破坏性操作，Agent 必须先执行 `--dry-run`，展示将删除的标签，再等待用户明确确认。
- 不传 `--name`/`--only-name` 时，`repo +tags` 使用 v1 分页标签接口；传搜索条件时使用支持名称筛选的标签接口。

## 参考

- [gitlink-repo](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
