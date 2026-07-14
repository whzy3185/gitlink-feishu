# 新增依赖风险审计工作流

`gitlink-cli workflow +dependency-audit` 新增了面向 Go 项目的本地依赖风险审计能力。命令可以直接读取 `go.mod`，也可以读取结构化 JSON 输入，输出 `table`、`markdown` 或 `json` 报告，便于维护者在合并前快速判断依赖变更是否需要重点复核。

审计规则保持确定性和可复现，不联网查询版本信息，而是专注于 go.mod 中已经存在的风险信号：本地 `replace`、pseudo version、预发布版本、语义化主版本和模块路径不匹配、缺失或过旧的 `go` 指令等。报告会给出风险等级、扣分后的健康分、finding 列表和处理建议，适合放进 PR 审查、发布前检查或自动化脚本。

本次变更包含命令注册、go.mod/JSON 输入解析、风险评分、中文和英文输出、README 示例、中文 README 说明以及单元测试。测试覆盖了 go.mod 块解析、JSON 输入、本地 replace、pseudo version、主版本不匹配、预发布版本、markdown/table 渲染和 shortcut 注册。
