---
name: gitlink-file
version: 1.0.0
description: "仓库文件操作：浏览目录、查看文件、创建、更新、删除文件。当用户需要在 GitLink 仓库中操作文件时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli file --help"
---

# gitlink-file（文件操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**

## Shortcuts

| Shortcut | 说明 |
|----------|------|
| `file +browse` | 浏览目录树或文件详情 |
| `file +get` | 获取文件内容 |
| `file +create` | 创建文件 |
| `file +update` | 更新文件 |
| `file +delete` | 删除文件 |

## 使用示例

```bash
# 浏览目录
gitlink-cli file +browse --owner Gitlink --repo forgeplus --path src/

# 获取文件内容
gitlink-cli file +get --owner Gitlink --repo forgeplus --path README.md

# 创建文件（内容自动 base64 编码）
gitlink-cli file +create --owner myuser --repo myrepo --path docs/guide.md --content "# Guide" --message "Add guide"

# 更新文件（SHA 自动获取）
gitlink-cli file +update --owner myuser --repo myrepo --path docs/guide.md --content "# Updated Guide"

# 删除文件
gitlink-cli file +delete --owner myuser --repo myrepo --path old-file.txt
```

## API 注意事项

- **内容自动 base64 编码**：`file +create` 和 `file +update` 会自动将 content 编码为 base64
- **SHA 自动获取**：`file +update` 和 `file +delete` 会自动获取文件 SHA，无需手动提供。也可通过 `--sha` 手动指定
- 文件路径使用 `--path` 参数，API 自动处理 base64 路径编码
- `file +browse` 和 `file +get` 使用非 v1 路径：`/{owner}/{repo}/sub_entries`
- 写操作使用：`/{owner}/{repo}/create_file`、`/{owner}/{repo}/update_file`、`/{owner}/{repo}/delete_file`
