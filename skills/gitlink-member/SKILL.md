---
name: gitlink-member
description: "仓库成员管理：列出、添加、批量添加、移除成员，调整成员角色，生成/查看/接受邀请链接，通过邀请码申请加入项目，退出仓库成员关系。"
metadata:
  cliHelp: "gitlink-cli member --help"
---

# gitlink-member（仓库成员管理）

当用户需要管理 GitLink 仓库成员、成员角色或邀请链接时使用本 Skill。

## 常用命令

| 命令 | 用途 |
|------|------|
| `member +list` | 列出仓库成员 |
| `member +add` | 通过用户 ID 添加仓库成员 |
| `member +batch-add` | 通过用户 ID 列表或 CSV 批量添加成员 |
| `member +remove` | 通过用户 ID 移除仓库成员 |
| `member +role` | 修改成员角色 |
| `member +invite-link` | 获取或生成当前邀请链接 |
| `member +invite-info` | 查看邀请链接信息 |
| `member +accept-invite` | 接受邀请链接 |
| `member +apply` | 通过项目邀请码申请加入项目 |
| `member +quit` | 退出当前仓库成员关系 |

## 示例

```bash
# 列出仓库成员
gitlink-cli member +list --owner Gitlink --repo forgeplus

# 添加成员
gitlink-cli member +add --owner Gitlink --repo forgeplus --user-id 101

# 批量添加前预览
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --user-ids 101,102 --dry-run

# 从 CSV 批量添加。CSV 支持 user_id、userid、id 列；无表头时读取第一列。
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --from members.csv

# 修改角色。角色支持 Manager、Developer、Reporter，也支持小写别名。
gitlink-cli member +role --owner Gitlink --repo forgeplus --user-id 101 --role Developer

# 获取或生成当前邀请链接。role 支持 manager、developer、reporter；apply 表示是否需要审核。
gitlink-cli member +invite-link --owner Gitlink --repo forgeplus --role developer --apply true

# 查看邀请链接信息
gitlink-cli member +invite-info --owner Gitlink --repo forgeplus --sign <invite_sign>

# 接受邀请链接
gitlink-cli member +accept-invite --owner Gitlink --repo forgeplus --sign <invite_sign>

# 通过邀请码申请加入项目。role 支持 manager、developer、reporter，建议先 dry-run。
gitlink-cli member +apply --code MPzQgH --role developer --dry-run
gitlink-cli member +apply --code MPzQgH --role developer

# 退出仓库成员关系。真实执行必须显式 --yes。
gitlink-cli member +quit --owner Gitlink --repo forgeplus --dry-run
gitlink-cli member +quit --owner Gitlink --repo forgeplus --yes
```

## 安全规则

- 执行 `member +remove`、`member +role`、`member +add`、`member +batch-add` 前，确认目标仓库和用户 ID。
- 执行 `member +apply`、`member +quit` 前，确认邀请码、目标仓库和期望角色；优先使用 `--dry-run` 预览。
- `member +quit` 会让当前用户离开仓库，真实执行必须带 `--yes`。
- 批量添加前优先使用 `--dry-run` 预览。
- 避免在公开日志中暴露邀请链接的完整 `sign`。
