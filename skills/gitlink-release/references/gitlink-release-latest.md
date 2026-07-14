# release +latest

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

获取最新发布版本。默认跳过草稿和预发布版本，返回第一个符合条件的正式发布版本。

## 命令

```bash
# 获取最新正式发布版本（跳过草稿和预发布）
gitlink-cli release +latest

# 指定仓库
gitlink-cli release +latest --owner someone --repo myrepo

# 包含预发布版本
gitlink-cli release +latest --include-prerelease

# 包含草稿
gitlink-cli release +latest --include-draft

# 输出为 JSON
gitlink-cli release +latest --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--owner` | 是* | 仓库所有者（可从 git remote 自动推断） |
| `--repo` | 是* | 仓库名称（可从 git remote 自动推断） |
| `--include-prerelease` | 否 | 是否包含预发布版本（默认 false） |
| `--include-draft` | 否 | 是否包含草稿版本（默认 false） |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |
| `--debug` | 否 | 启用调试输出 |

> *如果在 GitLink 仓库目录下执行，`--owner` 和 `--repo` 可自动推断。

## 输出示例

```json
{
  "ok": true,
  "data": {
    "id": 123,
    "tag_name": "v1.0.0",
    "name": "Version 1.0.0",
    "body": "## 更新内容\n- 新增功能 A\n- 修复 Bug B",
    "draft": false,
    "prerelease": false,
    "created_at": "2024-01-15T10:30:00Z",
    "published_at": "2024-01-15T10:30:00Z"
  }
}
```

## 注意事项

- 默认情况下，草稿和预发布版本会被跳过
- 如果没有符合条件的发布版本，会返回错误
- 返回的是第一个匹配的发布版本（API 返回的列表按时间倒序排列）

## References

- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
