# pm — 项目管理命令

> 关联 Issue: #12 | PR: #9

## 概述

pm 模块提供 GitLink 项目管理相关的命令，包括仪表盘、Sprint 任务、周报、标签、流水线和 Action 运行记录的查看。

## 命令列表

### pm +dashboards
- **用途**: 查看项目仪表盘数据
- **API**: GET /pm/dashboards?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +dashboards --project 123`

### pm +sprints
- **用途**: 查看 Sprint 任务列表
- **API**: GET /pm/sprint_issues?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +sprints --project 123`

### pm +weekly
- **用途**: 查看周报任务
- **API**: GET /pm/weekly_issues?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +weekly --project 123`

### pm +tags
- **用途**: 查看项目 Issue 标签
- **API**: GET /pm/issue_tags?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +tags --project 123`

### pm +pipelines
- **用途**: 查看项目 CI/CD 流水线列表
- **API**: GET /pm/pipelines?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +pipelines --project 123`

### pm +runs
- **用途**: 查看项目 Action 运行记录
- **API**: GET /pm/action_runs?project_id=\<id\>
- **参数**: --project (必填) 项目 ID
- **示例**: `gitlink-cli pm +runs --project 123`

## 向后兼容性

无破坏性变更。所有命令通过 pm 域组 + 前缀添加。
