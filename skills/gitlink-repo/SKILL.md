---
name: gitlink-repo
version: 1.0.0
description: "仓库管理：创建、查看、Fork、删除仓库，查看 README、语言统计、贡献者、关注者，并执行关注/点赞等互动操作。当用户需要操作或分析 GitLink 仓库时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-repo（仓库操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `repo +list` | 仓库列表 | 否（公开项目） |
| `repo +info` | 仓库详情 | 否（公开项目） |
| `repo +readme` | README 内容 | 否（公开项目） |
| `repo +languages` | 仓库语言统计 | 否（公开项目） |
| `repo +contributors` | 仓库贡献者列表 | 否（公开项目） |
| `repo +contributor-stats` | 贡献者代码行统计 | 否（公开项目） |
| `repo +code-stats` | 仓库代码统计 | 否（公开项目） |
| `repo +watchers` | 关注者列表 | 否（公开项目） |
| `repo +stargazers` | 点赞者列表 | 否（公开项目） |
| `repo +follow` | 关注仓库 | 是 |
| `repo +unfollow` | 取消关注仓库 | 是 |
| `repo +like` | 点赞仓库 | 是 |
| `repo +unlike` | 取消点赞仓库 | 是 |
| `repo +create` | 创建仓库 | 是 |
| `repo +fork` | Fork 仓库 | 是 |
| `repo +delete` | 删除仓库 | 是 |

## 使用示例

```bash
# 查看仓库信息
gitlink-cli repo +info --owner Gitlink --repo forgeplus

# 在 git 仓库目录下自动解析
cd ~/my-project
gitlink-cli repo +info

# 列出用户的仓库
gitlink-cli repo +list --user zhangsan

# 查看语言占比和贡献者
gitlink-cli repo +languages --owner Gitlink --repo forgeplus
gitlink-cli repo +contributors --owner Gitlink --repo forgeplus

# 查看代码统计
gitlink-cli repo +contributor-stats --owner Gitlink --repo forgeplus --ref master --pass-year 1
gitlink-cli repo +code-stats --owner Gitlink --repo forgeplus --ref master

# 查看社区关注数据
gitlink-cli repo +watchers --owner Gitlink --repo forgeplus --start-at 1714521600 --end-at 1717200000
gitlink-cli repo +stargazers --owner Gitlink --repo forgeplus --start-at 1714521600 --end-at 1717200000

# 预览并执行仓库互动操作
gitlink-cli repo +follow --owner Gitlink --repo forgeplus --dry-run
gitlink-cli repo +follow --owner Gitlink --repo forgeplus
gitlink-cli repo +unfollow --owner Gitlink --repo forgeplus --project-id 123
gitlink-cli repo +like --owner Gitlink --repo forgeplus
gitlink-cli repo +unlike --owner Gitlink --repo forgeplus --project-id 123

# 创建仓库
gitlink-cli repo +create --name my-project --description "项目描述"

# Fork 仓库
gitlink-cli repo +fork --owner Gitlink --repo forgeplus

# 删除仓库（⚠️ 危险操作）
gitlink-cli repo +delete --owner myuser --repo old-project
```

## Raw API 补充

Shortcuts 未覆盖的仓库操作可用 Raw API：

```bash
# 获取提交列表
gitlink-cli api GET /:owner/:repo/commits --query 'page=1&limit=20'

# 获取标签列表
gitlink-cli api GET /:owner/:repo/tags

# 获取文件内容
gitlink-cli api GET /:owner/:repo/raw/main/README.md
```

## 注意事项

- `repo +delete` 是不可逆操作，执行前必须确认用户意图
- 创建仓库默认为公开，使用 `--private true` 创建私有仓库
