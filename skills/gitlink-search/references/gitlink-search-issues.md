# search +issues

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

搜索指定仓库中的 Issue（疑修）。支持关键词搜索、状态筛选、负责人/作者/里程碑/标签过滤、排序等。

## 命令

```bash
# 基本搜索
gitlink-cli search +issues --owner MyOrg --repo my-project --keyword "登录失败"

# 简写
gitlink-cli search +issues -k "bug" --owner MyOrg --repo my-project

# JSON 格式输出
gitlink-cli search +issues -k "bug" --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--keyword` / `-k` | 是 | 搜索关键词 |
| `--category` / `-c` | 否 | Issue 类型：`all`（全部）、`opened`（开启中）、`closed`（已关闭），默认 `all` |
| `--assignee` / `-a` | 否 | 按负责人用户 ID 筛选 |
| `--author` | 否 | 按创建人用户 ID 筛选 |
| `--milestone` / `-m` | 否 | 按里程碑 ID 筛选 |
| `--tag` / `-t` | 否 | 按标签 ID 筛选（多个 ID 用逗号分隔） |
| `--sort-by` | 否 | 排序字段：`updated_on`（默认）、`created_on`、`priority` |
| `--sort-dir` | 否 | 排序方向：`desc`（默认，倒序）、`asc`（正序） |
| `--page` / `-p` | 否 | 页码，默认 1 |
| `--limit` / `-l` | 否 | 每页数量，默认 20 |

## API

```
GET /api/v1/{owner}/{repo}/issues?keyword=xxx&category=opened&page=1&limit=20
```

## 注意事项

- 搜索 Issue 需要指定 `--owner` 和 `--repo`（或在一个已配置 git remote 的仓库目录中运行）
- `--tag` 参数接受逗号分隔的标签 ID（如 `--tag 1,2,3`），而非标签名称
- `--assignee` 和 `--author` 接受用户 ID（数字），不是用户名
- 返回数据包含 `total_count`、`opened_count`、`closed_count` 和 `issues` 数组
- 每个 Issue 包含 `project_issues_index`（网页 URL 中的序号）、`subject`、`status_name`、`author`、`assigners` 等字段

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-search SKILL.md](../SKILL.md) -- 搜索操作总览
