# gitlink-issueops 速查参考

## webhook 合法事件名（`--events`）

来自 `shortcuts/webhook/webhook.go` 的白名单，逗号分隔多选：

| 事件 | 触发时机 |
|------|---------|
| `issues_only` | Issue 创建/状态变化（IssueOps 主事件） |
| `issue_comment` | Issue 评论 |
| `issue_assign` | Issue 指派 |
| `issue_label` | Issue 标签变化 |
| `pull_request_only` / `pull_request_assign` / `pull_request_comment` | PR 对应事件 |
| `push` / `create` / `delete` | 代码推送 / 引用创建 / 删除 |

注意：写 `issues` 会报 `invalid --events value`，必须用 `issues_only`。

## `webhook +tasks` 返回结构（实测）

```jsonc
{
  "ok": true,
  "data": {
    "hooktasks": [
      {
        "id": 4836246,              // 投递任务 id —— 去重游标用它
        "event_type": "issues",     // issues / issue_comment / ...
        "is_delivered": true,
        "is_succeed": true,         // 端点是否成功响应（失败也会记录负载！）
        "delivered_time": "2026-06-12 10:20:50",
        "payload_content": {
          "action": "opened",       // opened / ...
          "issue": {
            "id": 144169,                  // 全局 id
            "project_issues_index": 3,     // 仓库内编号 —— issue +comment --number 用它
            "subject": "[agent] …",
            "description": "…",
            "author": {"login": "recorder", "...": "…"},
            "tags": []
          },
          "repository": {"...": "…"},
          "sender": {"...": "…"}
        }
      }
    ]
  }
}
```

要点：
- **端点收不到也有记录**（`is_succeed: false` 而已），Replay 模式因此成立；
- 投递是异步的，创建 Issue 后约几秒到几十秒可见，轮询间隔建议 ≥30s；
- 去重：持久化已处理的最大 `id`，只处理更大的。

## Issue 的两套 id 与两套 API（最容易踩的坑）

| 用途 | 用哪个 id | 端点 |
|------|----------|------|
| 回帖 | 仓库内编号（`project_issues_index`） | `issue +comment --number <n>` |
| 看详情 | 编号 | `issue +view --number <n>` 或 v1 `GET /api/v1/:owner/:repo/issues/<n>.json` |
| **普通 Issue 打标签** | 编号 | **v1 `PATCH /api/v1/:owner/:repo/issues/<n>.json`，body `{"tag_ids":[<tag_id>]}`**（200 即生效） |
| PR 背后 issue 打标签 | 全局 id | 老 API `POST /api/:owner/:repo/issues/<全局id>.json`，body `{"issue_tag_ids":[…],"subject":…}`（gitlink-gatekeeper 实测） |

实测教训：
- 对**普通 Issue** 用老 API `POST /issues/<全局id>` 会 404；普通 Issue 一律走 v1 `PATCH` + `tag_ids`。
- v1 PATCH 即使只带 `tag_ids` 也返回 200 并生效，不必回带 subject/description。
- `tag_id` 从 `label +list --format json` 里查（`name` 匹配）。

## 已知 CLI 行为（0.2.0 实测）

- `gitlink-cli api POST "/:owner/:repo/…" --owner X --repo Y` 单次调用**不替换** `:owner/:repo` 占位符（请求会带着字面 `:owner` 发出去 → 404）。规避：写字面路径 `/X/Y/…`；占位符替换仅 `--batch-file` 配 `--var` 时可用。
- `label +create` 成功响应只有 `{"message":"success"}` 不带 id，需再 `label +list` 反查。

## Live vs Replay 对比

| | Live 回调 | Replay 轮询 |
|--|----------|------------|
| 公网端点 | 需要 | **不需要** |
| 实时性 | 秒级 | 轮询间隔（建议 ≥30s） |
| 基础设施 | 自建服务 + 签名校验 | 零（只用 CLI） |
| 丢事件 | 端点宕机会丢（可用 Replay 补偿） | 不丢（平台记录所有投递任务） |
| 适合 | 生产常驻 | 个人/演示/补偿通道 |
