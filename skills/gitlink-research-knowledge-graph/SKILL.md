---
name: gitlink-research-knowledge-graph
version: 0.1.0
description: "科研热点追踪与轻量知识图谱：按关键词搜索 GitLink 科研仓库，聚合仓库、语言、贡献者、Issue/PR 主题和研究方向，输出趋势分析、图谱节点边和选题建议。用于科研选题、技术生态调研和领域知识图谱构建。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli search --help"
---

# gitlink-research-knowledge-graph

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 只读执行。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 控制 API 调用量：每个关键词保留前 8 个仓库，深度分析不超过 8 个仓库。
- 对镜像仓库标注数据局限，不把 GitLink 平台内 watcher/fork 等同于原平台全量影响力。

## 工作流

1. 将研究主题拆成 3 到 5 个关键词，覆盖中文、英文、缩写和技术术语。

2. 搜索仓库：

```bash
gitlink-cli search +repos -k <keyword> --format json
```

3. 去重并选取候选仓库：

- 唯一键优先使用 `author.login/identifier`。
- 同一仓库匹配多个关键词时保留并记录全部关键词。
- 总候选控制在 20 个以内。
- 深度分析优先选择匹配关键词多、更新时间近、关注度高的 5 到 8 个。

4. 对重点仓库采集详情：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --format json
```

5. 按 `references/graph-schema.md` 输出图谱 JSON 和趋势报告。

## 输出格式

```markdown
# 科研热点与知识图谱：<topic>

## 搜索范围

关键词：...
命中仓库：...
深度分析：...

## 趋势洞察

1. ...

## 图谱摘要

| 节点类型 | 数量 |
|---|---:|
| repo | ... |
| language | ... |
| contributor | ... |
| topic | ... |

## 选题建议

1. ...
```

同时输出或附带如下 JSON 结构：

```json
{
  "nodes": [],
  "edges": []
}
```

## 分析原则

- 不要把搜索结果数量直接解释为学术热度，只能作为 GitLink 平台内信号。
- 图谱边必须有来源命令或字段依据。
- 选题建议要区分“成熟方向”“新兴方向”“生态空白”。
