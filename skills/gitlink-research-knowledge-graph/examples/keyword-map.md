# 关键词知识图谱示例

用户请求：

```text
请围绕“论文复现”和“open research”在 GitLink 上做科研热点搜索，并输出轻量知识图谱。
```

Agent 步骤：

```bash
gitlink-cli search +repos -k "论文复现" --format json
gitlink-cli search +repos -k "open research" --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

预期回答：

- 列出关键词和候选仓库。
- 提供 Markdown 趋势报告。
- 按 `references/graph-schema.md` 输出包含 `nodes` 和 `edges` 的 JSON。
