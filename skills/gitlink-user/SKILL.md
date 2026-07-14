---
name: gitlink-user
version: 1.0.0
description: "用户操作：查看当前用户、用户详情、Public Keys 和用户统计。当用户需要查看或管理 GitLink 用户信息时触发。"
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
| `user +current` | 当前用户详细资料 | 是 |
| `user +info` | 查看用户详情 | 否 |
| `user +keys` | 当前用户 Public Keys 列表 | 是 |
| `user +key-create` | 创建 Public Key | 是 |
| `user +key-delete` | 删除 Public Key | 是 |
| `user +activity` | 用户近期活动统计 | 否 |
| `user +headmap` | 用户贡献热力图 | 否 |
| `user +develop` | 用户开发能力统计 | 否 |
| `user +role` | 用户角色统计 | 否 |
| `user +major` | 用户专业定位统计 | 否 |

## 使用示例

```bash
# 查看当前用户
gitlink-cli user +me
gitlink-cli user +current --format json

# 查看其他用户
gitlink-cli user +info --login zhangsan

# Public Key 管理，写入/删除前先 dry-run
gitlink-cli user +keys --page 1 --limit 20
gitlink-cli user +key-create --title "work laptop" --key-file ~/.ssh/id_rsa.pub --dry-run
gitlink-cli user +key-delete --id 7 --dry-run

# 用户统计
gitlink-cli user +activity --login zhangsan
gitlink-cli user +headmap --login zhangsan --year 2026
gitlink-cli user +develop --login zhangsan --start-time 1704067200 --end-time 1735689599
gitlink-cli user +role --login zhangsan --start-time 1704067200 --end-time 1735689599
gitlink-cli user +major --login zhangsan --start-time 1704067200 --end-time 1735689599
```

## API 注意事项

- `user +key-create` 支持 `--key` 直接传入公钥内容，也支持 `--key-file` 从本地公钥文件读取。
- `user +key-create` 和 `user +key-delete` 支持 `--dry-run`，写入/删除前建议先预览请求。
- 统计命令使用用户 `login`，不是数字用户 ID。
