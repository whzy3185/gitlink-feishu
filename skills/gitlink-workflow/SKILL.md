---
name: gitlink-workflow
version: 1.0.0
description: "AI 自动化工作流：Issue 分类、PR Review、Release Notes 生成、仓库初始化、Sprint 报告等。当用户需要 AI 自动化 GitLink 操作时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli workflow --help"
---

# gitlink-workflow（AI 自动化工作流）

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

本技能提供 Claude Code 可直接执行的高级工作流模板。

## 工作流 1：Issue Triage（Issue 自动分类）

**场景**：自动为新 Issue 添加标签分类。

```bash
# 1. 获取未标记的 Issue 列表
gitlink-cli issue +list --state open --format json

# 2. 逐个查看 Issue 详情
gitlink-cli issue +view --id <issue_id> --format json

# 3. 根据内容分析，通过 Raw API 添加标签
gitlink-cli api POST /:owner/:repo/issues/:id --body '{"issue_tag_ids":[<tag_id>]}'
```

**分类规则建议**：
- 标题/描述包含 "bug"、"错误"、"失败" → bug 标签
- 标题/描述包含 "feature"、"新增"、"建议" → enhancement 标签
- 标题/描述包含 "question"、"如何"、"怎么" → question 标签

## 工作流 2：PR Review（代码审查辅助）

**场景**：获取 PR 变更，分析代码质量，添加 Review 评论。

```bash
# 1. 获取 PR 详情
gitlink-cli pr +view --id <pr_id> --format json

# 2. 获取变更文件列表
gitlink-cli pr +files --id <pr_id> --format json

# 3. 获取 PR 提交列表
gitlink-cli pr +diff --id <pr_id> --format json

# 4. 添加 Review 评论
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{"body":"代码审查意见...","event":"COMMENT"}'
```

## 工作流 3：Release Notes 生成

**场景**：从提交历史自动生成版本发布说明。

```bash
# 1. 获取两个版本之间的提交
gitlink-cli api GET /:owner/:repo/compare/:base...:head --format json

# 2. 获取已关闭的 Issue
gitlink-cli issue +list --state closed --format json

# 3. 生成 Release Notes 并创建发布
gitlink-cli release +create --tag v1.2.0 --name "v1.2.0" --body "## What's Changed\n- feat: 新功能 (#123)\n- fix: 修复问题 (#456)"
```

## 工作流 4：Repo Setup（仓库初始化）

**场景**：创建仓库并完成基础配置。

```bash
# 1. 创建仓库
gitlink-cli repo +create --name my-project --description "项目描述"

# 2. 设置分支保护
gitlink-cli branch +protect --name main --owner myuser --repo my-project

# 3. 创建初始 Issue
gitlink-cli issue +create --title "项目初始化" --body "- [ ] 完善 README\n- [ ] 配置 CI\n- [ ] 添加 License" --owner myuser --repo my-project
```

## 工作流 5：Sprint Report（Sprint 报告）

**场景**：汇总 Issue/PR 统计，生成周报。

```bash
# 1. 获取 Issue 统计
gitlink-cli issue +list --state open --format json
gitlink-cli issue +list --state closed --format json

# 2. 获取 PR 统计
gitlink-cli pr +list --state open --format json
gitlink-cli pr +list --state merged --format json

# 3. 获取项目动态
gitlink-cli api GET /:owner/:repo/activity --format json
```

## Workflow: PR Summary (Read-only)

Use `workflow +pr-summary` when a maintainer or Agent needs a structured PR review summary, review focus, test suggestions, or a markdown report that can be copied into a PR discussion.

```bash
# Read-only GitLink fetch mode
gitlink-cli workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 1 --format markdown

# Local JSON input mode for Agent pipelines
gitlink-cli workflow +pr-summary --from shortcuts/workflow/testdata/pr_summary.json --format json
```

Rules:
- Prefer `--format json` when another Agent consumes the output.
- Prefer `--format markdown` when a human maintainer needs a report.
- This command is read-only: it does not comment, approve, reject, merge, label, or close pull requests.
- Do not use LLM APIs for this workflow; it is rule-based and explainable.

## Workflow: Repo Report (Read-only)

Use `workflow +repo-report` when a maintainer or Agent needs a single repository workflow report
that aggregates health, issue triage, and PR review signals.
适用于需要生成仓库治理报告、比赛材料、维护者汇总或 Agent 综合分析的场景。

```bash
# Maintainer report
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format markdown

# Agent-readable report
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format json

# Local fixture mode
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format json
```

Rules:
- Prefer `--format json` when another Agent consumes the output.
- Prefer `--format markdown` for maintainer reports, competition materials, and review handoff.
- Treat remote mode as read-only aggregation only.
- Do not perform remote write operations.
- Do not comment, label, close, approve, reject, or merge from this workflow.
- PR details in remote report mode may be partial; use `workflow +pr-summary --number <n>` for a focused PR review.

## 最佳实践

- 所有工作流命令使用 `--format json` 以便解析输出
- 写入操作前确认用户意图
- 批量操作建议先用小范围测试
- 保存工作流执行结果以便回溯
