---
name: gitlink-repo
version: 1.0.0
description: "仓库管理：创建、查看、Fork、删除仓库，管理设置、Topics、导航和迁移。当用户需要操作 GitLink 仓库时触发。"
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
| `repo +detail` | 仓库详情元数据 | 否（公开项目） |
| `repo +simple` | 仓库简版元数据 | 否（公开项目） |
| `repo +settings` | 仓库设置元数据 | 是 |
| `repo +units` | 仓库导航配置 | 是 |
| `repo +units-update` | 更新仓库导航配置 | 是 |
| `repo +topics` | 项目 Topic 标签列表 | 否 |
| `repo +topic-add` | 添加项目 Topic 标签 | 是 |
| `repo +topic-delete` | 删除项目 Topic 标签 | 是 |
| `repo +transfer-orgs` | 可迁移组织列表 | 是 |
| `repo +transfer` | 发起仓库迁移 | 是 |
| `repo +transfer-cancel` | 取消仓库迁移 | 是 |
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

# 创建仓库
gitlink-cli repo +create --name my-project --description "项目描述"

# Fork 仓库
gitlink-cli repo +fork --owner Gitlink --repo forgeplus

# 查看仓库设置、导航和主题
gitlink-cli repo +detail --owner Gitlink --repo forgeplus
gitlink-cli repo +settings --owner Gitlink --repo forgeplus
gitlink-cli repo +units --owner Gitlink --repo forgeplus
gitlink-cli repo +topics --keyword go

# 更新导航和 Topics，写入前先 dry-run
gitlink-cli repo +units-update --owner Gitlink --repo forgeplus --units code,issues,pulls,wiki --dry-run
gitlink-cli repo +topic-add --project-id 17 --name go --dry-run
gitlink-cli repo +topic-delete --project-id 17 --id 8 --dry-run

# 仓库迁移，写入前先 dry-run
gitlink-cli repo +transfer-orgs --owner Gitlink --repo forgeplus
gitlink-cli repo +transfer --owner Gitlink --repo forgeplus --owner-name target-org --dry-run
gitlink-cli repo +transfer-cancel --owner Gitlink --repo forgeplus --dry-run

# 删除仓库（⚠️ 危险操作）
gitlink-cli repo +delete --owner myuser --repo old-project
```

## Raw API 补充

Shortcuts 未覆盖的仓库操作可用 Raw API：

```bash
# 获取 README
gitlink-cli api GET /:owner/:repo/readme

# 获取贡献者列表
gitlink-cli api GET /:owner/:repo/contributors

# 获取语言统计
gitlink-cli api GET /:owner/:repo/languages

# 获取提交列表
gitlink-cli api GET /:owner/:repo/commits --query 'page=1&limit=20'

# 获取标签列表
gitlink-cli api GET /:owner/:repo/tags

# 获取文件内容
gitlink-cli api GET /:owner/:repo/raw/main/README.md
```

## 注意事项

- `repo +delete` 是不可逆操作，执行前必须确认用户意图
- `repo +units-update`、`repo +topic-add`、`repo +topic-delete`、`repo +transfer`、`repo +transfer-cancel` 支持 `--dry-run`
- `repo +topic-add` / `repo +topic-delete` 需要 `project_id`，可先用 `repo +detail` 获取
- 创建仓库默认为公开，使用 `--private true` 创建私有仓库
