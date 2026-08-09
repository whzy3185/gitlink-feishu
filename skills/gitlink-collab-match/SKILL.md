---
name: gitlink-collab-match
version: 1.0.0
description: "科研协作智能匹配（子赛题四·S4）：分析科研仓库的技术缺口（未解决 Issue 主题/语言、开放 PR、研究空缺），结合候选人科研画像，智能匹配跨团队/跨学者协作伙伴。当用户要找协作者、推荐合作者、分析仓库需要什么样的人时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    python: ["scripts/research/requirements.txt"]
  cliHelp: "gitlink-cli research +match --help"
  scenario: "S4"
---

# gitlink-collab-match — 科研协作智能匹配

> 子赛题四「应用 GitLink 辅助科研」· 场景 **S4 科研协作智能匹配**

## 何时使用

- 课题组/科研团队想为一个科研代码仓库寻找合适的协作伙伴（跨团队/跨学者）。
- 想知道「这个仓库当前最缺哪方面的人/技能」。
- 为开源科研项目做人员招募建议、互补团队推荐。

## 前置条件

1. 已 `gitlink-cli auth login`（Token 7 天有效）。
2. 已 `pip install -r scripts/research/requirements.txt`（本场景实际只用标准库 + topics 词典，无需重型依赖）。
3. 目标仓库存在且有若干未解决 Issue（缺口信号来源）。

## 工作流

本 Skill 的算法由 `scripts/research/match.py` 实现（Go 出数据 + Python 做匹配）：

1. **缺口分析**：调 `issue +list --state open`（按优先级加权）+ `pr +list --state open` + `repo +languages` + README，用 `topics.py` 词典抽取出仓库的**缺口主题向量**与**需求语言**。
2. **候选池**：本仓库贡献者（`repo +contributors`，过滤 bot）+ 按缺口主题用 `search +users` 搜到的外部用户，上限默认 15。
3. **候选人画像**：对每个候选人调 `repo +list --user <login>`，聚合其公开仓库的主题向量、语言集合、fork 数（协作开放度）、活跃度。
4. **综合打分**：
   `score = 0.45×主题重叠(余弦) + 0.20×语言匹配(Jaccard) + 0.20×活跃度 + 0.15×协作开放度`（×100）。
5. **产物**：`match.json`（结构化）+ `report.md`（中文推荐报告，含缺口表 + 排名表 + 理由）+ `network.mmd`（Mermaid 协作网络图）。

## 命令

```bash
# 默认输出到 stdout（JSON）
python scripts/research/match.py --owner mindspore-Ecosystem --repo mindspore

# 输出三件产物到目录
python scripts/research/match.py --owner <OWNER> --repo <REPO> --top 10 --pool 15 --out ./out

# 可复现脚本（封装了上述流程）
bash skills/gitlink-collab-match/examples/collab-match-workflow.sh <OWNER> <REPO> [OUT_DIR]
```

## 输出结构（match.json）

```json
{
  "scenario": "S4_collaboration_matching",
  "repo": "owner/repo",
  "gap_topics": ["deep_learning", "computer_vision", "..."],
  "needed_languages": ["python", "..."],
  "gap_signals": [{"type":"unresolved_issue","topic":"deep_learning","evidence":"...","priority":"高"}],
  "candidates": [{"login":"...","score":17.0,"topic_overlap":0.31,"language_match":0.5,
                  "activity_level":"high","repo_languages":["python"],"reasons":["覆盖缺口主题: ..."]}]
}
```

## 验证

已在真实科研仓库 **`mindspore-Ecosystem/mindspore`**（20346 条 issue）上验证：
缺口主题正确识别为 deep_learning / scientific_computing / RL / CV 等；
Top 推荐为仓库真实活跃贡献者（yefeng / He_Wei / gaoyong10）。

## 兼容性

兼容 Claude Code 等 AI Agent：本 SKILL.md 即为 Agent 编排依据，
Agent 可直接调上述命令并把产物读回做进一步解读与文案化。
