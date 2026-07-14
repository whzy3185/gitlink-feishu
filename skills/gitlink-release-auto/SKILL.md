---
name: gitlink-release-auto
version: 1.0.0
description: "自动化 Release 管理：从提交历史自动生成 Release Notes、推荐语义化版本号、批量发布管理。当用户需要创建版本发布、生成更新日志、自动化发版流程时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli release --help"
---

# gitlink-release-auto（自动化 Release 管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — `release +view` 必须使用 `version_id`（从 `release +list` 返回），不能用 tag_name，否则返回 HTML 页面而非 JSON。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

---

## 功能概述

本技能提供完整的自动化发版流程：

1. **语义化版本推荐** — 根据提交类型自动推荐下一个版本号
2. **Release Notes 自动生成** — 从提交历史和已关闭 Issue 自动生成 Changelog
3. **一键发布** — 创建 Release 并关联 Tag
4. **发布后通知** — 在相关 Issue 中关联发版信息

---

## 一、版本号推荐（Semantic Versioning）

### Semver 规则

```
v<MAJOR>.<MINOR>.<PATCH>

MAJOR：有 Breaking Change（不向下兼容的变更）
MINOR：有 feat（新增功能，向下兼容）
PATCH：只有 fix / perf / refactor / docs 等修复性变更
```

### 步骤 1：获取当前版本号

```bash
# 获取最新 Release（从中找当前版本号）
gitlink-cli release +list --owner <owner> --repo <repo> --format json
# 取 data.releases[0].tag_name 作为当前版本

# 获取标签列表
gitlink-cli repo +tags --format json
```

### 步骤 2：获取自上次发版以来的提交

```bash
# 获取自上次 Release 以来的提交
git log v1.1.0..HEAD --format="%H %s %an %ad" --date=short 2>&1
```

### 步骤 3：AI 分析提交类型，推荐版本号

根据提交信息（Conventional Commits）进行分类：

```
决策逻辑：
1. 任意提交包含 "BREAKING CHANGE" → MAJOR 升级
   当前 v1.2.3 → 推荐 v2.0.0

2. 有 feat: 类型的提交（且无 Breaking Change）→ MINOR 升级
   当前 v1.2.3 → 推荐 v1.3.0

3. 仅有 fix/perf/refactor/docs/chore/ci 类型 → PATCH 升级
   当前 v1.2.3 → 推荐 v1.2.4
```

**版本号推荐输出示例：**

```
当前版本：v1.2.3
分析了 18 次提交（2026-04-15 至今）：
  - 3 个 feat：新增用户头像、搜索历史、消息通知
  - 5 个 fix：修复了 5 个 Bug
  - 2 个 docs：更新了接口文档
  - 无 Breaking Change

推荐版本：v1.3.0（MINOR 升级，有新功能）
备选版本：v1.2.4（如果认为功能较小，可用 PATCH）
```

---

## 二、Release Notes 自动生成

### 获取数据

```bash
# 1. 获取自上次 Release 以来的提交
git log v1.1.0..HEAD --format="%H %s %an %ad" --date=short 2>&1

# 2. 获取本期关闭的 Issue
gitlink-cli issue +list --state closed --owner <owner> --repo <repo> --format json

# 3. 获取本期合并的 PR 列表
gitlink-cli pr +list --state merged --owner <owner> --repo <repo> --format json
```

### Release Notes 模板

```markdown
## v1.3.0 (2026-05-07)

### ✨ 新功能

- feat(user): 新增用户头像上传功能 ([#234](link))
- feat(search): 支持全文搜索和历史记录 ([#256](link))
- feat(notify): 添加站内消息通知系统 ([#267](link))

### 🐛 Bug 修复

- fix(login): 修复手机号登录时验证码未清除的问题 ([#245](link))
- fix(upload): 修复大文件上传超时问题 ([#251](link))
- fix(api): 修复并发请求时偶发 500 错误 ([#259](link))

### ⚡ 性能优化

- perf(search): 优化搜索接口响应速度，平均提升 40% ([#261](link))

### 📦 其他变更

- docs: 更新 API 接口文档
- build: 升级 Go 依赖到最新版本
- ci: 添加代码覆盖率检查


### 🤝 贡献者

感谢以下贡献者参与本版本开发：@zhangsan、@lisi、@wangwu

```

---

> ⚠️ **重要**：`贡献者` 必须去重！release notes中如果有相同或者相似的记录，需要去重！不需要添加`完整变更日志`这个信息！


