# 科研知识图谱结构

节点类型：

| 类型 | 必填字段 | 说明 |
|---|---|---|
| `repo` | `id`, `owner`, `repo`, `label` | GitLink 仓库 |
| `language` | `id`, `name` | 编程语言 |
| `contributor` | `id`, `login` | 公开贡献者身份 |
| `topic` | `id`, `name` | 关键词或推断出的研究主题 |
| `issue` | `id`, `title`, `state` | 代表性 Issue |
| `pr` | `id`, `title`, `state` | 代表性 PR |

边类型：

| 类型 | 起点 | 终点 | 证据 |
|---|---|---|---|
| `uses_language` | repo | language | `repo +languages` |
| `contributed_by` | contributor | repo | `repo +contributors` |
| `matches_topic` | repo | topic | 搜索关键词或描述 |
| `has_issue` | repo | issue | `issue +list` |
| `has_pr` | repo | pr | `pr +list` |

示例：

```json
{
  "nodes": [
    {"id": "repo:Gitconomy/Git4Research", "type": "repo", "label": "Gitconomy/Git4Research"},
    {"id": "topic:open-research", "type": "topic", "label": "open research"}
  ],
  "edges": [
    {"source": "repo:Gitconomy/Git4Research", "target": "topic:open-research", "type": "matches_topic", "evidence": "search +repos keyword"}
  ]
}
```
