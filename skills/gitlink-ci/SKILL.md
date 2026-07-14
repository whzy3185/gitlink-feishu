---
name: gitlink-ci
version: 1.0.0
description: "CI/CD 操作：查看构建列表、构建日志、重启/停止构建。当用户需要操作 GitLink CI 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli ci --help"
---

# gitlink-ci（CI/CD 操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `ci +builds` | 构建列表 | 是 |
| `ci +logs` | 构建日志 | 是 |
| `ci +restart` | 重启构建 | 是 |
| `ci +stop` | 停止构建 | 是 |
| `ci +enable` | 启用 CI | 是 |
| `ci +disable` | 停用 CI | 是 |
| `ci +authorize` | CI 授权状态 | 是 |

## 使用示例

```bash
# 查看构建列表
gitlink-cli ci +builds --owner myuser --repo myrepo

# 查看构建日志
gitlink-cli ci +logs --build 42 --stage 1 --step 1

# 重启构建
gitlink-cli ci +restart --build 42

# 停止构建
gitlink-cli ci +stop --build 42

# 启用 CI（需先配置 .gitlink-ci.yml 流水线文件）
gitlink-cli ci +enable --owner myuser --repo myrepo

# 停用 CI
gitlink-cli ci +disable --owner myuser --repo myrepo

# 查看 CI 授权状态
gitlink-cli ci +authorize --owner myuser --repo myrepo
```

## 注意事项

- `ci +enable` 需要仓库已配置 `.gitlink-ci.yml` 流水线文件，否则返回 -1
- 构建操作需要仓库已启用 CI，可通过 `ci +authorize` 查看状态
