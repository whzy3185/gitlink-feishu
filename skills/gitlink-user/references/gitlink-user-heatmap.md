# user +heatmap

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看用户贡献热力图。

## 命令

```bash
# 指定用户和年份
gitlink-cli user +heatmap --user zhangsan --year 2026

# 省略 --user 时使用当前认证用户
gitlink-cli user +heatmap --year 2026

# JSON 格式，便于 Agent 解析
gitlink-cli user +heatmap --user zhangsan --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--user` / `-u` | 否 | 用户登录名；省略时使用当前认证用户 |
| `--year` | 否 | 热力图年份，例如 `2026` |
| `--format` | 否 | 输出格式：json / table / yaml |

## 输出字段

| 字段 | 说明 |
|------|------|
| `total_contributions` | 贡献总数 |
| `headmaps[].date` | 贡献日期 |
| `headmaps[].contributions` | 当日贡献数 |

## References

- [gitlink-user](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
