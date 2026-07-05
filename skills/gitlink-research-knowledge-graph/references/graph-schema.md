# Research knowledge graph schema

Node types:

| Type | Required fields | Description |
|---|---|---|
| `repo` | `id`, `owner`, `repo`, `label` | GitLink repository |
| `language` | `id`, `name` | Programming language |
| `contributor` | `id`, `login` | Public contributor identity |
| `topic` | `id`, `name` | Keyword or inferred research topic |
| `issue` | `id`, `title`, `state` | Representative Issue |
| `pr` | `id`, `title`, `state` | Representative PR |

Edge types:

| Type | From | To | Evidence |
|---|---|---|---|
| `uses_language` | repo | language | `repo +languages` |
| `contributed_by` | contributor | repo | `repo +contributors` |
| `matches_topic` | repo | topic | search keyword or description |
| `has_issue` | repo | issue | `issue +list` |
| `has_pr` | repo | pr | `pr +list` |

Example:

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

