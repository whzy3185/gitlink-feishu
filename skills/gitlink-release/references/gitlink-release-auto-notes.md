# release +auto-notes

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

自动生成 Release Notes。根据提交信息和已关闭的 Issue 自动生成格式化的发布说明。

## 命令

```bash
# 自动生成 Release Notes（基于最近的提交）
gitlink-cli release +auto-notes --to-tag v2.0.0

# 指定起始标签（比较两个标签之间的变更）
gitlink-cli release +auto-notes --from-tag v1.0.0 --to-tag v2.0.0

# 指定仓库
gitlink-cli release +auto-notes --to-tag v2.0.0 --owner someone --repo myrepo

# 输出为 JSON（包含统计信息）
gitlink-cli release +auto-notes --to-tag v2.0.0 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--to-tag, -t` | 是 | 目标标签（新版本标签） |
| `--from-tag, -f` | 否 | 起始标签（旧版本标签，不指定则使用最近 50 条提交） |
| `--owner` | 是* | 仓库所有者（可从 git remote 自动推断） |
| `--repo` | 是* | 仓库名称（可从 git remote 自动推断） |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |
| `--debug` | 否 | 启用调试输出 |

> *如果在 GitLink 仓库目录下执行，`--owner` 和 `--repo` 可自动推断。

## 输出示例

### 默认格式（纯文本）

```markdown
# Release Notes

## 🚀 新功能

- feat: 添加用户认证功能
- feat: 支持批量导入

## 🐛 Bug 修复

- fix: 修复登录超时问题
- fix: 解决文件上传失败

## 📝 其他变更

- docs: 更新 API 文档
- chore: 优化构建流程

## 🔗 相关 Issue

- #123 用户登录失败
- #456 文件上传异常
```

### JSON 格式

```json
{
  "ok": true,
  "data": {
    "release_notes": "# Release Notes\n\n## 🚀 新功能\n...",
    "commits_count": 15,
    "issues_count": 2
  }
}
```

## 分类规则

提交信息根据前缀自动分类：

| 前缀 | 分类 |
|------|------|
| `feat:` | 🚀 新功能 |
| `fix:` | 🐛 Bug 修复 |
| 其他 | 📝 其他变更 |

## 注意事项

- 如果不指定 `--from-tag`，会获取最近 50 条提交
- 已关闭的 Issue 会被包含在 Release Notes 的"相关 Issue"部分
- 生成的 Release Notes 可以直接用于 `release +create` 的 `--body` 参数

## References

- [gitlink-release](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
