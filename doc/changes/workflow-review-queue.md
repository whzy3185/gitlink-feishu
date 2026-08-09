# 新增 PR 审查队列工作流

`gitlink-cli workflow +review-queue` 新增了面向维护者的 PR 审查队列排序能力。命令可以从 GitLink 只读拉取 open PR 列表，也可以读取本地 JSON 输入；每个 PR 会复用已有 `workflow +pr-summary` 的变更类型、风险等级、审查重点和测试建议，再结合 diff 规模、提交数量、测试信号等因素计算优先级分数。

该能力适合 open PR 较多的仓库先做审查排队：高风险修复、涉及认证/API/核心路径的大改动会被排在前面，文档类和低风险小改动会自然下沉。输出支持 `table`、`markdown` 和 `json`，终端查看、PR/Issue 评论草稿以及脚本消费都可以直接复用。

命令保持工作流模块的安全边界：远端模式只发起 `GET /v1/{owner}/{repo}/pulls` 请求，不会写入评论、审批、拒绝或合并 PR。测试覆盖了本地队列排序、对象和数组两种 JSON 输入、远端查询参数、中文 markdown 渲染和 table 输出，确保新增功能可验证、可审查。
