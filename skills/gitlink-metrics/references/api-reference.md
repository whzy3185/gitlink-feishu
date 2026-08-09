# gitlink-metrics API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能全程只读。

## 采集的接口

| 接口 | 用途 |
|------|------|
| `GET /:owner/:repo.json` | 规模指标（issues/pulls/releases/star/fork 计数） |
| `GET /:owner/:repo/pulls.json` | PR 合并率（`pull_request_status`） |
| `GET /:owner/:repo/commits.json` | 按月活跃趋势（`timestamp`） |
| `GET /:owner/:repo/contributors.json` | 集中度与评分（`contributions`） |

## 使用的字段

- 仓库：`issues_count` / `pull_requests_count` / `version_releases_count` / `praises_count` / `forked_count`
- PR：`pull_request_status`（0=open,1=merged,2=closed）
- 提交：`timestamp`（unix 秒）
- 贡献者：`contributions`

## 输出字段（JSON）

```json
{
  "scale": {"issues","pulls","contributors","releases","stars","forks"},
  "pr_metrics": {"total","merged","closed","open","merge_rate"},
  "monthly_commits": {"2026-05": 93},
  "concentration": {"gini","cr3","cr5","bus_factor"},
  "score_card": {"活跃度","协作","社区","可维护性","综合"}
}
```

## 错误处理

采集失败返回非零退出码并打印原因。提交无可定位时间戳时不计入活跃趋势。
