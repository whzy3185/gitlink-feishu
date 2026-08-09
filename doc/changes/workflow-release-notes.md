# Workflow Release Notes Generator

## Summary

Adds `gitlink-cli workflow +release-notes`, a read-only workflow command that turns merged pull request metadata into structured release notes for maintainers, CI jobs, and AI agents.

## What It Does

- Reads PR data from a local JSON file or fetches merged pull requests from GitLink in read-only mode.
- Reuses the existing PR summary classifier to group changes into features, fixes, docs, tests, refactors, CI, mixed changes, and unknown changes.
- Highlights notable feature and high-risk PRs.
- Detects common breaking-change wording such as "breaking change", "incompatible", and "migration required".
- Produces suggested verification notes from PR analysis.
- Supports `json`, `table`, and `markdown`; markdown is the default when no global `--format` is set.

## Examples

```bash
# Generate markdown release notes from merged PRs.
gitlink-cli workflow +release-notes \
  --owner Gitlink --repo gitlink-cli \
  --version v1.2.0 --limit 20

# Generate release notes from local PR metadata for scripts or offline workflows.
gitlink-cli workflow +release-notes \
  --from release-prs.json \
  --from-ref v1.1.0 --to-ref v1.2.0 \
  --format json
```

## Input Shape

`--from` accepts either a `ReleaseNotesInput` object:

```json
{
  "repository": "Gitlink/gitlink-cli",
  "version": "v1.2.0",
  "from_ref": "v1.1.0",
  "to_ref": "v1.2.0",
  "pull_requests": [
    {
      "number": 42,
      "title": "feat: add export workflow",
      "author": "alice",
      "changed_files": [
        {"filename": "shortcuts/export/export.go", "additions": 120}
      ],
      "commits": [
        {"sha": "abc123", "message": "feat: add export workflow"}
      ]
    }
  ]
}
```

or a raw `[]PRSummaryInput` array.

## Tests

Unit tests cover shortcut registration, local object and array inputs, release note grouping, remote merged PR fetch parameters, markdown rendering, and table rendering.

## 中文说明

`workflow +release-notes` 面向发布前整理变更说明的场景。维护者可以从 GitLink 只读拉取已合并 PR，也可以在 CI 或离线脚本中传入本地 JSON。命令会复用现有 PR 分析规则，把变更自动归入新增能力、修复、文档、测试、重构、CI 等分组，并提取重点变更、潜在破坏性变更和建议验证项。它只生成本地输出，不会评论、审批、合并或修改远端数据，适合放进 Release 发布流程和比赛交付材料中。
