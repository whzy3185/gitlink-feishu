# pr +review

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

创建 Pull Request 审查，可提交普通审查意见、通过（approve）或拒绝/请求修改（reject）。写入前建议先使用 `--dry-run` 预览请求内容。

## 命令

```bash
# 预览普通审查评论
gitlink-cli pr +review --id 3 --status common --content "整体看起来可以" --dry-run

# 提交普通审查评论
gitlink-cli pr +review --id 3 --status common --content "整体看起来可以"

# 通过 PR
gitlink-cli pr +review --id 3 --status approved --content "LGTM"

# 请求修改 / 拒绝通过
gitlink-cli pr +review --id 3 --status rejected --content "测试未通过，请修复后再合并"

# 绑定到指定 commit
gitlink-cli pr +review --id 3 --status approved --content "LGTM" --commit <commit_sha>
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（网页 URL `/pulls/N` 中的 `N`，即 `pull_request_number`） |
| `--status` / `-s` | 否 | 审查状态：`common`、`approved`、`rejected`，默认 `common` |
| `--content` / `-c` | 是 | 审查内容 |
| `--commit` / `-m` | 否 | 绑定审查的 commit SHA，对应 API 字段 `commit_id` |
| `--dry-run` | 否 | 只预览请求，不创建审查 |

## API

```
POST /v1/{owner}/{repo}/pulls/{number}/reviews
```

请求体：

```json
{
  "content": "LGTM",
  "status": "approved",
  "commit_id": "<commit_sha>"
}
```

其中 `commit_id` 仅在传入 `--commit` 时发送。

## 安全流程

对写操作建议遵循：

1. 先用 `pr +view` / `pr +files` / `pr +reviews` 获取上下文。
2. 准备审查内容和状态。
3. 执行 `pr +review --dry-run` 预览。
4. 用户确认后去掉 `--dry-run` 执行真实写入。

## 注意事项

- `--status approved` 表示通过 PR；`--status rejected` 表示拒绝/请求修改；`--status common` 表示普通审查意见。
- `--content` 必填，避免产生没有上下文的 approve/reject。
- `--dry-run` 不会请求 GitLink API，适合 Agent 在执行写操作前展示计划。

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
- [pr +reviews](gitlink-pr-reviews.md) -- 查看 PR 审查记录
