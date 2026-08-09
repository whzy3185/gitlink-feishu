---
name: gitlink-research-progress
version: 1.0.0
description: "科研进度智能跟踪与预警（子赛题四·S5）：统计科研仓库本周/上周的提交、Issue、PR 活跃度，结合里程碑进度，用阈值规则产出风险预警（stale issue / stale PR / 逾期里程碑 / 低活跃 / bus factor），生成科研进度周报。当用户要做项目周报、进度跟踪、风险预警、里程碑监控时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "python scripts/research/report.py --help"
  scenario: "S5"
---

# gitlink-research-progress — 科研进度智能跟踪与预警

> 子赛题四「应用 GitLink 辅助科研」· 场景 **S5 科研进度智能跟踪与预警**

## 何时使用

- 课题组负责人想每周自动生成「科研项目进度周报」。
- 想及时发现项目停滞（低活跃）、Issue/PR 堆积、里程碑逾期、单点依赖（bus factor）。
- 为科研项目做里程碑监控与风险预警。

## 前置条件

1. 已 `gitlink-cli auth login`。
2. 本场景为纯标准库实现，无需额外 pip 依赖（`scripts/research/report.py`）。

## 工作流

算法由 `scripts/research/report.py` 实现（Go 出数据 + Python 做统计/预警）：

1. **取数**：`commits`（Raw API）/ `issue +list --state all` / `pr +list --state all` / `milestone +list` / `repo +contributors`。
2. **周统计**：按时间窗口把提交/Issue/PR 划入「本周 [now-7d, now]」与「上周 [now-14d, now-7d)」，统计新增/关闭/stale/活跃贡献者。
3. **里程碑进度**：按 `milestone_name` 归集 Issue 的 open/closed，算完成率与逾期。
4. **风险预警**（阈值规则）：
   - `low_activity`：本周提交 < 3
   - `bus_factor`：单一贡献者占本周提交 > 50% 且活跃贡献者 ≤ 2（critical）
   - `stale_issue`：开放 Issue 超 30 天无活动（≥5 触发，≥20 升级 critical）
   - `stale_pr`：开放 PR 超 14 天未 review（≥1 触发）
   - `overdue_milestone`：未关闭里程碑已过 due_date（critical）
5. **趋势**：本周 vs 上周 commit 环比，给出 increasing/stable/decreasing。
6. **产物**：`report.json`（结构化）+ `weekly_report.md`（中文周报：活动对比表 + 里程碑表 + 风险预警表）。

## 命令

```bash
python scripts/research/report.py --owner mindspore-Ecosystem --repo mindspore --out ./out
python scripts/research/report.py --owner O --repo R            # 仅打印 JSON

# 可复现脚本
bash skills/gitlink-research-progress/examples/progress-report-workflow.sh <OWNER> <REPO> [OUT_DIR]
```

## 输出结构（report.json）

```json
{
  "scenario": "S5_progress_tracking", "repo": "owner/repo",
  "week_stats": {"this_week": {...}, "last_week": {...}, "window": {...}},
  "trend": {"commit_delta_pct": 12.5, "activity_level": "increasing"},
  "milestones": [{"name":"v1.0","open":2,"closed":1,"completion_pct":33.3,"overdue":true}],
  "risk_warnings": [{"level":"critical","type":"bus_factor","message":"...","suggestion":"..."}]
}
```

## 验证

已在真实科研仓库 **`mindspore-Ecosystem/mindspore`** 上验证取数与统计口径；
11 个纯单元测试覆盖时间解析、周分桶、stale 判定、bus factor、里程碑逾期、趋势计算。

## 兼容性

兼容 Claude Code 等 AI Agent：本 SKILL.md 为编排依据，Agent 调上述命令并把周报读回做解读与跟进建议。
