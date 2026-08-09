# Workflow Stale Report

## Summary

Adds a new read-only `workflow +stale` shortcut so maintainers can scan stale issues and pull requests without falling back to ad hoc scripts or Raw API calls.

## What it does

- Reads stale scan input from a local JSON file or fetches open queues directly from GitLink in read-only mode.
- Classifies each issue or pull request into `fresh`, `watch`, `stale`, or `zombie` buckets based on the last known activity time.
- Produces stable `json`, terminal-friendly `table`, and maintainer-friendly `markdown` output.
- Includes per-item next actions, suggested follow-up comments, queue-level recommendations, and fetch fallback notes.

## API handling

- Issue scan mode normalizes real GitLink fields such as `subject`, `status_name`, `status` objects, `issue_tags`, and `comment_journals_count`.
- PR scan mode uses list metadata first, then probes `/issues/{id}/journals` when the PR list lacks a reliable `updated_at` field.
- Journal fallback is best-effort: if probing fails, the report keeps the PR and records a note that age was estimated from creation time.

## Tests

- `go test ./shortcuts/workflow`
- `go test ./...`
- `go build ./...`
- `git diff --check`

## 中文说明

新增只读命令 `workflow +stale`，用于扫描仓库中的陈旧 Issue / PR 队列，并输出维护者可直接使用的处理报告。它支持本地 JSON 输入，也支持直接读取 GitLink 开放队列；会按最后活动时间分成 `fresh`、`watch`、`stale`、`zombie` 四档，并给出逐条建议动作、建议跟进评论和队列级建议。

为了让结果更贴近 GitLink 真实接口，这次同时补强了 Issue 字段兼容性，支持 `subject`、`status_name`、`status` 对象、`issue_tags`、`comment_journals_count` 等字段。对 PR，命令会优先读取列表元数据；当列表缺少可靠的更新时间时，再回退查询 `/issues/{id}/journals` 推断最近活动时间，失败时保留条目并在报告里说明是按创建时间估算。
