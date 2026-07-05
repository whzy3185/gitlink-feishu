# 科研 Skills 套件

本次新增 7 个面向 GitLink 科研仓库的只读 Agent Skills：

- `gitlink-research-reproducibility`：科研代码复现性清单和整改计划。
- `gitlink-research-compliance`：许可证、安全策略、依赖和敏感文件风险审计。
- `gitlink-research-progress-tracker`：科研项目周报和风险预警。
- `gitlink-research-collaboration-map`：面向课题组的贡献者、Issue 和 PR 协作画像。
- `gitlink-research-knowledge-graph`：基于关键词的科研生态调研和轻量知识图谱工作流。
- `gitlink-research-data-provenance`：数据集来源、引用、许可证和隐私风险审计。
- `gitlink-research-artifact-handbook`：面向答辩、交接和开源发布的科研成果手册生成。

这些 Skills 按独立 `gitlink-cli` Skill PR 交付物设计，不依赖外部项目代码，也可被端到端科研工作流组合调用。

验证方式：

```bash
make validate-research-skills
git diff --check
```

真实 Agent 验证说明见 `docs/research-skills-agent-test-report.md`。
