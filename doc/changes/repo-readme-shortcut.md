# 完善仓库 README 快捷命令

`repo +readme` 现在补齐了中英文帮助文案、README 使用示例和 `gitlink-repo` Skill 说明，日常查看仓库 README 或子目录 README 时不再需要退回到 Raw API。

命令会把 `--ref` 传给 README API 的 `ref` 查询参数，把 `--path` 规范化后传给 `filepath` 查询参数，支持类似 `--path docs`、`--path /docs/guide/` 的子目录 README 查询方式。

本次变更同时补充了仓库 README 快捷命令的单元测试，覆盖默认查询、指定分支和子目录、HTTP 错误返回等路径，确保后续调整不会破坏请求路径或查询参数。
