# release +update

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

更新发行版。该命令会先读取 `release +edit` 对应接口返回的当前值，再合并用户传入的字段，避免未传字段被清空。

## 命令

```bash
# 更新发布说明，先预览
gitlink-cli release +update --id 12345 --body "Updated changelog" --dry-run

# 确认后提交
gitlink-cli release +update --id 12345 --body "Updated changelog"

# 更新标题、tag、目标分支和附件
gitlink-cli release +update --id 12345 \
  --name "v1.1.0" --tag v1.1.0 --target main --attachment-ids 12,34

# 切换草稿/预发布状态
gitlink-cli release +update --id 12345 --draft false --prerelease true --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id, -i` | 是 | 发行版 ID（version_id） |
| `--tag, -t` | 否 | Tag 名称 |
| `--name, -n` | 否 | 发行版名称 |
| `--body, -b` | 否 | 发布说明 |
| `--target` | 否 | 目标分支 |
| `--draft` | 否 | 是否为草稿（`true`/`false`） |
| `--prerelease` | 否 | 是否为预发布（`true`/`false`） |
| `--attachment-ids` | 否 | 逗号分隔的附件 ID |
| `--dry-run` | 否 | 仅预览更新请求，不写入 |

## Workflow

> [!CAUTION]
> This is a **Write Operation** -- confirm user intent.

1. 先用 `release +list` 确认 `version_id`。
2. 执行 `release +update ... --dry-run` 预览请求。
3. 用户确认后移除 `--dry-run` 执行更新。

## References
- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
