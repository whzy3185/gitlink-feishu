---
name: gitlink-pr-review-warroom
version: 1.1.0
description: "以只读方式整理 GitLink PR 审查队列和单个 PR 的 patchset、Reviewer 最后有效决定、线程正文、采集完整性与下一步建议。适用于维护者在集中 Review、比赛收尾或贡献高峰期间建立共享审查工作台。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli workflow --help"
---

# GitLink PR Review Warroom

开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，了解认证、仓库定位和输出约定。

本 Skill 只读取 GitLink 数据并生成本地报告。不得自动评论、审批、拒绝、请求 Reviewer、解决线程或合并 PR。

## 目标

帮助维护者完成：

1. 拉取完整的开放 PR 队列。
2. 按风险、规模和测试信号排序。
3. 为重点 PR 获取稳定的 `review.context/v1` 上下文。
4. 区分当前 Review、过期 Review 和无法确认版本的 Review。
5. 识别当前版本仍需响应的 Review 线程。
6. 生成可交给飞书、企业微信或其他协作适配层的只读 JSON/Markdown。

## 第一步：建立审查队列

```bash
gitlink-cli workflow +review-queue \
  --owner <owner> \
  --repo <repo> \
  --state open \
  --all \
  --limit 50 \
  --max-items 1000 \
  --format json
```

`--all` 只发出连续的 GET 请求。`--max-items` 是防止异常分页导致无限拉取的安全上限。

优先处理：

- `priority=high`
- `risk_level=critical|high`
- 大文件数或大 diff
- 测试信号不明确

队列分数只用于排序，不等于最终 Review 结论。

## 第二步：获取单个 PR Review Context

```bash
gitlink-cli workflow +review-context \
  --owner <owner> \
  --repo <repo> \
  --number <pr-number> \
  --include-versions \
  --include-reviews \
  --include-threads \
  --format json
```

核心字段：

```text
schema_version
current_head_sha
current_patchset
review_records[].commit_id
review_records[].freshness
reviewer_summaries[].latest_effective_review
reviewer_summaries[].current_decision
threads[].state
threads[].need_respond
threads[].freshness
threads[].content
review_summary
collection_status
partial
section_statuses
fetch_errors
work_item.source_fingerprint
work_item.unknowns
```

没有平台凭据时，可使用固定 fixture 完成离线演示：

```bash
gitlink-cli workflow +review-context \
  --from shortcuts/workflow/testdata/review_context_p1_fixture.json \
  --format markdown
```

## 第三步：解释版本新鲜度

只使用下列规则：

```text
review.commit_id == current_head_sha  -> current
review.commit_id != current_head_sha  -> outdated
任一字段缺失                         -> unknown
```

允许安全的七位以上 SHA 前缀匹配。更短的前缀不得视为同一提交。

- `outdated` 保留为历史证据，不参与当前版本通过或阻断判断。
- `unknown` 必须保守处理，不得据此判断可合并。
- `approved` 仅代表存在当前批准，不等于 `merge_ready`。
- 同一 Reviewer 的多次当前 Review 以最后一条可可靠排序的决定为准。
- 多条冲突决定无法按时间或数字 ID 排序时，保持 `unknown` 并交给人工。

## 第四步：解释线程

- 当前版本 `opened && need_respond=true`：贡献者仍需响应。
- 当前版本 `type=problem && state=opened`：视为待处理问题。
- `resolved`：保留历史，不计入待响应数。
- `disabled`：保留历史，不参与当前决定。
- `unknown_parent=true`：回复引用的父记录缺失，必须报告。
- 过期线程不自动阻断当前版本，但应作为历史上下文展示。
- 报告必须保留 `threads[].content`；正文缺失时明确显示平台未返回内容。

## 第五步：检查采集完整性

在解释 Review 结论前先检查：

```text
collection_status
partial
section_statuses[]
fetch_errors[]
work_item.source_scope
```

- `complete`：必需分段均成功、未采样，且版本绑定没有未知项。
- `partial`：至少一个分段失败、达到读取上限或版本绑定未知。
- `failed`：没有取得可用只读分段。
- `partial=true` 时，不得用本次空值覆盖协作平台中的上一份完整镜像。
- `fetch_errors` 必须进入维护者告警和重试队列，不能仅记录自由文本。

## 第六步：形成维护者报告

```bash
gitlink-cli workflow +review-context \
  --owner <owner> \
  --repo <repo> \
  --number <pr-number> \
  --format markdown
```

报告必须区分：

- GitLink PR 状态；
- GitLink 正式 Review 状态；
- Review 新鲜度；
- 线程状态；
- 聚合决定；
- 每名 Reviewer 的最后有效决定；
- 采集完整性和结构化错误；
- 未知项；
- 推荐下一步。

不要把 `work_item.review_stage=ready_for_decision` 描述为已经允许合并。

## 安全边界

本 Skill 禁止调用：

```text
pr +review
pr +review-comment
pr +review-comment-update
pr +review-comment-delete
pr +merge
api POST|PUT|PATCH|DELETE
任何飞书或企业微信 send/apply 操作
```

如果用户要求写回，停止本流程并明确说明 P1 只读边界。写回必须进入单独阶段，重新读取 head SHA，生成 dry-run，并获得显式确认。

## 验证

```bash
go test ./shortcuts/workflow -run 'ReviewContext|ReviewQueue'
go run . workflow +review-context --help
go run . workflow +review-queue --help
```
