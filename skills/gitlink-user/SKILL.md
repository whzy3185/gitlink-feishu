---
name: gitlink-user
version: 1.1.0
description: "用户操作：查看当前用户、用户详情、贡献热力图、活跃度、开发能力、角色定位和专业定位统计。当用户需要查看 GitLink 用户信息或用户统计画像时触发。"
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
| `user +activity` | 查看用户近期活跃度统计 | 否 |
| `user +headmap` | 查看用户贡献热力图，可按年份过滤 | 否 |
| `user +develop` | 查看用户开发能力统计 | 否 |
| `user +role` | 查看用户角色定位统计 | 否 |
| `user +major` | 查看用户专业定位 / 项目分类统计 | 否 |

## 使用示例

```bash
# 查看当前用户
gitlink-cli user +me --format json

# 查看其他用户
gitlink-cli user +info --login zhangsan --format json

# 用户近期活跃度
gitlink-cli user +activity --login zhangsan --format json

# 用户贡献热力图
gitlink-cli user +headmap --login zhangsan --year 2026 --format json

# 用户开发能力、角色定位、专业定位
gitlink-cli user +develop --login zhangsan --start-time 1717200000 --end-time 1719800000 --format json
gitlink-cli user +role --login zhangsan --format json
gitlink-cli user +major --login zhangsan --format json
```

## 参数说明

- `--login` 不传时优先使用全局 `--owner`，否则通过 `/users/me` 解析当前登录用户。
- `--start-time` / `--end-time` 为 Unix 时间戳，必须是非负整数，且 `start-time <= end-time`。
- `--year` 必须是四位年份。
- 新增统计命令全部是只读 `GET` 操作，适合 Agent 做开源贡献画像、科研仓库成员分析和自动报告。

## Raw API 补充

```bash
# 用户贡献热力图
gitlink-cli api GET /users/:user_id/headmaps

# 用户统计
gitlink-cli api GET /users/:user_id/statistics/activity
gitlink-cli api GET /users/:user_id/statistics/develop
gitlink-cli api GET /users/:user_id/statistics/role
gitlink-cli api GET /users/:user_id/statistics/major

# 用户项目动态
gitlink-cli api GET /users/:user_id/project_trends
```
