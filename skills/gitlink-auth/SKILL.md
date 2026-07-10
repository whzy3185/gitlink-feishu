---
name: gitlink-auth
version: 1.0.0
description: "认证管理：登录、查看登录状态、管理 Token、退出登录。当用户首次使用 gitlink-cli、遇到 401 认证错误、Token 过期、需要登录或退出时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli auth --help"
---

# gitlink-auth（认证操作）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证原理、Token 存储位置、认证错误处理（401/403）和全局参数。**
**CRITICAL — 认证 Token 属于敏感信息，禁止明文输出到终端或日志。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 本 Skill 聚焦于 `gitlink-cli auth` 命令的具体操作；认证的全局规则、错误处理、安全约定见 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## Shortcuts

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `auth login` | 交互式登录（用户名 + 密码） | 否（登录本身） |
| `auth login --token` | 使用已有 Token 登录 | 否 |
| `auth status` | 查看当前登录状态与 Token 有效期 | 否 |
| `auth logout` | 退出登录并清除本地凭证 | 否 |

## 使用示例

```bash
# 方式 1：交互式登录（推荐，按提示输入用户名和密码）
gitlink-cli auth login

# 方式 2：粘贴已有 Token 登录（适合已有 GitLink 个人访问令牌的场景）
gitlink-cli auth login --token

# 查看登录状态（确认身份、Token 剩余有效期、存储位置）
gitlink-cli auth status

# 退出登录，清除本地凭证
gitlink-cli auth logout
```

## 工作流

### 工作流 1：首次使用认证

1. 运行 `gitlink-cli auth login` 完成交互式登录
2. 运行 `gitlink-cli auth status` 确认登录成功
3. 验证可调用受保护接口：`gitlink-cli api GET /users/me --format json`

### 工作流 2：Token 过期恢复

遇到 `401 请登录后再操作` 错误时：

1. `gitlink-cli auth status` 确认是否过期
2. 若过期，重新执行 `gitlink-cli auth login`
3. 非交互环境（CI/脚本）改用环境变量：`export GITLINK_TOKEN="your-token"`

## 决策规则

| 条件 | 操作 |
|------|------|
| 用户首次使用 / 未登录 | 引导 `auth login` |
| 报错 `401` | Token 失效或过期，重新 `auth login` |
| 报错 `403` | 已登录但无权限，确认 owner/repo 正确性，非认证问题 |
| CI / 脚本等非交互环境 | 使用 `GITLINK_TOKEN` 环境变量，避免交互式登录 |
| 切换账号 | 先 `auth logout` 再 `auth login` |

## 注意事项

- GitLink Token 有效期 **7 天**，过期需重新登录或刷新 Token
- Token 存储在系统密钥管理器（macOS Keychain / Linux Secret Service / Windows Credential Manager），Fallback 为 `~/.config/gitlink-cli/credentials`
- **禁止** 将 Token 明文输出、打印或写入版本控制的文件
- 非交互环境优先使用 `GITLINK_TOKEN` 环境变量，而非交互式 `auth login`
- `auth logout` 会清除本地凭证，下次操作前需重新登录

## References

- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证原理、Token 说明、401/403 错误处理、全局参数、安全规则
- [examples/auth-workflow.md](examples/auth-workflow.md) — 认证完整工作流示例
