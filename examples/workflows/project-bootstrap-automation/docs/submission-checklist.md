# 提交核对清单

## 官方交付要求映射

| 要求 | 本项目对应内容 |
| --- | --- |
| 工作流串联不少于 3 个 CLI 命令或 Skill 调用 | `scripts/bootstrap_project.go` 规划或执行 `repo +info`、`branch +list`、`branch +create`、`issue +create`、`issue +comment` |
| 提供可复现执行脚本或 Agent 对话记录 | `scripts/run_demo.ps1` |
| 在至少一个真实 GitLink 项目上运行并展示效果 | 已在 `puygob236/gitlink-bootstrap-demo` 完成仓库读取、分支读取、Issue 创建和 Issue 摘要回写验证 |
| 提供工作流说明文档 | `README.md`、`docs/quickstart.md`、`docs/runbook.md` |
| 提供架构图 | `docs/architecture.md` |
| 代码开源并托管到 GitLink | 放置于 `examples/workflows/project-bootstrap-automation/` |
| 提供完整中文 README | `README.md` |

## 验证状态

- `go test ./examples/workflows/project-bootstrap-automation/scripts`：通过
- `.\scripts\run_demo.ps1`：通过
- dry-run 生成 7 个 gitlink-cli 调用计划，满足赛题要求
- `.\scripts\run_demo.ps1 -Config examples\verification_comment_config.json -Apply -PublishIssueNumber 4`：通过，3 个真实 gitlink-cli 调用状态均为 `ok`

## 交付内容

- `README.md`、`docs/`、`scripts/`、`examples/` 均位于本目录。
- `outputs/` 为运行时生成目录，评审可通过复现脚本重新生成。
- `examples/demo_outputs/` 用于保存固定示例产物。
