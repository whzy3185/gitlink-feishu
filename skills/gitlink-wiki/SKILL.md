---
name: gitlink-wiki
version: 1.0.0
description: "Wiki 操作：查看、创建、更新、删除 Wiki 页面。当用户需要管理 GitLink 仓库 Wiki 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
---

# gitlink-wiki（Wiki 操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `wiki +list` | 列出 Wiki 页面 | 是 |
| `wiki +view` | 查看 Wiki 页面内容 | 是 |
| `wiki +create` | 创建 Wiki 页面 | 是 |
| `wiki +update` | 更新 Wiki 页面 | 是 |
| `wiki +delete` | 删除 Wiki 页面 | 是 |

## 使用示例

```bash
# 列出所有 Wiki 页面
gitlink-cli wiki +list --owner myuser --repo myrepo

# 查看 Wiki 页面
gitlink-cli wiki +view --name Home

# 创建 Wiki 页面
gitlink-cli wiki +create --name Guide --content "使用指南内容"

# 更新 Wiki 页面（带提交信息）
gitlink-cli wiki +update --name Guide --content "更新后的内容" --message "更新使用指南"

# 删除 Wiki 页面
gitlink-cli wiki +delete --name OldPage
```

## 注意事项

- Wiki 命令会自动从仓库信息中获取 `projectId`，无需手动指定
- `--content` 参数的内容会自动进行 base64 编码
- 在 git 仓库目录下执行时，`--owner` 和 `--repo` 会自动解析