## 三、一键发布完整流程

### 标准发版步骤

```bash
# Step 1：确认当前版本和推荐版本
gitlink-cli release +list --owner <owner> --repo <repo> --format json

# Step 2：获取变更内容（提交历史）
gitlink-cli repo +commits --query 'page=1&limit=30&ref=master' --format json

# Step 3：生成 Release Notes（AI 分析提交后组织内容）

# Step 4：创建 Release
gitlink-cli release +create \
  --owner <owner> \
  --repo <repo> \
  --tag v1.3.0 \
  --name "v1.3.0 - 用户体验升级版" \
  --body "## v1.3.0\n\n### ✨ 新功能\n- ...\n\n### 🐛 Bug 修复\n- ..." \
  --target master

# Step 5：验证发布成功
gitlink-cli release +list --owner <owner> --repo <repo> --format json
# 从列表取 version_id

gitlink-cli release +view --owner <owner> --repo <repo> --id <version_id>
```

> ⚠️ **重要**：`release +view` 必须用 `version_id`（整数），不能用 tag 名称（如 v1.3.0）

### 预发布版本

```bash
# 创建 alpha/beta/rc 预发布版本
gitlink-cli release +create \
  --tag v1.3.0-beta.1 \
  --name "v1.3.0 Beta 1" \
  --body "预发布版本，用于测试..." \
  --prerelease true \
  --target develop
```

---

## 四、发布后操作

### 关联 Issue 通知

```bash
# 在关联的 Issue 中添加评论，通知已发版
gitlink-cli issue +comment \
  --id <issue_id> \
  --body "🎉 此问题已在 v1.3.0 中修复，请更新到最新版本验证。"
```

### 批量关联 Issue

```bash
# 遍历本期关闭的 Issue，逐一添加发版通知评论
gitlink-cli issue +list --state closed --format json
# 对每个 issue_id 执行 issue +comment
```

### 触发部署（可选）

```bash
# 发版后触发部署（通过 CI 重新构建 tag 对应的分支）
gitlink-cli ci +restart --build <latest_build_number>
```

---

## 五、完整发版检查清单

在执行发版前，确认以下事项：

```
发版前检查：
□ 所有计划纳入本次版本的 PR 已合并
□ CI 在 master 分支的最新构建是成功的
□ 所有计划修复的 Issue 已关闭
□ 文档已更新（README、API 文档等）
□ 数据库迁移脚本已准备就绪（如有）

版本号确认：
□ 版本号遵循 Semantic Versioning
□ 是否有 Breaking Change（需 MAJOR 升级）
□ 是否发预发布版本（alpha/beta/rc）

Release Notes 检查：
□ 新功能描述清晰
□ Bug 修复有具体说明
□ Breaking Change 有迁移指南
□ 关联了对应的 Issue / PR 链接

发版操作：
□ 在正确的分支/提交上打 Tag
□ Release 创建成功
□ 相关 Issue 已通知
```

---

## 六、版本管理最佳实践

### 发版节奏建议

| 模式 | 说明 | 适用场景 |
|------|------|---------|
| 定期发版 | 每 1~2 周一次 PATCH，每月一次 MINOR | 功能迭代稳定的项目 |
| 功能发版 | 功能完成即发版 | 需求驱动、快速迭代 |
| 按需发版 | Bug 修复立即发版 | 线上紧急修复（hotfix） |

### Hotfix 发版流程

```bash
# 1. 从当前 Release Tag 创建 hotfix 分支
gitlink-cli branch +create --name hotfix/v1.2.4-critical-fix --from v1.2.3

# 2. 在 hotfix 分支修复问题，合并回 master
# （通过 PR 流程）

# 3. 快速发布 PATCH 版本
gitlink-cli release +create \
  --tag v1.2.4 \
  --name "v1.2.4 紧急修复版" \
  --body "## 紧急修复\n\n- fix: 修复生产环境关键 Bug（#xxx）" \
  --target master
```

---

## 注意事项

- ✅ **版本号一旦发布不可修改**，确认无误后再执行创建
- ✅ **BREAKING CHANGE 必须在 Release Notes 中明确标注**，并提供迁移指南
- ⚠️ **`release +view` 始终使用 `version_id`**
- ✅ **预发布版本（beta/rc）先内部测试**，验证后再发正式版
- ✅ **发版后通知**，注意发版后，要对所有关联的issue添加通知评论
