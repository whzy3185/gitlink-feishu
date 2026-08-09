# Capability shortcut

新增 `capability` 命令组 + `internal/capability` 包，提供 GitLink 后端 API 能力探测：

- `capability +check` — 向后端发送探测请求，检查各命令模块依赖的 API 是否就绪，结果缓存 24 小时
- `capability +list` — 查看已缓存的能力探测结果

实现要点：

- 新增 `internal/capability` 包：`Registry` 记录各域（label/notification/pm/wiki/pipeline/webhook/member/milestone/export/search/workflow 等）的可用状态（Available/Unavailable/Error/Unknown），带 24 小时缓存（`~/.config/gitlink-cli/capabilities.json`）。
- 探测复用 `internal/client`，对需要 owner/repo 上下文的域自动从 git remote 推断或 `--owner/--repo` 指定。
- `+check` 输出表格（模块 / 状态 / 说明）；`+list` 直接读缓存不发请求，过期会提示。

背景：不同 GitLink 实例后端能力不一致，命令调用前无法预知某模块是否可用。`capability` 组让用户/Agent 在调用前自检，提升跨实例兼容性与错误可诊断性。含完整单元测试（状态判定、探测 200/401/403/404、HTML 响应、repo 上下文等）。
