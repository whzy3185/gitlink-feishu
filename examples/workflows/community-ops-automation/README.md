# GitLink 构建端到端自动化工作流

面向 GitLink 竞赛子赛题三的端到端自动化工作流项目。

本项目面向开源社区运营场景，使用 `gitlink-cli` 串联仓库信息、Issue、PR 和 Release 数据采集，自动生成社区周报、Release Notes 草稿和结构化摘要，并支持将摘要发布到指定 GitLink Issue。该流程覆盖“数据采集 -> 指标分析 -> 文档生成 -> 结果发布”的完整闭环。

## 交付物

- `scripts/gitlink_workflow.py`：主工作流入口
- `scripts/run_demo.ps1`：一键复现脚本
- `docs/architecture.md`：架构图与流程说明
- `docs/quickstart.md`：最短复现路径
- `docs/runbook.md`：运行手册
- `docs/verification.md`：真实仓库验证记录
- `docs/submission-checklist.md`：参赛提交核对清单
- `docs/upload-to-gitlink.md`：仓库目录结构说明
- `examples/sample_config.json`：参赛仓库配置
- `examples/demo_active_config.json`：公开仓库验证配置
- `examples/demo_outputs/`：真实运行示例产物
- `tests/test_gitlink_workflow.py`：单测
- `LICENSE`：Apache 2.0

## 运行方式

推荐直接运行一键脚本：

```powershell
.\scripts\run_demo.ps1
```

切换到参赛仓库配置：

```powershell
.\scripts\run_demo.ps1 -Config examples\sample_config.json
```

## 输出

- `outputs/*_report.md`
- `outputs/*_release_notes.md`
- `outputs/*_summary.json`

## 已验证仓库

- `puygob236/gitlink-cli`：完成仓库信息、Issue、PR、Release 采集，并完成 Issue 摘要回写验证
- `Gitlink/gitlink-cli`：完成仓库信息、Issue、PR、Release 采集，并生成包含有效统计数据的周报、Release Notes 和结构化摘要

## 项目定位

- 满足子赛题三“端到端自动化工作流”的要求
- 串联 4 个数据采集命令和 1 个结果发布命令
- 支持在真实 GitLink 项目上复现
- 提供运行脚本、验证记录、示例产物和单元测试
