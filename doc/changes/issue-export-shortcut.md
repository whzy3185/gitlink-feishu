# Issue 批量导出命令

新增 `gitlink-cli issue +export`，用于把筛选后的 Issue 跨页导出为 CSV、JSON 或 Markdown。维护者经常需要把 Issue 列表带出 GitLink，用于周报、迁移、离线排查或交给脚本/AI Agent 做进一步分析；过去只能手动翻页复制或依赖原始 API 拼参数，容易漏页，也不方便统一字段。

命令复用 `issue +list` 的常用筛选条件，包括状态、关键词、参与范围、作者、负责人、里程碑、状态 ID、标签和排序参数；同时增加 `--limit`、`--max` 控制导出规模，`--fields` 控制输出字段，`--export-format` 选择 CSV/JSON/Markdown，`--output` 写入文件。不传 `--output` 时会直接把导出内容输出到 stdout，方便管道处理。

实现上新增独立的 Issue 导出分页逻辑，兼容 GitLink Issue 列表返回的 `issues` 包装结构，并把嵌套的状态、优先级、作者、负责人、标签等字段规范化为稳定列。已补充单元测试覆盖筛选参数、多页导出、`--max` 截断、Markdown 转义和非法参数校验。
