---
name: gitlink-invite
version: 1.0.0
description: "项目邀请管理：生成邀请链接、查看链接信息、接受邀请、加入项目、退出项目。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli invite --help"
---

# gitlink-invite（项目邀请管理）

当用户需要管理 GitLink 项目邀请、通过邀请链接加入项目或退出项目时使用本 Skill。

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有 Shortcuts 在执行写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `invite +generate` | 生成或获取项目邀请链接 | 是（项目管理员） |
| `invite +show` | 查看邀请链接详细信息 | 否 |
| `invite +accept` | 通过邀请链接接受邀请 | 是 |
| `invite +join` | 申请加入项目 | 是 |
| `invite +quit` | 退出项目 | 是 |

## 使用示例

```bash
# 生成邀请链接（管理员权限）
# role: manager/developer/reporter
# apply: true（需要审核）/false（自动加入）
gitlink-cli invite +generate --owner Gitlink --repo forgeplus --role developer --apply false

# 查看邀请链接信息
gitlink-cli invite +show --sign <invite_sign>

# 通过邀请链接接受邀请
gitlink-cli invite +accept --owner Gitlink --repo forgeplus --sign <invite_sign>

# 申请加入项目（需要项目邀请码）
gitlink-cli invite +join --code <invite_code> --role developer

# 退出项目
gitlink-cli invite +quit --owner Gitlink --repo forgeplus
```

## 参数说明

### invite +generate

| 参数 | 说明 | 必填 | 默认值 |
|------|------|------|--------|
| `--owner` | 仓库所有者 | 是 | - |
| `--repo` | 仓库名称 | 是 | - |
| `--role` | 邀请角色：manager/developer/reporter | 否 | developer |
| `--apply` | 是否需要审核：true/false | 否 | false |

### invite +show

| 参数 | 说明 | 必填 |
|------|------|------|
| `--sign` | 邀请链接签名 | 是 |

### invite +accept

| 参数 | 说明 | 必填 |
|------|------|------|
| `--owner` | 仓库所有者 | 是 |
| `--repo` | 仓库名称 | 是 |
| `--sign` | 邀请链接签名 | 是 |

### invite +join

| 参数 | 说明 | 必填 |
|------|------|------|
| `--code` | 项目邀请码 | 是 |
| `--role` | 申请角色：manager/developer/reporter | 是 |

### invite +quit

| 参数 | 说明 | 必填 |
|------|------|------|
| `--owner` | 仓库所有者 | 是 |
| `--repo` | 仓库名称 | 是 |

## 安全规则

- 执行 `invite +quit` 前，确认用户了解退出项目的影响
- 避免在公开日志中暴露邀请链接的完整 `sign`
- `invite +generate` 需要项目管理员权限
- `invite +join` 可能需要项目管理员审核

## 工作流示例

### 场景：邀请新成员加入项目

```bash
# 1. 项目管理员生成邀请链接
gitlink-cli invite +generate --owner myorg --repo myproject --role developer --apply false

# 2. 将返回的 sign 分享给新成员
# 输出示例：{"sign": "abc123xyz", "role": "developer", ...}

# 3. 新成员查看邀请链接信息
gitlink-cli invite +show --sign abc123xyz

# 4. 新成员接受邀请
gitlink-cli invite +accept --owner myorg --repo myproject --sign abc123xyz
```

### 场景：申请加入项目

```bash
# 1. 获取项目邀请码（从项目设置页面）
# 2. 申请加入
gitlink-cli invite +join --code PROJECT123 --role developer

# 3. 等待项目管理员审核
```

### 场景：退出项目

```bash
# 退出前确认
gitlink-cli invite +quit --owner myorg --repo myproject
```
