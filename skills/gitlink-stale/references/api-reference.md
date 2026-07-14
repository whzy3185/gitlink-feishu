# gitlink-stale API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能采集只读；关闭操作由维护者确认执行。

## 采集的接口

### Issue 列表

```
GET /:owner/:repo/issues.json?page=1&limit=50
```

使用字段：

| 字段 | 用途 |
|------|------|
| `id` / `name` / `subject` | 编号与标题 |
| `issue_status` | 过滤已关闭（含"关"/closed 的跳过） |
| `updated_at` / `created_at` | 相对时间，估算停滞天数 |
| `author_login` | 作者 |

### PR 列表

```
GET /:owner/:repo/pulls.json?page=1&limit=50
```

| 字段 | 用途 |
|------|------|
| `pull_request_number` | 编号 |
| `pull_request_status` | 仅取 0（open） |
| `pr_time` / `updated_at` | 相对时间，估算停滞天数 |

## 时间估算

`updated_at`/`created_at` 多为相对时间字符串（"3天前"/"2个月前"/"1年前"），
本技能用正则换算为天数：天/周(×7)/月(×30)/年(×365)，"刚刚/小时"记为 0。
这是粗略估算，用于分级而非精确计时。

## 写操作（关闭）

```
gitlink-cli issue +close --owner <o> --repo <r> --number <web序号>
```

属写操作，需用户确认。本技能只生成建议清单。

## 输出字段（JSON）

```json
{"total_open": 27, "by_grade": {"活跃":20,"留意":6,"陈旧":1},
 "cleanup_count": 1, "cleanup": [{"kind","id","title","age_days","grade","author"}]}
```
