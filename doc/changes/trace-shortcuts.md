# Trace Shortcuts

新增 `trace` Shortcut 组，封装 GitLink 代码溯源分析相关接口，支持：

- `trace +init`
- `trace +results`
- `trace +start`
- `trace +rescan`
- `trace +report`

该能力覆盖账号初始化、按分支发起分析、分页查看分析结果、按项目重新扫描以及获取分析报告。`trace +start` 和 `trace +rescan` 支持 `--dry-run`，便于在自动化脚本或 AI Agent 执行前预览实际请求，降低误触发平台任务的风险。

同时补充了单元测试、README 示例、AI Agent Skill 文档和 i18n 文案，确保命令帮助、中文/英文提示和自动化使用场景保持一致。
