# 提交核对清单

## 官方交付要求映射

| 要求 | 本项目对应内容 |
| --- | --- |
| 工作流串联不少于 3 个 CLI 命令或 Skill 调用 | `scripts/gitlink_workflow.py` 串联 `repo +info`、`issue +list`、`pr +list`、`release +list`，并支持 `issue +comment` 发布摘要 |
| 提供可复现执行脚本或 Agent 对话记录 | `scripts/run_demo.ps1` |
| 在至少一个真实 GitLink 项目上运行并展示效果 | `docs/verification.md`、`docs/demo-output.md`、`examples/demo_outputs/` |
| 提供工作流说明文档 | `README.md`、`docs/quickstart.md`、`docs/runbook.md` |
| 提供架构图 | `docs/architecture.md` 引用 `docs/assets/architecture-workflow-v2.svg` |
| 代码开源并托管到 GitLink | `https://gitlink.org.cn/puygob236/gitlink-cli` 的 `examples/workflows/community-ops-automation/` |
| 提供完整中文 README | `README.md` |
| 开源协议 | `LICENSE`，Apache 2.0 |

## 验证状态

- `python -m py_compile .\scripts\gitlink_workflow.py .\tests\test_gitlink_workflow.py`：通过
- `python -m unittest discover -s tests`：通过
- `.\scripts\run_demo.ps1`：已在 `Gitlink/gitlink-cli` 上跑通
- `.\scripts\run_demo.ps1 -Config examples\sample_config.json`：已在 `puygob236/gitlink-cli` 上跑通
- `.\scripts\run_demo.ps1 -Config examples\sample_config.json -PublishIssueId 2`：已完成 Issue 摘要回写验证

## 交付内容

- `README.md`、`docs/`、`scripts/`、`examples/`、`tests/`、`LICENSE` 均位于 `examples/workflows/community-ops-automation/`。
- `outputs/` 为运行时生成目录，评审可通过复现脚本重新生成。
- `examples/demo_outputs/` 提供固定示例产物，便于快速查看报告格式和输出内容。
