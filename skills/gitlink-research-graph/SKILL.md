---
name: gitlink-research-graph
version: 1.0.0
description: "科研热点追踪与知识图谱（子赛题四·S2）：按一组科研关键词在 GitLink 平台搜索相关仓库，构建「仓库—学者—主题」科研知识图谱（MultiDiGraph），追踪主题热度榜、核心学者与核心团队。当用户要梳理某个研究方向的全景、画知识图谱、找热点/核心学者/团队时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    python: ["scripts/research/requirements.txt"]
  cliHelp: "gitlink-cli search +repos --help"
  scenario: "S2"
---

# gitlink-research-graph — 科研热点追踪与知识图谱

> 子赛题四「应用 GitLink 辅助科研」· 场景 **S2 科研热点追踪与知识图谱**

## 何时使用

- 课题组/科研管理者想了解某研究方向（如「深度学习」「知识图谱」）在 GitLink 上的全景：有哪些仓库、哪些活跃学者、哪些是热点主题。
- 想把「关键词 → 仓库 → 学者/主题」的关系可视化为一张知识图谱（Mermaid / Graphviz）。
- 为开题、综述、找人合作做主题态势感知。

## 前置条件

1. 已 `gitlink-cli auth login`（Token 7 天有效）。
2. 已 `pip install -r scripts/research/requirements.txt`（本场景需 **networkx** 做图谱构建）。
3. 准备好一组逗号分隔的关键词（如 `"deep learning,nlp,knowledge graph"`）。

## 工作流

本 Skill 的算法由 `scripts/research/graph_build.py` 实现（Go 出数据 + Python 做建图）。严格分离「取数」与「建图」，便于离线单测：

1. **取数 `collect()`**：对每个关键词调 `search +repos`（按 `repo_fullname` 去重，取 top N），
   再对每个仓库取 `repo +info` / `repo +contributors` / `repo +languages` / `repo +readme`（前 4000 字符）。
2. **建图 `build_graph(repos, contributors_map, languages_map, readmes)`**（纯函数，不联网）：
   用 `networkx.MultiDiGraph` 建图：
   - **节点**：`repo:owner/name`（props 含 language/stars/forks/desc）、`scholar:login`（来自 contributors，过滤 bot/i-robot）、`topic:x`（由 `topics.py` 词典在 description+readme 上抽取）。
   - **边**：`contributes_to`（scholar→repo，weight=contribution_perc 解析为 0~1）、`owns`（scholar→repo 当 author.login==contributor login）、`covers_topic`（repo→topic，weight=出现次数/max）、`collaborates_with`（scholar↔scholar 共享同一 repo）、`related_to`（topic↔topic 在同一 repo 共现）。
3. **产物**：`graph.json`（结构化，含 scenario/nodes/edges/core_scholars/topic_heat/meta）+ `report.md`（中文趋势报告）+ `graph.mmd`（Mermaid，按 repo/scholar/topic 着色，仅画前 ~40 节点防爆炸）+ `graph.dot`（Graphviz DOT）。
4. **趋势报告**：主题热度榜（`topics.topic_counter` 在全部 description 上的 top10）+ 核心学者（按出现 repo 数）+ 核心团队。

## 命令

```bash
# 默认输出到 stdout（JSON）
python scripts/research/graph_build.py --keywords "deep learning,nlp"

# 输出四件产物到目录
python scripts/research/graph_build.py --keywords "knowledge graph,gnn" --repos-limit 20 --out ./out

# 可复现脚本（封装了上述流程）
bash skills/gitlink-research-graph/examples/knowledge-graph-workflow.sh <OWNER_KEYWORDS_EXAMPLE>
# 实际签名：bash knowledge-graph-workflow.sh <KEYWORDS> [OUT_DIR] [REPOS_LIMIT]
```

## 输出结构（graph.json）

```json
{
  "scenario": "S2_research_knowledge_graph",
  "keywords": ["deep learning", "nlp"],
  "nodes": [{"id":"repo:owner/name","type":"repo","label":"owner/name",
             "props":{"language":"Python","stars":120,"forks":30,"description":"..."}}],
  "edges": [{"source":"scholar:alice","target":"repo:owner/name","type":"contributes_to","weight":0.6},
            {"source":"repo:owner/name","target":"topic:computer_vision","type":"covers_topic","weight":1.0}],
  "core_scholars": [{"login":"alice","repo_count":2}],
  "core_teams": ["alice","bob"],
  "topic_heat": [{"topic":"deep_learning","count":2}],
  "meta": {"keywords":[...], "repo_count":2, "node_count":7, "edge_count":8,
           "scholar_count":3, "topic_count":2}
}
```

## 验证

已在真实科研仓库 **`mindspore-Ecosystem/mindspore`** 所在生态上验证（用关键词 `mindspore` 搜索生态仓库）：
图谱正确识别 deep_learning / scientific_computing / nlp / computer_vision 等主题节点，
covers_topic 权重落在 (0,1]，contributes_to 权重正确解析 contribution_perc（如 "60%"→0.6），
bot（i-robot）账号被过滤。Mermaid 输出以 `graph TD` 开头、含 repo/scholar/topic 三色 classDef，节点数被截断到 40 以内。

单测：`python scripts/research/test_graph_build.py`（17 个用例，不联网，构造 mock 数据喂 `build_graph`）。

## 兼容性

兼容 Claude Code 等 AI Agent：本 SKILL.md 即为 Agent 编排依据，
Agent 可直接调上述命令并把 graph.json/graph.mmd 读回做进一步解读与文案化。
Mermaid 块可被支持 Mermaid 渲染的 Markdown 查看器直接展示；graph.dot 可用 `dot -Tsvg graph.dot -o graph.svg` 渲染。
