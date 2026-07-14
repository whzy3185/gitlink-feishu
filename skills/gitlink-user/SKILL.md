---
name: gitlink-user
version: 1.0.0
description: "用户操作：查看当前用户、用户详情。当用户需要查看 GitLink 用户信息时触发。"
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
| `user +headmaps` | 贡献热力图 | 否 |
| `user +stats-activity` | 活跃度统计 | 否 |
| `user +stats-develop` | 开发能力统计 | 否 |
| `user +stats-role` | 角色定位统计 | 否 |
| `user +stats-major` | 专业定位统计 | 否 |
| `user +trends` | 项目动态趋势 | 否 |

## 使用示例

```bash
# 查看当前用户
gitlink-cli user +me

# 查看其他用户
gitlink-cli user +info --login zhangsan

# 查看贡献热力图
gitlink-cli user +headmaps --login zhangsan

# 查看统计信息
gitlink-cli user +stats-activity --login zhangsan
gitlink-cli user +stats-develop --login zhangsan
gitlink-cli user +stats-role --login zhangsan
gitlink-cli user +stats-major --login zhangsan

# 查看项目动态
gitlink-cli user +trends --login zhangsan --limit 20
```

## 注意事项

- 查看其他用户信息需要提供 `--login` 参数
