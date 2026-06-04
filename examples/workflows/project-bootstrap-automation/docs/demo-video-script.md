# 演示视频脚本

本文档用于录制参赛演示视频。建议视频时长控制在 3 到 5 分钟，录屏范围包括终端、项目目录和 GitLink 页面。

## 录制前准备

1. 打开终端，进入仓库根目录。
2. 切换到 `project-bootstrap-automation-fork` 分支。
3. 确认当前目录无未提交运行产物。
4. 浏览器打开验证仓库页面：
   `https://gitlink.org.cn/puygob236/gitlink-bootstrap-demo`
5. 如需展示真实回写，提前完成 `gitlink-cli auth login`。

## 镜头一：项目定位

展示目录：

```powershell
cd examples\workflows\project-bootstrap-automation
Get-ChildItem
```

讲解要点：

- 本项目是 GitLink 子赛题三端到端自动化工作流。
- 场景是项目一键初始化与协作启动。
- 主实现为 Go，入口是 `scripts/bootstrap_project.go`。

## 镜头二：架构和交付物

展示文档：

```powershell
Get-Content docs\workflow-spec.md -TotalCount 40
Get-Content docs\architecture.md -TotalCount 35
```

讲解要点：

- 工作流分为配置输入、资产生成、CLI 编排、GitLink 落地、结果归档。
- 默认串联 `repo +info`、`branch +list`、`branch +create`、`issue +create`。
- 可选追加 `issue +comment` 完成结果回写。

## 镜头三：单元测试

执行命令：

```powershell
go test -count=1 ./scripts
```

讲解要点：

- 测试覆盖文件生成、CLI 计划、Issue 正文、输出 manifest 和幂等跳过判断。
- 测试通过后再进行 dry-run 演示。

## 镜头四：dry-run 复现

执行命令：

```powershell
.\scripts\run_demo.ps1
```

讲解要点：

- dry-run 不写入 GitLink，只生成材料和命令计划。
- 输出中应显示 7 个 `gitlink-cli` 调用计划。
- 该模式适合评审复现和本地检查。

展示输出：

```powershell
Get-ChildItem outputs
Get-Content outputs\command_log_*.json -TotalCount 80
```

## 镜头五：查看生成报告

执行命令：

```powershell
Get-Content outputs\*_bootstrap_report.md -TotalCount 80
Get-Content outputs\*_summary.md
```

讲解要点：

- 初始化报告包含目标项目、生成文件、分支计划和 Issue 计划。
- 摘要可用于回写到 GitLink Issue。

## 镜头六：真实仓库验证

展示 GitLink 页面：

```text
https://gitlink.org.cn/puygob236/gitlink-bootstrap-demo
```

讲解要点：

- 该仓库用于真实运行验证。
- 已验证仓库读取、分支读取、Issue 创建和 Issue 摘要回写。
- 真实写入命令记录在 `docs/verification.md`。

可展示命令：

```powershell
Get-Content docs\verification.md
```

## 镜头七：赛题要求映射

展示命令：

```powershell
Get-Content docs\submission-checklist.md
```

讲解要点：

- 工作流串联超过 3 个 CLI 调用。
- 提供可复现脚本。
- 已在真实 GitLink 项目上验证。
- 提供说明文档和架构图。

## 录制后清理

演示结束后删除运行时输出目录：

```powershell
Remove-Item outputs -Recurse -Force
```

`outputs/` 是可复现运行产物，不作为固定源码提交；固定示例保存在 `examples/demo_outputs/`。
