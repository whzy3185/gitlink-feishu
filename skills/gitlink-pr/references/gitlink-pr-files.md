# pr +files / pr +diff

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看 Pull Request 的变更文件列表或代码差异。

## 命令

### pr +files — 变更文件列表

返回 PR 涉及的文件列表，包含文件名、增删行数统计。

```bash
gitlink-cli pr +files --id 3
gitlink-cli pr +files -i 3 --format json
```

### pr +diff — 代码差异（逐行 diff）

返回 PR 的逐行代码差异，包含每个文件的 sections 和 lines。

```bash
# 查看完整 diff
gitlink-cli pr +diff --id 3

# 仅查看某个文件的 diff
gitlink-cli pr +diff --id 3 --file src/main.go
gitlink-cli pr +diff --id 3 -f src/main.go

# 仅查看统计摘要（文件数量、增删行数）
gitlink-cli pr +diff --id 3 --stat
```

## 参数

### pr +files

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（`pull_request_number`） |

### pr +diff

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（`pull_request_number`） |
| `--file` / `-f` | 否 | 过滤特定文件路径的 diff |
| `--stat` | 否 | 仅显示统计摘要，不显示逐行内容 |

## API

```
# +files 使用
GET /{owner}/{repo}/pulls/{number}/files

# +diff 使用（两步调用）
GET /v1/{owner}/{repo}/pulls/{number}/versions
GET /v1/{owner}/{repo}/pulls/{number}/versions/{version_id}/diff
```

## +diff 实现说明

`pr +diff` 通过两步 API 调用获取完整 diff：

1. 先调用 `versions` 端点获取 PR 版本列表，取最新版本的 ID
2. 再调用 `versions/{id}/diff` 端点获取该版本的完整 diff 数据

返回数据包含每个文件的 `sections` → `lines`，每行有：
- `type` 4: diff hunk header（如 `@@ -10,3 +10,4 @@`）
- `type` 2: 新增行
- `type` 3: 删除行

## 注意事项

- `pr +files` 和 `pr +diff` 是**不同命令**：`+files` 返回文件概览，`+diff` 返回逐行差异
- `pr +diff` 需要两次 API 调用，对大型 PR 可能稍慢
- 使用 `--stat` 可快速查看变更规模，适合大型 PR
- 适合 code review 流程：先 `pr +view` 查看概览 → `pr +diff --stat` 看规模 → `pr +diff` 看具体变更

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
