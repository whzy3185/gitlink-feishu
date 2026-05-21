# pr +reviews

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看 Pull Request 的审查记录，可按审查状态过滤。适合在合并前检查是否已有 approve、reject 或普通审查意见。

## 命令

```bash
# 查看 PR 审查记录
gitlink-cli pr +reviews --id 3

# 简写
gitlink-cli pr +reviews -i 3

# 按状态筛选
gitlink-cli pr +reviews -i 3 --status approved
gitlink-cli pr +reviews -i 3 --status rejected
gitlink-cli pr +reviews -i 3 --status common

# 指定仓库并输出 JSON
gitlink-cli pr +reviews --owner Gitlink --repo forgeplus -i 3 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（网页 URL `/pulls/N` 中的 `N`，即 `pull_request_number`） |
| `--status` / `-s` | 否 | 审查状态：`common`、`approved`、`rejected` |

## API

```
GET /v1/{owner}/{repo}/pulls/{number}/reviews
```

当传入 `--status` 时，会附加查询参数：

```
status=<common|approved|rejected>
```

## 典型流程

1. `pr +view --id <number>` 确认 PR 状态。
2. `pr +files --id <number>` 查看变更文件。
3. `pr +reviews --id <number>` 查看现有审查记录。
4. 根据审查结果决定是否使用 `pr +review` 添加普通评论、approve 或 reject。

## 注意事项

- `--id` 不是数据库 ID，而是网页 URL `/pulls/N` 中的 PR 序号。
- 这是只读查询命令，不会修改 PR。
- `approved` 表示通过，`rejected` 表示拒绝/请求修改，`common` 表示普通审查评论。

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
- [pr +review](gitlink-pr-review.md) -- 创建 PR 审查
