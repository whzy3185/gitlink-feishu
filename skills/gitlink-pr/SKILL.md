---
name: gitlink-pr
version: 1.0.0
description: "Pull Request 管理：创建、查看、合并、关闭 PR，查看变更文件、Diff 和 patchset/version。当用户需要操作 GitLink PR 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-pr（Pull Request 操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `pr +list` | PR 列表 | 否（公开项目） |
| `pr +create` | 创建 PR | 是 |
| `pr +view` | PR 详情 | 否（公开项目） |
| `pr +merge` | 合并 PR | 是 |
| `pr +refuse` | 拒绝并关闭 PR | 是 |
| `pr +reopen` | 重开已关闭的 PR | 是 |
| `pr +files` | 变更文件列表 | 否 |
| `pr +diff` | 查看代码差异（含逐行 diff） | 否 |
| `pr +comment` | 给 PR 添加评论 | 是 |

## 使用示例

```bash
# 列出 PR
gitlink-cli pr +list --owner Gitlink --repo forgeplus --state open

# 搜索并排序 PR
gitlink-cli pr +list --owner Gitlink --repo forgeplus --state merged --keyword release --sort-by updated_at --sort-direction desc

# 创建 PR（源分支必须有实际代码变更）
gitlink-cli pr +create --title "feat: 新增搜索功能" --head feature/search --base master --body "实现了全文搜索"

# 查看 PR 详情（使用 pull_request_number，即网页 URL 中的序号）
gitlink-cli pr +view --id 3

# 合并 PR（支持 merge/rebase/squash）
gitlink-cli pr +merge --id 3
gitlink-cli pr +merge --id 3 --method squash

# 拒绝 PR（拒绝合并并关闭）
gitlink-cli pr +refuse --id 3

# 重开已关闭的 PR
gitlink-cli pr +reopen --id 3

# 查看变更文件（含 diff 内容）
gitlink-cli pr +files --id 3

# 查看代码差异（逐行 diff）
gitlink-cli pr +diff --id 3

# 查看某个文件的 diff
gitlink-cli pr +diff --id 3 --file src/main.go

# 仅查看 diff 统计摘要
gitlink-cli pr +diff --id 3 --stat

# 给 PR 添加评论
gitlink-cli pr +comment --id 3 --body "LGTM, ready to merge"
```

## 创建 PR 的完整流程

### 向他人仓库提 PR（Fork 流程，必须遵守）

```bash
# 1. Fork 目标仓库
gitlink-cli repo +fork --owner TargetOrg --repo target-repo

# 2. Clone 自己的 Fork
git clone https://www.gitlink.org.cn/MyUser/target-repo.git
cd target-repo
git remote add upstream https://www.gitlink.org.cn/TargetOrg/target-repo.git

# 3. 创建分支、修改、提交
git checkout -b fix/my-change
# ... 修改文件 ...
git add -A && git commit -m "fix: my change"

# 4. Push 到自己的 Fork（不是 upstream）
git push origin fix/my-change

# 5. 从 Fork 向主仓库提 PR
gitlink-cli pr +create --owner TargetOrg --repo target-repo \
  --head MyUser:fix/my-change --base master \
  --title "fix: my change"
```

> ⛔ **禁止直接往主仓库推分支再提 PR，即使有 admin 权限也不行。除非用户明确要求「直接 push」。**

### 在自己仓库直接提 PR（仅限用户明确要求时）

```bash
# 1. 创建分支
gitlink-cli branch +create --name feature-branch --from master

# 2. 在分支上创建/修改文件（content 必须 base64 编码）
gitlink-cli repo +create-file --body '{
  "filepath": "new-file.md",
  "content": "<base64编码的内容>",
  "branch": "feature-branch",
  "message": "add new file"
}'

# 3. 创建 PR
gitlink-cli pr +create --title "feat: 新功能" --head feature-branch --base master
```

## Raw API 补充

```bash
# 创建文件（content 必须 base64 编码）
gitlink-cli repo +create-file --body '{"filepath":"file.md","content":"<base64>","branch":"dev","message":"add file"}'

# 更新文件（需要先通过 sub_entries 获取文件 SHA）
gitlink-cli repo +files --query 'filepath=file.md&ref=dev'
# 从 entries.sha 获取 SHA，然后：
gitlink-cli repo +update-file --body '{"filepath":"file.md","content":"<base64>","sha":"<sha>","branch":"dev","message":"update file"}'

# 检查是否可合并
gitlink-cli pr +check-merge --body '{"head":"dev","base":"main"}'

# 创建 Review
gitlink-cli pr +review --body '{"content":"LGTM","status":"approved"}'

# 查看 Review 列表（支持 status 过滤）
gitlink-cli pr +reviews
gitlink-cli pr +reviews?status=approved

# 获取可用分支
gitlink-cli pr +branches

# 查看 PR patchset/version 列表（v1 API）
gitlink-cli pr +versions

# 查看指定 patchset/version diff（可通过 filepath 过滤文件）
gitlink-cli pr +versions/:version_id/diff
```

## 注意事项

- ⛔ **GitLink 的 PR 操作必须用 `gitlink-cli pr`，不能用 `gh pr`。** `gh` 是 GitHub CLI，无法操作 GitLink 平台。详见 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 的「工具使用边界」章节。
- GitLink 的默认分支通常是 `master`（非 `main`），创建 PR 时注意 `--base` 参数
- 合并 PR 前建议先用 `pr +view` 确认状态
- **PR 创建要求源分支与目标分支有实际代码差异**，否则返回"分支内容相同，无需创建合并请求"
- PR 查看/合并/关闭/重开需要使用 `pull_request_number`（即网页 URL `/pulls/N` 中的序号，从 `pr +list` 返回）
- `pr +merge` 默认使用 merge 方式，可通过 `--method` 指定 rebase 或 squash
- `pr +diff` 调用 `/pulls/:id/files` 端点，返回变更文件列表及每文件的 patch 内容
- `pr +files` 调用同一 `/pulls/:id/files` 端点，返回变更文件列表（不含 patch 渲染）
- `pr +diff` 支持 `--file` 过滤特定文件和 `--stat` 只看统计摘要
- `pr +versions` / `pr +version-diff` 使用 v1 API，`--id` 为网页 URL `/pulls/N` 中的 PR 序号，`--version-id` 为 patchset/version id
- 同一个 PR 分支继续 push 新 commit 会生成新的 patchset/version；正常根据 review 修改代码时，应优先在原 PR 分支继续 push，不要关闭 PR 重开
- `pr +list` 的 `--state` 参数（open/merged/closed）仅影响统计计数；需要查看所有 PR 时传 `--state all`
- PR 状态值：`pull_request_status` 0=open, 1=merged, 2=closed
- 关联已有 Issue 时，把 Issue 编号或 URL 写入 PR `--body`，或使用 `issue +comment` 留痕；不要用 Raw API 对 Issue 做不完整更新，否则可能清空 Issue 描述
