# Export shortcut

新增 `export` Shortcut 组，将仓库数据导出为 CSV / JSON 文件，支撑离线分析、科研数据抽取与外部报表：

- `export +issues` — 导出 Issue 列表（GET `/v1/:owner/:repo/issues`）
- `export +prs` — 导出 PR 列表（GET `/v1/:owner/:repo/pulls`）
- `export +contributors` — 导出贡献者统计（GET `/:owner/:repo/contributors`）

实现要点：

- 统一 `--format csv|json`（默认 csv）与 `--output` 输出路径；`issues`/`prs` 额外支持 `--state open|closed|all` 过滤与 `--page/--limit` 分页。
- CSV 表头固定（`id,title,state,created_at` 等），便于直接导入 Excel / pandas；JSON 保留原始字段，供 `workflow` 模块与科研 Skill 二次处理。
- 导出过程只读、分页拉取全量，避免一次性请求超限。

背景：此前要做仓库数据导出只能手工拼 Raw API 并自行解析分页。`export` 组将其提升为一等命令，是子任务四科研场景（贡献排行、Issue 趋势、PR 效率）的数据入口，并与 `gitlink-contributor-insight`、`gitlink-research-tracker` 等 Skill 衔接。关联 PR #15。
