---
name: gitlink-file
version: 1.0.0
description: "文件内容操作：无需克隆即可查看、搜索、创建、更新、删除 GitLink 仓库文件。当用户需要读取或修改仓库中的单个文件（如 README、配置文件）而不想克隆仓库时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli file --help"
---

# gitlink-file（文件内容操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — `file +create` / `+update` / `+delete` 会直接产生提交，执行前务必先确认用户意图，并优先使用 `--new-branch` 提交到新分支。**

> 目录列表和 README 查看请使用 `repo +tree` 和 `repo +readme`（见 [`../gitlink-repo/SKILL.md`](../gitlink-repo/SKILL.md)）。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `file +view` | 查看文件内容；`--raw` 仅输出解码后的正文 | 否（公开项目） |
| `file +search` | 按文件名搜索仓库文件 | 否（公开项目） |
| `file +create` | 创建文件并提交到指定分支 | 是 |
| `file +update` | 更新文件并提交到指定分支 | 是 |
| `file +delete` | 删除文件并提交 | 是 |

## 示例

```bash
# 查看文件（--raw 直接输出正文，可管道/重定向）
gitlink-cli file +view --owner Gitlink --repo forgeplus --path README.md
gitlink-cli file +view --owner Gitlink --repo forgeplus --path README.md --raw > README.md

# 指定分支/标签/提交
gitlink-cli file +view --owner Gitlink --repo forgeplus --path app/models/user.rb --ref develop

# 按文件名搜索
gitlink-cli file +search --owner Gitlink --repo forgeplus --keyword controller

# 创建文件（内容内联或来自本地文件，二选一）
gitlink-cli file +create --owner me --repo proj --path docs/note.md -c "# 笔记" -b master -m "add note"
gitlink-cli file +create --owner me --repo proj --path docs/note.md --content-file note.md -b master

# 更新文件并提交到从 master 新建的分支（推荐，便于走 PR 流程）
gitlink-cli file +update --owner me --repo proj --path docs/note.md -c "..." -b master --new-branch feature/docs

# 删除文件
gitlink-cli file +delete --owner me --repo proj --path docs/note.md -b master -m "remove note"
```

## Notes

- `file +view` 调用 `/api/{owner}/{repo}/sub_entries`；`+search` 调用 `/api/{owner}/{repo}/files`。
- 写操作调用 `/api/v1/{owner}/{repo}/contents/batch`，内容使用 `text` 编码传输（生产环境不接受 base64）。
- `--message` 缺省为 `<action> <path>`；`--new-branch` 从 `--branch` 新建分支并提交到新分支。
- 修改公共仓库文件时，优先 `--new-branch` + `pr +create`（见 [`../gitlink-pr/SKILL.md`](../gitlink-pr/SKILL.md)），避免直接推主分支。
