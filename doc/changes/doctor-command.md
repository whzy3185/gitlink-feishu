# CLI 自诊断命令

新增 `gitlink-cli doctor`，用于在用户遇到“无法认证、仓库识别失败、配置异常、API 请求失败”等问题时快速定位原因。命令会一次性检查配置文件是否存在且可解析、`base_url` 和 `default_format` 是否合理、本地 Token 或 `GITLINK_TOKEN` 是否可用、当前目录能否解析出 GitLink 仓库上下文，以及认证 API 是否能正常返回当前用户。

输出沿用项目已有的 `ok/data/error/meta` 结构，诊断结果包含每个检查项的状态、说明、细节和可执行修复建议，便于人类阅读，也便于 Agent 或 CI 解析。默认会验证认证 API 连通性，`--skip-network` 可在离线环境或 CI 中只做本地检查。

本次变更同时补充了中英文帮助文案、README 使用示例和单元测试。测试覆盖了正常本地检查、损坏配置文件、非法 `base_url`、仓库上下文缺失、认证 API mock 成功，以及命令 JSON envelope 输出，确保诊断命令在常见失败场景下返回结构化结果而不是直接崩溃。
