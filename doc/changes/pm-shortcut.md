# PM (项目管理) shortcut

新增 `pm` Shortcut 组，封装 GitLink 项目管理相关只读接口，补齐仓库协作元数据的命令化访问：

- `pm +dashboards` — 查看项目仪表盘数据
- `pm +sprints` — 查看 Sprint 任务列表
- `pm +weekly` — 查看周报任务
- `pm +tags` — 查看项目 Issue 标签
- `pm +pipelines` — 查看项目 CI/CD 流水线列表
- `pm +runs` — 查看项目 Action 运行记录

实现要点：

- 全部为只读（GET）命令，通过 Raw API 访问项目管理后端，统一 `owner/repo` 自动解析与 `--format json|table|yaml` 输出。
- 面向「项目经理 / 科研课题负责人」视角：一条命令拿到仪表盘、Sprint、周报、流水线运行等聚合视图，无需在 Web 上多次跳转。
- 与 `gitlink-pm` Skill 配套，供 AI Agent 做项目健康巡检与进度跟踪。

背景：项目管理数据此前散落在多个 Web 页面，无命令行入口。`pm` 组将其收敛为 6 条命令，是子任务三「项目一键初始化 / 进度跟踪」与子任务四「科研进度智能跟踪与预警」的基础数据层。关联 PR #12。
