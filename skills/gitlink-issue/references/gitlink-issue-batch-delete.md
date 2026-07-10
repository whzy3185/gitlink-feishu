# issue +batch-delete

批量删除多个 Issue。这是**危险操作**，必须使用 `--confirm` 确认。

## 使用方法

```bash
# 预览删除（不实际删除）
gitlink-cli issue +batch-delete --ids 10,20,30 --dry-run

# 确认删除
gitlink-cli issue +batch-delete --ids 10,20,30 --confirm
```

## 参数

| 参数 | 短选项 | 必需 | 说明 |
|------|--------|------|------|
| `--ids` | `-i` | 是 | 逗号分隔的 Issue ID 列表 |
| `--dry-run` | | 否 | 仅预览，不实际删除 |
| `--confirm` | | 否 | 确认执行删除（必须提供此标志才会执行） |

## API 端点

`DELETE /api/v1/{owner}/{repo}/issues/batch_destroy.json`

Body: `{"ids": [10, 20, 30]}`
