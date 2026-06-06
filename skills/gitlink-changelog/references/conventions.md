# Conventional Commits 归类规则

本技能按 [Conventional Commits](https://www.conventionalcommits.org/) 规范解析提交。

## 提交格式

```
<type>(<scope>)<!>: <description>
```

- `type`：变更类型（见下表）
- `scope`：可选，影响范围（如 `auth`、`issue`）
- `!`：可选，标记不兼容变更
- `description`：变更描述

## 识别的类型

| type | 含义 | 报告分组 |
|------|------|----------|
| feat | 新功能 | ✨ 新功能 |
| fix | 缺陷修复 | 🐛 缺陷修复 |
| perf | 性能优化 | ⚡ 性能优化 |
| refactor | 重构 | ♻️ 重构 |
| docs | 文档 | 📝 文档 |
| test | 测试 | ✅ 测试 |
| build | 构建 | 📦 构建 |
| ci | 持续集成 | 👷 持续集成 |
| style | 代码风格 | 💄 风格 |
| chore | 工程杂项 | 🔧 工程 |
| revert | 回退 | ⏪ 回退 |

不匹配上述类型的提交归为 `other`，不计入分组（但仍计入总数）。

## 不兼容变更（BREAKING CHANGE）

满足任一即标记为不兼容变更，单列在报告顶部 ⚠️ 区块：

- 类型后带 `!`，如 `feat!:` 或 `feat(api)!:`
- 提交正文包含 `BREAKING CHANGE`

## 报告分组顺序

feat → fix → perf → refactor → docs → test → build → ci → style → chore → revert

每组最多展示 30 条，超出省略。
