# user +statistics

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看用户聚合统计。`user +stats` 是该命令的短别名。

## 命令

```bash
# 指定用户
gitlink-cli user +statistics --user zhangsan

# 指定时间窗口（Unix 时间戳）
gitlink-cli user +statistics --user zhangsan --start-time 1704067200 --end-time 1735689600

# 使用短别名
gitlink-cli user +stats --user zhangsan --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--user` / `-u` | 否 | 用户登录名；省略时使用当前认证用户 |
| `--start-time` | 否 | 开始时间（Unix 时间戳） |
| `--end-time` | 否 | 结束时间（Unix 时间戳） |
| `--format` | 否 | 输出格式：json / table / yaml |

## 输出字段

返回 GitLink API 的用户统计结构。常见字段包括：

| 字段 | 说明 |
|------|------|
| `issues_count` | Issue 数量 |
| `pull_requests_count` | Pull Request 数量 |
| `commits_count` | 提交数量 |
| `projects_count` | 项目数量 |

## References

- [gitlink-user](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
