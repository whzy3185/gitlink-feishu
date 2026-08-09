# Issue 批量重开 / 批量评论 shortcuts

## Summary

为 `issue` shortcut 组新增两条批量操作命令，补齐 Issue 生命周期批量运维能力：

- `issue +batch-reopen` — 按网页 Issue 编号或 CSV 文件批量重新打开已关闭的 Issue。
- `issue +batch-comment` — 按网页 Issue 编号或 CSV 文件对多个 Issue 批量追加同一段评论。

两条命令复用了 `issue +batch-close` 的基础设施（`collectIssueNumbers`、`--numbers` / `--from` / `--dry-run`、`batchCloseSummary` 汇总结构），仅替换最后的写操作，保持与批量关闭一致的使用体验。

## 命令清单

- issue +batch-reopen
- issue +batch-comment

## OpenAPI coverage

| Command | Method | Endpoint |
|---|---|---|
| `issue +batch-reopen` | PATCH | `/api/v1/{owner}/{repo}/issues/{number}.json` |
| `issue +batch-comment` | POST | `/api/v1/{owner}/{repo}/issues/{number}/journals.json` |

## 实现要点

- 输入与 `batch-close` 一致：`--numbers/-n` 接逗号分隔的网页 Issue 编号，`--from` 读 CSV（识别 `number` / `issue_number` / `project_issues_index` 列或无表头首列），二者可叠加并自动去重；编号统一校验为正整数。
- `batch-reopen`：先 `GET` 取回 Issue 当前的 `subject` / `description`，再 `PATCH /v1/{owner}/{repo}/issues/{number}` 回传原内容并把 `status_id` 设为打开状态常量 `openIssueStatusID = 1`，避免重开时丢失标题与描述。
- `batch-comment`：`--body/-b` 为必填，对应 journal 的 `notes` 字段，逐条 `POST /v1/{owner}/{repo}/issues/{number}/journals`。
- 两条命令均支持 `--dry-run`：不发起写请求，逐条返回 `planned` 计划态，便于预览影响范围。
- 结果以 `batchCloseSummary`（`repository` / `dry_run` / `total` / `succeeded` / `failed` / `results`）输出，逐条记录 `action` / `status` / `error`；存在失败时以非零错误码退出并报失败计数。

## Examples

```bash
# 批量重开指定编号的 Issue
gitlink-cli issue +batch-reopen \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 1,2,3

# 先预览，不实际改动
gitlink-cli issue +batch-reopen \
  --owner Gitlink \
  --repo forgeplus \
  --from issues.csv \
  --dry-run

# 批量给多个 Issue 追加同一段评论
gitlink-cli issue +batch-comment \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 4,5 \
  --body "已在新版本修复，请验证。"

# 输出 JSON 便于脚本处理
gitlink-cli issue +batch-comment \
  --owner Gitlink \
  --repo forgeplus \
  --numbers 4,5 \
  --body "已在新版本修复，请验证。" \
  --format json
```

## Tests

```bash
GOPROXY=https://goproxy.cn,direct go test ./shortcuts/issue/...
go vet ./...
go run . issue +batch-reopen --help
go run . issue +batch-comment --help
go run . issue +batch-reopen --owner Gitlink --repo forgeplus --numbers 1,2,3 --dry-run --format json
go run . issue +batch-comment --owner Gitlink --repo forgeplus --numbers 4,5 --body "test" --dry-run --format json
```
