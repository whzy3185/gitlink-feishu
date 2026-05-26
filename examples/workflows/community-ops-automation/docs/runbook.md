# 运行手册

## 前置条件

- 已安装 `gitlink-cli`
- 已完成 `gitlink-cli auth login`
- 目标仓库有可读权限

官方快速开始里要求的验证命令是：

```powershell
gitlink-cli user +me
```

## 运行方式

### 1. 只生成报告

```powershell
python .\scripts\gitlink_workflow.py --config .\examples\sample_config.json
```

### 2. 生成报告并发布摘要

```powershell
python .\scripts\gitlink_workflow.py --config .\examples\sample_config.json --publish-issue-id 123
```

### 3. 一键复现

```powershell
.\scripts\run_demo.ps1
```

## 输出文件

- `outputs/*_report.md`：完整周报
- `outputs/*_release_notes.md`：Release Notes 草稿
- `outputs/*_summary.json`：结构化摘要

## 验证清单

- `repo +info` 能返回仓库信息
- `issue +list` 能返回 Issue 列表
- `pr +list` 能返回 PR 列表
- `release +list` 能返回 Release 列表
- 报告文件能落盘
- Release Notes 草稿能落盘
- 发布模式能把摘要写回指定 Issue

## 真实项目配置

- `examples/demo_active_config.json` 指向 `Gitlink/gitlink-cli`，用于验证活跃公开仓库的数据分析能力。
- `examples/sample_config.json` 指向 `puygob236/gitlink-cli`，用于验证参赛仓库的采集和 Issue 回写能力。
