# 示例输出

本目录保存 `scripts/bootstrap_project.go` 在 dry-run 模式下生成的固定示例产物，便于快速查看工作流输出格式。

生成命令：

```powershell
go run scripts\bootstrap_project.go --config examples\sample_project.json --output-dir examples\demo_outputs --now 2026-05-24T08:00:00Z
```

产物说明：

- `*_bootstrap_report.md`：项目初始化报告
- `*_summary.md`：可发布到 Issue 的初始化摘要
- `*_manifest.json`：结构化初始化清单
- `*_files.json`：生成文件内容包
- `command_log_*.json`：gitlink-cli 命令计划
