---
name: gitlink-release-notes
version: 1.0.0
description: "Release Notes 生成：根据 commit 和 PR 记录生成结构化版本说明。当用户需要生成版本发布说明、制作 Release Notes、总结版本变更时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli release --help"
---

# gitlink-release-notes（Release Notes 生成）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读操作，不会修改任何仓库。无需用户额外确认即可执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

根据仓库的 PR 记录，自动生成结构化的 Release Notes：

1. **获取版本信息** — 查看已有 Release 确定版本号
2. **收集 PR 变更** — 获取合并的 PR 记录
3. **自动分类** — 将变更归类为 Feature/BugFix/Refactor/Docs
4. **生成发布说明** — 按标准模板输出 Release Notes

---

## 工作流：Release Notes 生成

### Step 1：获取仓库信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

提取：`full_name`, `version_releases_count`。

### Step 2：获取已有 Release

```bash
gitlink-cli release +list --owner <owner> --repo <repo> --format json
```

查看发布历史，确定下一版本号。

### Step 3：获取已合并的 PR

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --format json
```

筛选 `pull_request_status=1` (已合并) 的 PR。

### Step 4：分类汇总

| 类型 | 标题关键词 | 说明 |
|------|-----------|------|
| 🚀 Features | `feat:` | 新功能 |
| 🐛 Bug Fixes | `fix:` | Bug 修复 |
| 🔧 Refactoring | `refactor:`, `perf:`, `chore:` | 重构和性能优化 |
| 📝 Documentation | `docs:` | 文档更新 |
| ⚡ CI/CD | `ci:`, `build:` | CI/CD 流水线 |
| 🧪 Tests | `test:` | 测试相关 |

### Step 5：生成 Release Notes

---

## 输出模板

```markdown
# 🚀 Release {{version}} — {{release_title}}

> 发布日期：{{release_date}}
> 仓库：[{{full_name}}](https://www.gitlink.org.cn/{{owner}}/{{repo}})
> 此版本包含 {{total_prs}} 个合并 PR，由 {{total_contributors}} 位贡献者完成

---

## 🔢 变更统计

| 类型 | 数量 |
|------|------|
| 🚀 Features | {{feat_count}} |
| 🐛 Bug Fixes | {{fix_count}} |
| 🔧 Refactoring | {{refactor_count}} |
| 📝 Documentation | {{docs_count}} |
| ⚡ CI/CD | {{ci_count}} |
| 🧪 Tests | {{test_count}} |

---

## 🚀 Features

- **{{title}}** (#{{number}}) — {{author_login}}

> 如无，输出：此版本无新增功能。

## 🐛 Bug Fixes

- **{{title}}** (#{{number}}) — {{author_login}}

> 如无，输出：此版本无 Bug 修复。

## 🔧 Refactoring

- **{{title}}** (#{{number}}) — {{author_login}}

## 📝 Documentation

- **{{title}}** (#{{number}}) — {{author_login}}

---

## 👏 贡献者

- @{{login}}（{{pr_count}} 个 PR）

---

## 📦 完整 PR 列表

| PR | 标题 | 类型 | 作者 |
|----|------|------|------|
| #{{number}} | {{title}} | {{type}} | {{author_login}} |

> 此 Release Notes 由 `gitlink-cli` 自动生成
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 无已合并 PR | 报告"新版本尚无合并的 PR" |
| PR 标题无类型前缀 | 归入"Other"分类 |
| PR 数量过多（>50） | 仅列出最近 30 个 PR |

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **本 Skill 为纯只读操作**，不会修改仓库
- ✅ **Owner/repo 优先从 `git remote` 自动解析**
- ⚠️ **PR 标题需遵循 Conventional Commits 规范**，否则分类可能不准确
- ⚠️ **GitLink Release 功能使用率低**，`release +list` 可能返回空
