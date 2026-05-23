# pr +versions

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看 Pull Request 的 patchset/version 列表。GitLink 会在同一个 PR 分支继续 push 新 commit 时生成新的 version，适合 review 过程中追踪每轮变更。

## 命令

```bash
# 查看 PR patchset/version 列表
gitlink-cli pr +versions --id 3

# 简写
gitlink-cli pr +versions -i 3

# 指定仓库并输出 JSON
gitlink-cli pr +versions --owner Gitlink --repo forgeplus -i 3 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（网页 URL `/pulls/N` 中的 `N`，即 `pull_request_number`） |

## API

```
GET /v1/{owner}/{repo}/pulls/{number}/versions
```

## 典型流程

1. `pr +view --id <number>` 确认 PR 状态。
2. `pr +versions --id <number>` 查看每轮 patchset/version。
3. 选取需要审查的 `version.id`。
4. 使用 `pr +version-diff --id <number> --version-id <version_id>` 查看指定版本 diff。

## 注意事项

- `--id` 不是数据库 ID，而是网页 URL `/pulls/N` 中的 PR 序号。
- patchset/version 是只读查询命令，不会修改线上数据。
- 根据 review 修改代码时，优先向原 PR 分支继续 push，GitLink 会生成新的 patchset/version；不要因为描述或代码更新而关闭 PR 重开。

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
- [pr +version-diff](gitlink-pr-version-diff.md) -- 查看指定 patchset/version diff
