# 工作流说明

## 场景定位

本工作流面向 GitLink 子赛题三“构建端到端自动化工作流”，选择“项目一键初始化”作为应用场景。目标是在新开源项目创建初期，将项目配置、初始化文件、协作分支、初始 Issue 和执行报告统一串联，形成可复现、可审计的启动流程。

该场景覆盖开源项目常见的启动缺口：

- README、License、CI 配置和协作文档不完整。
- 初始任务缺少统一模板，Issue 粒度和验收标准不一致。
- 分支、Issue、报告产物分散，难以复盘初始化过程。
- 真实写入和演示复现之间缺少安全边界。

## 端到端流程

工作流由 `scripts/bootstrap_project.go` 实现，默认读取 `examples/sample_project.json`，并按以下顺序执行：

1. 解析项目配置，读取项目名称、仓库 owner/name、许可证、初始化分支和初始 Issue。
2. 生成初始化文件包，包括 README、LICENSE、CI 配置、贡献指南和路线图。
3. 规划或执行 `repo +info`，检查目标 GitLink 仓库状态。
4. 规划或执行 `branch +list`，读取分支状态。
5. 规划或执行 `branch +create`，创建协作分支。
6. 规划或执行 `issue +create`，创建初始化任务。
7. 可选执行 `issue +comment`，将初始化摘要回写到指定 Issue。
8. 生成 Markdown 报告、摘要、manifest、文件包和命令日志。

## 串联的 GitLink CLI 能力

默认 dry-run 配置会生成 7 个 `gitlink-cli` 调用计划：

| 顺序 | CLI 能力 | 用途 |
| ---: | --- | --- |
| 1 | `repo +info` | 检查目标仓库信息 |
| 2 | `branch +list` | 读取当前分支列表 |
| 3 | `branch +create` | 创建 `develop` 协作分支 |
| 4 | `branch +create` | 创建 `release/v0.1` 发布分支 |
| 5 | `issue +create` | 创建 README 与快速开始任务 |
| 6 | `issue +create` | 创建 CI 检查任务 |
| 7 | `issue +create` | 创建 v0.1 里程碑任务 |

当传入 `-PublishIssueNumber` 时，会追加 `issue +comment`，用于把初始化摘要发布到指定 GitLink Issue。

## 运行模式

| 模式 | 命令 | 行为 |
| --- | --- | --- |
| dry-run | `.\scripts\run_demo.ps1` | 生成材料和命令计划，不写入 GitLink |
| apply | `.\scripts\run_demo.ps1 -Apply` | 执行真实 GitLink CLI 命令 |
| apply + create repo | `.\scripts\run_demo.ps1 -Apply -CreateRepo` | 先创建仓库，再执行初始化流程 |
| apply + comment | `.\scripts\run_demo.ps1 -Apply -PublishIssueNumber 1` | 执行真实命令并回写摘要 |

## 输出产物

运行后会生成以下文件：

| 文件 | 说明 |
| --- | --- |
| `*_bootstrap_report.md` | 初始化报告，展示目标项目、生成文件、分支计划和 Issue 计划 |
| `*_summary.md` | 可发布到 Issue 的初始化摘要 |
| `*_manifest.json` | 结构化初始化清单 |
| `*_files.json` | 生成文件内容包 |
| `command_log_*.json` | gitlink-cli 命令计划或执行结果 |

固定示例输出保存在 `examples/demo_outputs/`，用于评审快速查看产物格式。`outputs/` 是运行时目录，可通过脚本重新生成。

## 工程边界

- 主实现使用 Go，便于与 `gitlink-cli` 主仓库技术栈保持一致。
- 默认 dry-run，避免演示阶段误写远端仓库。
- 真实写入必须显式传入 `-Apply`。
- 命令日志记录每个 CLI 调用的状态，便于复盘和排查。
- 测试覆盖文件生成、CLI 编排、Issue 内容生成、幂等跳过判断和输出 manifest。

## 赛题价值

该工作流不是单个命令封装，而是面向真实开源项目启动流程的组合式方案。它把 `gitlink-cli` 的仓库、分支、Issue 和评论能力整合为一个可复现闭环，符合子赛题三对“串联多个 CLI 命令或 Skill 调用”“真实项目运行展示”“工作流说明文档和架构图”的要求。
