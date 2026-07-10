---
name: gitlink-research-visual
version: 1.0.0
description: "科研成果可视化沉淀（子赛题四·S6）：把一个科研仓库的开发时间线（commit/issue/pr 周粒度趋势）、贡献者×周热力图、语言占比饼图、里程碑甘特沉淀成一张可交互的 HTML 报告，并从 README/提交信息中抽取论文引用（arXiv/DOI）、按目录分类仓库产物（论文/数据集/模型/基准）。当用户提到「科研成果可视化」「可视化沉淀」「开发节奏」「贡献热力图」「论文引用抽取」「产物分类」「research visualization」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    python: ["scripts/research/requirements.txt"]
  cliHelp: "gitlink-cli api --help"
  scenario: "S6"
---

# gitlink-research-visual — 科研成果可视化沉淀

> 子赛题四「应用 GitLink 辅助科研」· 场景 **S6 科研成果可视化沉淀**

## 何时使用

- 想把一个科研代码仓库的「成果产出」沉淀成可视化报告（给导师/合作者/项目主页展示）。
- 想看开发节奏：提交/Issue/PR 的周粒度趋势、贡献者活跃热力图。
- 想自动整理一份仓库里「能被引用的源头」清单：论文 arXiv/DOI 链接、数据集/模型/基准产物。
- 给开源科研项目做年度总结、阶段性汇报、成果网页的数据底座。

## 前置条件

1. 已 `gitlink-cli auth login`（Token 7 天有效）。
2. 已 `pip install -r scripts/research/requirements.txt`；本场景核心算法仅用标准库，
   生成交互 HTML 需 `plotly>=5.18`（已列入 requirements）。**未安装 plotly 时优雅降级**：仅写 `visual.json` + `report.md` 并提示。
3. 目标仓库存在且有提交历史（时间线/热力图的数据来源）。

## 工作流

本 Skill 的算法由 `scripts/research/visual.py` 实现（Go 出数据 + Python 做可视化）：

1. **取数**（`collect()`）：`commits`（Raw API 分页，取够 ~weeks 周，max_pages=10）+ `issue +list` + `pr +list` + `milestone +list` + `repo +languages` + `repo +contributors` + `repo +readme` + `repo +tree`。
2. **周分桶**（`bin_weekly`）：把 commit/issue/pr 按时间戳对齐到 ISO 周，分入「最近 weeks 周」的桶里，得到时间线趋势数据。
3. **贡献热力图**（`contribution_heatmap`）：top 贡献者 × 周桶的提交数矩阵，定位核心贡献者与活跃周期。
4. **论文引用抽取**（`extract_paper_links`）：正则匹配 arXiv（`arxiv.org/abs|pdf/`、`arXiv:`）与 DOI（`doi.org/`、裸 `10.xxxx/`），去重并带上下文片段。
5. **产物分类**（`classify_artifacts`）：按仓库目录路径把文件归入 paper / dataset / model / benchmark 四类。
6. **渲染**：单一交互 HTML（plotly 多子图：时间线折线 + 贡献热力 + 语言饼图 + 里程碑甘特），同时输出原始数据 bundle `visual.json`（供网页前端二次开发）+ `report.md`（中文摘要）。

## 命令

```bash
# 默认输出到 stdout（JSON）
python scripts/research/visual.py --owner mindspore-Ecosystem --repo mindspore

# 输出可视化三件套到目录（visual.html + visual.json + report.md）
python scripts/research/visual.py --owner <OWNER> --repo <REPO> --weeks 26 --out ./out

# 可复现脚本（封装了上述流程）
bash skills/gitlink-research-visual/examples/research-visual-workflow.sh <OWNER> <REPO> [OUT_DIR] [WEEKS]
```

## 输出结构（visual.json）

```json
{
  "scenario": "S6_research_visualization",
  "repo": "owner/repo",
  "weeks": 26,
  "timeline": {"labels": ["2026-W01", "..."], "commits": [...], "issues": [...], "prs": [...]},
  "heatmap": {"users": ["alice", "..."], "weeks": [...], "matrix": [[0,1,...], ...]},
  "languages": {"Python": "99.7%"},
  "milestones": [{"title": "v1.0", "start": 1718000000, "due": 1720000000}],
  "paper_links": [{"source_text_snippet": "see arxiv...", "target": "https://arxiv.org/abs/2401.00012", "type": "arxiv"}],
  "artifacts": [{"path": "data/train.csv", "category": "dataset", "name": "train.csv"}],
  "artifact_summary": {"paper": 3, "dataset": 5, "model": 2, "benchmark": 1},
  "meta": {"commit_count": 800, "issue_count": 20346, "pr_count": 9, "milestone_count": 4, "contributor_count": 6}
}
```

## 验证

已在真实科研仓库 **`mindspore-Ecosystem/mindspore`**（default_branch=master；issue≈20346；PR=9；贡献者=6）上验证：
周分桶正确对齐 ISO 周（无 plotly 时优雅降级为 `visual.json` + `report.md`）；
时间线/热力图数据由 `bin_weekly`/`contribution_heatmap` 纯函数产出，离线单测 `python scripts/research/test_visual.py` 全部通过（33 个用例，覆盖周分桶/热力矩阵/论文链接抽取/产物分类）。

## 兼容性

兼容 Claude Code 等 AI Agent：本 SKILL.md 即为 Agent 编排依据，
Agent 可直接调上述命令并把产物（HTML/JSON/Markdown）读回做进一步解读与文案化。
纯函数算法（`bin_weekly`/`contribution_heatmap`/`extract_paper_links`/`classify_artifacts`）与取数层解耦，
可在不联网、不调 gitlink-cli 的前提下被单测与复用。
