---
name: gitlink-meta
version: 1.0.0
description: "公开元数据查询：查询 GitLink 许可证模板和 .gitignore 模板。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli meta --help"
---

# gitlink-meta（公开元数据查询）

**CRITICAL — 开始前建议阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、全局参数和输出格式说明。**

`meta` 命令只读取公开模板数据，不修改远端资源，适合创建仓库前查询可用的许可证和 `.gitignore` 模板。

## Shortcuts

| Shortcut | 说明 | 操作类型 |
|----------|------|----------|
| `meta +licenses` | 查询许可证模板列表 | Read |
| `meta +ignores` | 查询 `.gitignore` 模板列表 | Read |

## 参数参考

| 命令 | 参数 | 必填 | 说明 |
|------|------|------|------|
| `meta +licenses` | `--name, -n` | 否 | 按许可证名称过滤，例如 `MIT` |
| `meta +ignores` | `--name, -n` | 否 | 按模板名称过滤，例如 `Go` |
| 两者 | `--format` | 否 | 输出格式：`json`/`table`/`yaml` |

## 使用示例

```bash
# 查询许可证模板
gitlink-cli meta +licenses --name MIT --format json

# 查询 .gitignore 模板
gitlink-cli meta +ignores --name Go --format json

# 列出全部模板
gitlink-cli meta +licenses
gitlink-cli meta +ignores
```

## Agent 工作流建议

1. 创建仓库前先用 `meta +licenses` / `meta +ignores` 查询模板名称或 ID。
2. 将查询结果与用户需求对齐，例如开源许可证选择、语言模板选择。
3. 后续再调用仓库创建或更新命令，减少用户手动查网页的成本。

## References

- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、输出格式
- GitLink OpenAPI：`GET /api/licenses.json`、`GET /api/ignores.json`
