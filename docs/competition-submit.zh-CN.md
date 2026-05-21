# GitLink CLI Agent Workflow 增强套件参赛说明

## 1. 作品概述

作品名称：GitLink CLI Agent Workflow 增强套件。

本作品基于 `Gitlink/gitlink-cli` 开源仓库，面向开源项目维护者和 AI Agent
增加规则型、可解释、只读安全的协作分析工作流能力。目标不是替代维护者，
而是把 Issue 分诊、仓库健康度评估、PR 审阅摘要和仓库工作流报告变成
CLI 原生命令，降低维护前的信息整理成本。

当前已实现四个命令：

- `workflow +triage`
- `workflow +health`
- `workflow +pr-summary`
- `workflow +repo-report`

## 2. 对应赛题与场景

本作品主要对应子赛题一：`gitlink-cli` 功能增强 / 开源项目贡献。

对应工作流场景：

- Issue 自动分拣
- PR Review 辅助
- 仓库健康度评估
- 仓库工作流报告生成

## 3. 功能完整性说明

### workflow +triage

`workflow +triage` 对 Issue 做规则型智能分诊，支持本地参数、`--from`
JSON 文件和 GitLink 远端只读 fetch。输出包括 Issue 类型识别、优先级判断、
缺失信息检测、风险标记、建议操作、建议评论、规则命中原因，并支持
`json`、`table`、`markdown` 三种格式。

### workflow +health

`workflow +health` 生成仓库健康度评分，覆盖 Issue / PR backlog、最近活跃度、
Release、CI、文档、License、CONTRIBUTING 和 Agent readiness 等指标。
当远端 API 或某些指标不可用时，命令不会伪造结果，而是标记为 unknown
并在 scoring notes 中说明，保证 Agent 和维护者可以判断可信度。

### workflow +pr-summary

`workflow +pr-summary` 对单个 PR 生成审阅摘要，分析 PR 元数据、changed files
和 commits，输出 change type、risk level、review focus、test suggestions、
merge checklist 和 reasoning。该命令只读，不评论、不 approve、不 reject、
不 merge。

### workflow +repo-report

`workflow +repo-report` 聚合 health、triage 和 pr-summary 的能力，生成一份
仓库工作流报告。报告包含整体分数、风险等级、Issue 分布、PR 风险分布、
维护建议和判断依据，适合维护者、比赛材料和 AI Agent 使用。

## 4. 创新性说明

- Agent-native structured output：`json` 给 Agent，`markdown` 给维护者，`table` 给终端用户。
- Rule-based explainable intelligence：不依赖 LLM，所有判断都有规则依据。
- Safety-first read-only workflow：远端模式只读，不污染仓库状态。
- GitLink CLI 原生集成：基于 shortcuts 架构，不是外部脚本。
- Repository workflow report：一条命令聚合多个仓库治理维度。
- Bilingual support：支持 `en` / `zh-CN`。

## 5. 实用价值说明

| 问题 | 对应功能 | 价值 |
|---|---|---|
| 开源仓库 Issue 积压 | `workflow +triage` | 快速识别类型、优先级和缺失信息 |
| PR 审阅前理解成本高 | `workflow +pr-summary` | 生成审阅重点、测试建议和合并清单 |
| 仓库维护状态不清晰 | `workflow +health` | 给出健康度评分、风险等级和修复建议 |
| 维护者需要汇总报告 | `workflow +repo-report` | 一条命令生成可复制的仓库工作流报告 |
| AI Agent 需要稳定输出 | `json` DTO | 字段稳定，适合脚本和 Agent 消费 |

## 6. 技术路线

- Go + Cobra
- `gitlink-cli` shortcuts 架构
- GitLink API 只读 fetch
- API response normalization
- rule engine
- health scoring
- workflow-local renderer
- `json` / `table` / `markdown`
- `httptest` mock
- testdata reproducible examples

## 7. 安全边界

- 不调用 LLM API
- 不执行远端写操作
- 不自动评论
- 不自动打标签
- 不关闭 Issue
- 不 approve / reject / merge PR
- 不修改 `internal/output`
- 远端模式只读 fetch
- 所有分析在本地规则层完成

## 8. 测试与验证

测试命令：

```bash
gofmt -w shortcuts/workflow/*.go shortcuts/register.go
go test ./shortcuts/workflow
go test ./...
```

测试覆盖：

- triage rules
- health scoring
- pr-summary rules
- repo-report aggregation
- fetch normalization
- partial failure
- `json` / `table` / `markdown` render
- `--from` testdata
- command smoke tests

## 9. 可复现演示命令

稳定本地演示命令：

```bash
gitlink-cli workflow +triage --from shortcuts/workflow/testdata/issue_bug.json --format table
gitlink-cli workflow +health --from shortcuts/workflow/testdata/health_good.json --format markdown
gitlink-cli workflow +pr-summary --from shortcuts/workflow/testdata/pr_summary.json --format markdown
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format markdown
```

远端只读演示命令：

```bash
gitlink-cli workflow +triage --owner Gitlink --repo gitlink-cli --state open --limit 5 --format table
gitlink-cli workflow +health --owner Gitlink --repo gitlink-cli --stale-days 30 --format table
gitlink-cli workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 1 --format markdown
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format markdown
```

如果真实远端受认证、网络或 API 形态影响，可使用 `--from` testdata 稳定复现。

## 10. 成果落地计划

- 当前成果已在个人仓库完成。
- 下一步将提交 GitLink 官方主仓库 PR。
- 以 PR 已提交且通过 CI 作为成果落地基础。
- 争取在 Review 后根据维护者意见迭代。
- `release-notes` / `stale` 作为后续规划。

## 11. 后续规划

- `workflow +release-notes`
- `workflow +stale`
- 更完整的真实 GitLink API field normalization
- 官方 Skill 收录申请
- 更多真实项目验证
