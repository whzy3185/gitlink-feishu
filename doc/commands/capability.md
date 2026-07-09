# capability — API 后端能力探测

> 关联：跨平台兼容性 / 各模块可用性自检

## 概述

`capability` 模块向 GitLink 后端发送探测请求，检查各命令模块依赖的 API 是否就绪。结果缓存 24 小时（`~/.config/gitlink-cli/capabilities.json`），用于在 `--help` 里给不可用模块标注 ⚠，调用时给出中文错误指引，提升跨实例兼容性。

## 命令列表

### capability +check — 探测后端能力
向 GitLink 后端探测，检查各模块是否可用并缓存结果。
- **探测域**：`label` `notification` `pm` `wiki` `pipeline` `webhook` `member` `milestone` `export` `search` `workflow`
- **参数**：无显式参数；需要 owner/repo 上下文的域会自动从 `git remote` 推断，或用全局 `--owner/--repo` 指定。
- **输出**：表格列出每个模块的状态（可用 ✓ / 不可用 ✗ / 错误 ✗ / 未知 ?）与说明。
- **示例**：
  - `gitlink-cli capability +check`
  - `gitlink-cli capability +check --owner Gitlink --repo gitlink-cli`

### capability +list — 查看缓存结果
读取上一次 `+check` 的缓存结果，不发起网络请求。
- **示例**：
  - `gitlink-cli capability +list`
- 缓存过期（>24h）会提示运行 `capability +check` 刷新。

## 与其它模块的关系

`register.go` 的 `annotatedDesc` 会在各命令组描述后追加状态标记：
- 可用 → `✓`；不可用/错误 → `⚠`。
因此 `gitlink-cli --help` 里带 ⚠ 的模块即表示后端暂不支持，调用前可先用 `capability +check` 确认。
