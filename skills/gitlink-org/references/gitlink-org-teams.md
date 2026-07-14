# org +teams

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

列出指定组织下的团队。

## 命令

```bash
# 列出组织团队
gitlink-cli org +teams --id Gitlink

# 分页
gitlink-cli org +teams --id Gitlink --page 1 --limit 50

# JSON 格式输出，便于 Agent 解析
gitlink-cli org +teams --id Gitlink --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | 组织标识（login name 或 ID） |
| `--page` / `-p` | 否 | 页码（默认 1） |
| `--limit` / `-l` | 否 | 每页数量（默认 20） |
| `--format` | 否 | 输出格式：json / table / yaml |

## 输出字段

返回组织团队列表，字段取决于 GitLink API 返回结构，常见字段包括：

| 字段 | 说明 |
|------|------|
| `id` | 团队 ID |
| `name` | 团队名称 |
| `description` | 团队描述 |
| `members_count` | 团队成员数量 |
| `projects_count` | 团队项目数量 |

## 注意事项

- 该命令是只读操作，不会修改组织团队配置。
- 创建团队、删除团队或批量调整团队项目仍需使用 Raw API，执行写入/删除前必须确认用户意图。
- 如果需要查看团队成员，请先确认 GitLink API 是否公开对应团队成员端点，再使用 Raw API。

## References

- [gitlink-org](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
