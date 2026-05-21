# Defense Q&A

## 1. 作品解决了什么问题？

本作品解决开源项目维护中信息整理成本高的问题。维护者面对大量 Issue、PR 和仓库状态信号时，往往需要手动判断优先级、风险和下一步动作。本项目把这些判断沉淀为 GitLink CLI 原生命令，提供可复现、可解释、只读安全的工作流分析能力。

## 2. 为什么不使用 LLM？

本阶段不使用 LLM 是为了降低依赖风险和运行成本，并保证输出稳定可测。比赛目标是贡献可落地的 CLI 功能，规则型分析更适合进入基础工具链。未来如果需要接入 LLM，也可以在稳定 DTO 和安全边界之上扩展，而不是直接绑定外部模型。

## 3. 规则型分析如何体现智能化？

智能化不等于必须调用大模型。本项目通过关键词、权重评分、缺失信息检测、风险标记和健康度评分，把维护经验转化为自动化规则。每个结论都有 reasoning 和 matched rules，维护者可以理解判断来源，Agent 也可以消费结构化结果继续处理。

## 4. 如何保证不污染远端仓库？

所有 workflow 命令都遵守只读边界。远端模式只 fetch 数据，不评论、不打标签、不关闭 Issue，也不 approve、reject 或 merge PR。命令默认生成分析结果和建议，不执行写操作，因此不会改变真实仓库状态，适合在评审和演示中安全运行。

## 5. 与普通 CLI 命令相比有什么区别？

普通 CLI 命令主要完成单个 API 操作，例如查看 Issue 或 PR。本作品新增的是工作流级分析命令，会聚合输入、应用规则、生成风险等级、建议和报告。它不是简单包装 API，而是为维护者和 AI Agent 提供更高层的协作决策辅助。

## 6. 与人工维护 Issue / PR 相比有什么价值？

人工维护仍然是最终决策者，但本项目能先完成重复的信息整理工作。例如自动识别 bug、docs、security，指出缺失复现信息，给出 PR 审阅重点和测试建议。这样维护者可以把时间放在判断和修复上，而不是反复阅读和归类。

## 7. json/table/markdown 三种输出分别面向谁？

`json` 面向 AI Agent 和脚本，字段稳定，便于自动处理；`table` 面向终端用户，适合快速查看摘要；`markdown` 面向维护者、Issue/PR 评论草稿和比赛文档，便于复制传播。三种输出复用同一分析结果，减少重复实现。

## 8. repo-report 的评分如何计算？

`repo-report` 以 health score 为基础，结合高风险 Issue、缺失信息数量、高风险或 critical PR 等信号进行扣分，并限制在 0 到 100。风险等级按分数区间划分；如果出现 security P0 Issue 或 critical PR，会提升整体风险等级，保证安全问题优先暴露。

## 9. 如果 GitLink API 字段变化怎么办？

fetch 层使用 response normalization 处理多种字段形态，例如不同的 author、label、release、PR 字段别名。如果真实 API 继续变化，后续只需要在 workflow fetch 层补充映射和 httptest，不需要修改规则引擎或输出协议，维护成本较低。

## 10. 为什么成果可以落地到 gitlink-cli 主仓库？

实现遵循现有 shortcuts 架构，没有修改 `cmd/` 和 `internal/output`，也没有新增第三方依赖。功能边界清楚、默认只读、测试覆盖集中，适合以 PR 形式提交到主仓库。维护者可以分阶段 review，不需要一次接受复杂平台级改造。

## 11. 当前局限是什么？

当前局限主要是远端 API 形态仍需更多真实项目验证，`repo-report` 的远端 PR 部分使用 PR 列表元数据，深度 files/commits 分析仍通过单独的 `workflow +pr-summary` 完成。此外 `release-notes` 和 `stale` 仍是后续规划，尚未实现。

## 12. 后续规划是什么？

后续计划包括 `workflow +release-notes`、`workflow +stale`、更完整的真实 GitLink API 字段归一化、官方 Skill 收录申请和更多真实项目验证。所有后续功能仍会坚持只读优先、可测试、可解释，不会默认执行破坏性远端操作。

## 13. 如何验证功能正常？

可以运行 `gofmt -w shortcuts/workflow/*.go shortcuts/register.go`、`go test ./shortcuts/workflow` 和 `go test ./...`。演示时优先使用 `shortcuts/workflow/testdata/` 下的 JSON 文件，避免网络和认证影响。远端命令也只读，可作为 smoke 验证。

## 14. 如果 PR 没被合并，成果落地如何体现？

子赛题一鼓励提交官方 PR。即使短期未合并，只要 PR 已提交、CI 通过并进入维护者 Review，就已经具备成果落地基础。项目还提供个人仓库、完整测试、文档、演示脚本和后续迭代计划，便于根据维护者反馈继续推进。

## 15. 这个项目如何服务 AI Agent？

AI Agent 需要稳定、结构化、可解释的工具输出。本项目为 Issue、PR、健康度和仓库报告提供稳定 JSON DTO，并保留 reasoning、risk、recommendations 等字段。Agent 可以读取这些结果，生成后续任务、报告或维护计划，而不依赖不稳定的自然语言解析。
