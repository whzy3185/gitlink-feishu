# Keyword knowledge graph example

User request:

```text
请围绕“论文复现”和“open research”在 GitLink 上做科研热点搜索，并输出轻量知识图谱。
```

Agent steps:

```bash
gitlink-cli search +repos -k "论文复现" --format json
gitlink-cli search +repos -k "open research" --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

Expected answer:

- List keywords and candidate repositories.
- Provide a Markdown trend report.
- Provide `nodes` and `edges` JSON using `references/graph-schema.md`.

