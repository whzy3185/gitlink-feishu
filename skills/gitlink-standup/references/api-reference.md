# gitlink-standup API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能全程只读。

## 采集的接口

### 用户动态

```
GET /users/:login/project_trends.json?page=<n>&limit=<n>
```

返回 `project_trends[]`，使用字段：

| 字段 | 说明 |
|------|------|
| `trend_type` | 活动类型：CommitLog / Issue / PullRequest / VersionRelease |
| `action_type` | 动作描述（如"创建了代码提交(Commit)"） |
| `action_time` | 相对时间（如"3天前"） |
| `name` | 活动标题（提交信息 / Issue 标题 / PR 标题） |

### 用户信息（可选）

```
gitlink-cli user +info --login <login> --format json
```

## 时间说明

`action_time` 为相对时间字符串，非精确时间戳，因此本技能按**活动条数 + 类型**汇总，
通过 `--limit` 控制采集的近期活动量，而非按精确日期区间过滤。

## 输出字段（JSON）

```json
[
  {"login": "wbtiger", "total_activities": 30,
   "by_type": {"CommitLog": 26, "PullRequest": 4},
   "samples": {"CommitLog": ["..."], "PullRequest": ["..."]}}
]
```

## 错误处理

某成员动态采集失败时，记为 0 活动，不中断其他成员。
