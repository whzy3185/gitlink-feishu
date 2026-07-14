---
name: gitlink-user
version: 1.0.0
description: "用户操作：查看当前用户、用户详情、贡献热力图、统计和项目趋势。当用户需要查看 GitLink 用户信息时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli user --help"
---

# gitlink-user（用户操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `user +me` | 当前登录用户 | 是 |
| `user +info` | 查看用户详情 | 否 |
| `user +heatmap` | 用户贡献热力图 | 省略 `--user` 时需要 |
| `user +statistics` | 用户聚合统计 | 省略 `--user` 时需要 |
| `user +stats` | `user +statistics` 的短别名 | 省略 `--user` 时需要 |
| `user +project-trends` | 用户项目趋势 | 省略 `--user` 时需要 |
| `user +trends` | `user +project-trends` 的短别名 | 省略 `--user` 时需要 |

## 使用示例

```bash
# 查看当前用户
gitlink-cli user +me

# 查看其他用户
gitlink-cli user +info --login zhangsan

# 用户贡献热力图
gitlink-cli user +heatmap --user zhangsan --year 2026

# 用户统计
gitlink-cli user +statistics --user zhangsan --start-time 1704067200 --end-time 1735689600

# 用户项目动态
gitlink-cli user +project-trends --user zhangsan
```

## 注意事项

- `user +heatmap`、`user +statistics`、`user +project-trends` 都是只读命令。
- 省略 `--user` 时会先调用 `user +me` 等价的 `/users/me` 解析当前登录用户，因此需要已登录。
- `user +stats` 和 `user +trends` 是为贡献者分析工作流保留的短别名。
