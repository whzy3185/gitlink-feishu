# pr +version-diff

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看 Pull Request 指定 patchset/version 的 diff。可选 `--file` 只查看某个文件的 diff，适合 Agent 或维护者针对 review 反馈定位变更。

## 命令

```bash
# 查看指定 patchset/version diff
gitlink-cli pr +version-diff --id 3 --version-id 16040

# 简写
gitlink-cli pr +version-diff -i 3 -v 16040

# 只查看某个文件的 diff
gitlink-cli pr +version-diff -i 3 -v 16040 --file shortcuts/pr/pr.go

# 指定仓库并输出 JSON
gitlink-cli pr +version-diff --owner Gitlink --repo forgeplus -i 3 -v 16040 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（网页 URL `/pulls/N` 中的 `N`，即 `pull_request_number`） |
| `--version-id` / `-v` | 是 | patchset/version id，可从 `pr +versions` 返回结果获取 |
| `--file` / `-f` | 否 | 按文件路径过滤 diff，对应 API 查询参数 `filepath` |

## API

```
GET /v1/{owner}/{repo}/pulls/{number}/versions/{version_id}/diff
```

当传入 `--file` 时，会附加查询参数：

```
filepath=<path>
```

## 典型流程

```bash
# 1. 列出 PR 的 patchset/version
gitlink-cli pr +versions -i 3 --format json

# 2. 选择需要查看的 version id
# 3. 查看该 version 的 diff
gitlink-cli pr +version-diff -i 3 -v 16040

# 4. 如只关心某个文件，使用 --file 过滤
gitlink-cli pr +version-diff -i 3 -v 16040 --file shortcuts/pr/pr.go
```

## 注意事项

- `--version-id` 是 version id，不是 commit SHA。
- 这是只读查询命令，不会修改 PR。
- 如果要审查最新一轮变更，先用 `pr +versions` 获取最新 version，再调用本命令。

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
- [pr +versions](gitlink-pr-versions.md) -- 查看 PR patchset/version 列表
