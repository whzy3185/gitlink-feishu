# GitLink 项目一键初始化与协作启动工作流

面向 GitLink 竞赛子赛题三的端到端自动化工作流示例。

本项目聚焦开源项目从 0 到可协作状态的启动过程，使用 `gitlink-cli` 串联仓库检查、分支规划、初始 Issue 创建和结果回写等能力，自动生成 README、LICENSE、CI 配置、协作文档、初始化报告和结构化清单。该流程覆盖“项目配置 -> 初始化文件生成 -> GitLink 命令编排 -> 任务落地 -> 报告归档”的完整闭环。

## 交付物

- `scripts/bootstrap_project.go`：主工作流入口
- `scripts/run_demo.ps1`：一键复现脚本
- `examples/sample_project.json`：示例项目配置
- `examples/verification_comment_config.json`：真实回写验证配置
- `examples/demo_outputs/`：固定示例输出
- `docs/workflow-spec.md`：工作流说明文档
- `docs/architecture.md`：架构与流程说明
- `docs/assets/bootstrap-architecture.svg`：架构图
- `docs/quickstart.md`：最短复现路径
- `docs/runbook.md`：运行手册
- `docs/verification.md`：验证记录
- `docs/submission-checklist.md`：赛题要求映射
- `scripts/bootstrap_project_test.go`：Go 单元测试

## 实现语言

本工作流主实现采用 Go，主要考虑如下：

- 与 `gitlink-cli` 主仓库技术栈一致，便于维护者阅读、测试和后续集成。
- 可直接复用 Go 标准库完成 JSON 配置解析、文件生成、命令编排和单元测试，不引入额外运行时依赖。
- Windows、Linux 和 macOS 均可通过 `go run` 复现，便于评审在不同环境中执行。
- 对命令执行结果、退出码和结构化日志的处理更接近 `gitlink-cli` 自身工程风格。

## 运行方式

进入本目录后执行 dry-run：

```powershell
.\scripts\run_demo.ps1
```

执行后会生成：

- `outputs/*_bootstrap_report.md`
- `outputs/*_summary.md`
- `outputs/*_manifest.json`
- `outputs/*_files.json`
- `outputs/command_log_*.json`

输出文件名包含目标仓库和生成时间，格式如下：

- `{owner}_{repo}_{YYYYMMDD_HHMMSS}_bootstrap_report.md`
- `{owner}_{repo}_{YYYYMMDD_HHMMSS}_summary.md`
- `{owner}_{repo}_{YYYYMMDD_HHMMSS}_manifest.json`
- `{owner}_{repo}_{YYYYMMDD_HHMMSS}_files.json`
- `command_log_{YYYYMMDD_HHMMSS}.json`

例如 `puygob236_gitlink-bootstrap-demo_20260524_080000_bootstrap_report.md`。实际运行时会按当前时间生成新文件名，`examples/demo_outputs/` 中的固定时间戳文件仅作为示例产物。

如需执行真实 GitLink 写操作，在完成 GitLink 认证并核对目标仓库后使用：

```powershell
.\scripts\run_demo.ps1 -Apply
```

如需连同仓库创建一起执行：

```powershell
.\scripts\run_demo.ps1 -Apply -CreateRepo
```

如需把初始化摘要发布到指定 Issue：

```powershell
.\scripts\run_demo.ps1 -Apply -PublishIssueNumber 1
```

## 工作流串联的 gitlink-cli 调用

默认配置会规划 7 个 `gitlink-cli` 调用：

1. `repo +info`
2. `branch +list`
3. `branch +create`
4. `branch +create`
5. `issue +create`
6. `issue +create`
7. `issue +create`

当指定 `-PublishIssueNumber` 时，会额外追加 `issue +comment`，用于把初始化摘要回写到 GitLink Issue。
当指定 `-CreateRepo` 时，会在检查仓库前追加 `repo +create`。

## 文档索引

- 工作流说明：`docs/workflow-spec.md`
- 架构说明与架构图：`docs/architecture.md`
- 复现指南：`docs/quickstart.md`
- 运行手册：`docs/runbook.md`
- 验证记录：`docs/verification.md`
- 提交核对清单：`docs/submission-checklist.md`

## 场景价值

- 降低新开源项目启动成本，避免 README、License、CI、初始任务缺失。
- 将项目初始化过程结构化，便于团队复用和审计。
- 将 `gitlink-cli` 的仓库、分支、Issue 和评论能力串联为可复现方案。
- 支持 dry-run 和 apply 两种模式，兼顾演示稳定性和真实落地。
