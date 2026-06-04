---
name: gitlink-catalog
description: "查询 GitLink 平台模板目录，包括仓库创建时可选的许可证模板和 .gitignore 模板。"
---

# gitlink-catalog

当用户需要为新仓库选择许可证或 `.gitignore` 模板时使用本 Skill。命令只读取 GitLink 平台目录，不需要 owner/repo 上下文，适合仓库初始化、脚手架生成和自动化创建仓库前的候选项校验。

## Commands

| 命令 | 用途 |
|------|------|
| `catalog +licenses` | 列出许可证模板 |
| `catalog +ignores` | 列出 `.gitignore` 模板 |

## Examples

```bash
# 列出全部许可证模板
gitlink-cli catalog +licenses

# 搜索 MIT 相关许可证模板
gitlink-cli catalog +licenses --name MIT

# 搜索 Go 项目的 .gitignore 模板
gitlink-cli catalog +ignores --name Go
```

## Notes

- `--name` 由 GitLink API 处理，适合在候选项较多时缩小结果范围。
- 输出默认是 JSON，可配合 `--format json` 给 Agent 或脚本继续处理。
