# Demo Script

## 0:00-0:20 项目背景

屏幕内容：
- 打开仓库 README 的 Workflow Agent Commands 小节。
- 展示 `workflow +triage`、`workflow +health`、`workflow +pr-summary`、`workflow +repo-report`。

旁白：
本项目为 GitLink CLI 增加面向维护者和 AI Agent 的规则型工作流增强套件。
它不依赖 LLM，不做远端写操作，通过 CLI 原生命令提供 Issue 分诊、健康度评分、
PR 审阅摘要和仓库工作流报告。

截图点位：
- README workflow 命令列表。

## 0:20-0:50 workflow +triage

演示命令：

```bash
gitlink-cli workflow +triage --from shortcuts/workflow/testdata/issue_bug.json --format table
```

旁白：
这个命令会对 Issue 做规则型分诊，自动识别 Issue 类型，判断优先级，
发现缺失信息，并给出建议操作。表格输出适合维护者在终端快速查看。

截图点位：
- table 输出中的 type、priority、missing、action。

## 0:50-1:20 workflow +health

演示命令：

```bash
gitlink-cli workflow +health --from shortcuts/workflow/testdata/health_good.json --format markdown
```

旁白：
健康度命令根据 Issue、PR、Release、CI、文档、License 和贡献指南等指标，
生成仓库健康度评分、风险等级和维护建议。Markdown 输出可直接复制到报告中。

截图点位：
- health score、risk level、recommendations。

## 1:20-1:50 workflow +pr-summary

演示命令：

```bash
gitlink-cli workflow +pr-summary --from shortcuts/workflow/testdata/pr_summary.json --format markdown
```

旁白：
PR 摘要命令会识别 PR 类型和风险等级，生成 review focus、test suggestions
和 merge checklist。它只读分析，不会评论、approve、reject 或 merge PR。

截图点位：
- Review Focus、Test Suggestions、Merge Checklist。

## 1:50-2:20 workflow +repo-report

演示命令：

```bash
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format markdown
```

旁白：
仓库报告命令聚合 Issue、PR 和仓库健康度，生成一份完整的仓库工作流报告。
它适合维护者快速了解项目状态，也适合比赛展示项目的综合能力。

截图点位：
- Report score、Risk level、Issue Triage Summary、PR Review Summary。

## 2:20-2:40 JSON 输出给 Agent

演示命令：

```bash
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format json
```

旁白：
同一个报告可以输出稳定 JSON 字段，供 AI Agent 和脚本消费。
这让 Agent 可以基于结构化结果继续做排序、摘要或生成后续任务。

截图点位：
- JSON 中的 `report_score`、`risk_level`、`issue_summary`、`pr_summary`。

## 2:40-3:00 安全边界与总结

屏幕内容：
- 展示参赛说明中的安全边界小节。

旁白：
整个 workflow-agent 套件不依赖 LLM，远端模式只读，不评论、不打标签、不关闭 Issue，
也不 approve、reject 或 merge PR。后续规划包括 `workflow +release-notes`
和 `workflow +stale`，但当前提交保持聚焦、可测试、可落地。

截图点位：
- 安全边界列表。
