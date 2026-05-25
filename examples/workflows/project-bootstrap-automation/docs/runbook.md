# 运行手册

## 模式说明

| 模式 | 命令 | 说明 |
| --- | --- | --- |
| dry-run | `.\scripts\run_demo.ps1` | 只生成材料和命令计划，不写入 GitLink |
| apply | `.\scripts\run_demo.ps1 -Apply` | 执行真实 `gitlink-cli` 命令 |
| apply + create repo | `.\scripts\run_demo.ps1 -Apply -CreateRepo` | 先创建仓库，再执行初始化命令 |
| apply + comment | `.\scripts\run_demo.ps1 -Apply -PublishIssueNumber 1` | 执行真实命令，并将摘要评论到指定 Issue |

## 配置文件

默认配置位于：

```text
examples/sample_project.json
```

主要字段：

- `project`：项目名称、描述、语言、许可证
- `repository`：目标 GitLink 仓库 owner/name
- `branches`：需要创建的协作分支
- `issues`：初始化 Issue 列表
- `publish.issue_number`：可选的摘要发布 Issue 编号

## 输出文件

| 文件 | 说明 |
| --- | --- |
| `*_bootstrap_report.md` | 初始化报告 |
| `*_summary.md` | 可发布到 Issue 的摘要 |
| `*_manifest.json` | 结构化初始化清单 |
| `*_files.json` | 生成文件内容包 |
| `command_log_*.json` | gitlink-cli 命令计划或执行结果 |

## 安全边界

- 默认 dry-run，不进行远端写操作。
- 只有显式传入 `-Apply` 才执行真实 GitLink 命令。
- `-PublishIssueNumber` 只在明确指定 Issue 编号时追加评论命令。
- 所有命令会写入 `command_log_*.json`，便于复盘和审计。
