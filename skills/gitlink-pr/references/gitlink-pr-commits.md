# pr +commits

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

查看 Pull Request 包含的提交列表。该命令适合代码审查、commit message 质量检查、PR 门禁和 Release Notes 生成。

## 命令

```bash
# 查看 PR 提交列表
gitlink-cli pr +commits --id 3

# 简写
gitlink-cli pr +commits -i 3

# JSON 格式
gitlink-cli pr +commits -i 3 --format json
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id, -i` | 是 | PR 序号（`pull_request_number`） |

## API

```text
GET /{owner}/{repo}/pulls/{number}/commits
```

## 注意事项

- 返回提交 SHA、提交消息、作者、提交者和时间信息。
- 在 `gitlink-gatekeeper`、`gitlink-commit-quality` 等 Skill 中优先使用此 Shortcut，不再需要 Raw API。

## 参考

- [gitlink-pr](../SKILL.md)
- [gitlink-shared](../../gitlink-shared/SKILL.md)
