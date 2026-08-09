---
name: gitlink-ignore
version: 1.0.0
description: "GitLink .gitignore 模板查询：列出/筛选平台内置忽略文件模板，用于创建仓库或补齐工程化配置。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli ignore --help"
---

# gitlink-ignore

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**

## Shortcuts

| Shortcut | 说明 | 需要认证 |
| --- | --- | --- |
| `ignore +list` | 查看 GitLink 内置 `.gitignore` 模板列表 | 否 |

## 使用示例

```bash
# 列出所有模板
gitlink-cli ignore +list

# 按名称筛选模板
gitlink-cli ignore +list --name Go
gitlink-cli ignore +list --name Python
```

## Agent 使用建议

- 创建仓库前，可用 `ignore +list --name <语言>` 检查平台是否支持对应 `.gitignore` 模板。
- 结合 `repo +create --ignore-id <id>`（如果当前版本支持）或 raw API 创建仓库时填入模板 ID。
- 该命令只读，不会修改任何仓库状态。

## OpenAPI

- `GET /ignores?name=<name>`
