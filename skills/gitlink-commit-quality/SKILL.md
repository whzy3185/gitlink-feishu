---
name: gitlink-commit-quality
version: 1.0.0
description: "提交质量守护：检查提交信息规范（Conventional Commits）、PR 描述完整性、分支命名规范、变更文件合理性审核。当用户需要规范团队提交流程、审查代码提交质量时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-commit-quality（提交质量守护）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能概述

本技能覆盖开发流程中的提交质量管控，包括：

1. **Commit Message 规范检查** — 验证是否符合 Conventional Commits 规范
2. **PR 质量检查** — PR 标题、描述、关联 Issue 完整性
3. **分支命名规范** — 检查分支名是否符合团队约定
4. **变更合理性检查** — 一次提交是否改动过多、是否有无关文件混入

---

## 一、Commit Message 规范检查

### Conventional Commits 规范

标准格式：
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**合法的 type 值：**

| type | 含义 | 示例 |
|------|------|------|
| `feat` | 新增功能 | `feat(auth): 添加 JWT 认证` |
| `fix` | 修复 Bug | `fix(login): 修复密码验证逻辑错误` |
| `docs` | 文档变更 | `docs: 更新 API 接口文档` |
| `style` | 代码格式（不影响逻辑） | `style: 统一缩进为 4 空格` |
| `refactor` | 代码重构 | `refactor(user): 提取公共验证逻辑` |
| `perf` | 性能优化 | `perf(query): 优化数据库查询索引` |
| `test` | 测试相关 | `test(auth): 补充登录测试用例` |
| `build` | 构建系统变更 | `build: 升级 Go 版本到 1.21` |
| `ci` | CI 配置变更 | `ci: 添加 lint 检查步骤` |
| `chore` | 其他杂项 | `chore: 更新 .gitignore` |
| `revert` | 回滚提交 | `revert: revert feat(auth): ...` |

### 检查 PR 的所有提交信息

```bash
# 获取 PR 关联的提交列表
gitlink-cli pr +diff --id <pr_id> --owner <owner> --repo <repo> --format json

# 或通过 Raw API 获取提交详情
gitlink-cli api GET /:owner/:repo/pulls/:pr_id/commits --format json
```

### Commit Message 质量检查清单

对每条提交信息执行以下检查：

```
✅ 必须检查：
□ type 是否是合法值（feat/fix/docs/style/refactor/perf/test/build/ci/chore/revert）
□ description 是否是中文或英文（不能是无意义内容如 "update"、"fix"、"tmp"、"xxx"）
□ description 首字母是否小写（英文时）
□ description 末尾是否没有句号
□ 整条信息是否超过 72 个字符（首行）

⚠️ 建议检查：
□ scope 是否有意义（如 auth/user/api/db 等模块名）
□ Breaking Change 是否在 footer 中标注（BREAKING CHANGE: ...）
□ 关联 Issue 是否在 footer 中标注（Closes #123 / Fixes #456）
```

### 不合规示例 vs 合规示例

| 不合规 | 问题 | 合规写法 |
|--------|------|---------|
| `update` | 无意义描述 | `feat(user): 新增用户头像上传功能` |
| `fix bug` | 无具体说明 | `fix(login): 修复手机号登录时验证码未清除的问题` |
| `WIP` | 临时提交混入 | 应在本地整理后再提交 |
| `Feat: Add search` | type 大写 | `feat(search): 添加全文搜索功能` |
| `feat: 添加搜索.` | 末尾有句号 | `feat(search): 添加搜索功能` |

---

## 二、PR 质量检查

### 获取 PR 信息

```bash
# 获取 PR 详情
gitlink-cli pr +view --id <pr_id> --owner <owner> --repo <repo> --format json
```

### PR 质量检查清单

```
✅ 必须检查：
□ PR 标题是否符合 Conventional Commits 格式
□ PR 描述是否有实质内容（不能为空或仅有一行）
□ PR 描述是否说明了"做了什么"和"为什么这么做"
□ PR 变更文件数量是否合理（建议单次 PR < 20 文件）
□ PR 变更行数是否合理（建议单次 PR < 500 行净变更）

⚠️ 建议检查：
□ PR 描述是否关联了对应 Issue（格式：Closes #123）
□ 是否有 Test Plan（如何验证这次改动）
□ 是否有截图（UI 改动）
□ 是否更新了相关文档
```

### PR 描述模板（推荐）

在 Review 评论中可以建议作者按以下模板完善 PR 描述：

```markdown
## 变更说明

<!-- 简述本次 PR 做了什么 -->

## 原因/背景

<!-- 为什么需要这个改动？关联的 Issue？ -->
Closes #<issue_number>

## 测试方案

<!-- 如何验证这次改动是正确的？ -->
- [ ] 单元测试通过
- [ ] 手动测试步骤：...

## 截图（如有 UI 变更）

<!-- 添加 before/after 截图 -->

## Checklist

- [ ] 代码符合项目编码规范
- [ ] 相关文档已更新
- [ ] 测试覆盖新增/修改的代码
```

---

## 三、分支命名规范检查

### 获取分支信息

```bash
# 从 PR 信息中获取分支名
gitlink-cli pr +view --id <pr_id> --format json
# 关注 head_branch 字段

# 列出所有分支
gitlink-cli branch +list --owner <owner> --repo <repo> --format json
```

### 推荐的分支命名规范

```
格式：<type>/<short-description>
或：  <type>/<issue-id>-<short-description>

示例：
  feat/user-authentication
  fix/login-password-validation
  fix/123-token-expiry-bug
  docs/api-reference-update
  refactor/extract-auth-middleware
  release/v1.2.0
  hotfix/critical-security-patch
```

### 不合规的分支名示例

| 不合规 | 问题 | 建议 |
|--------|------|------|
| `test123` | 无意义 | `test/add-login-unit-tests` |
| `dev` / `development` | 过于宽泛 | 使用具体功能描述 |
| `zhangsan_feature` | 用人名命名 | 用功能命名 |
| `fix-bug` | 不够具体 | `fix/login-crash-on-empty-password` |
| 含中文 | 可能导致编码问题 | 使用英文 |

---

## 四、变更合理性检查

### 获取变更文件

```bash
gitlink-cli pr +files --id <pr_id> --owner <owner> --repo <repo> --format json
```

### 变更文件审核要点

```
📊 规模检查：
□ 变更文件数 > 20：建议拆分成多个 PR
□ 净变更行数 > 500：建议拆分
□ 单个文件变更 > 300 行：值得重点关注

🔍 文件类型检查：
□ 是否有无关文件混入（如调试文件、个人 IDE 配置）
□ 是否有 .env 文件、密钥文件被提交
□ 是否有二进制文件或大文件（图片、视频等）
□ 是否有自动生成文件（lock 文件、build 产物）混入功能 PR

⚠️ 敏感文件检查（高优先级）：
□ .env / .env.local / .env.production
□ *.pem / *.key / *.p12 / *.pfx
□ id_rsa / id_dsa
□ config/secret* / credentials*
□ *password* / *secret* / *token*（文件名）
```

---

## 五、完整质量报告输出格式

```markdown
## 📋 提交质量检查报告

**检查范围：** PR #42 — feat: 添加用户认证模块

---

### 1. PR 基本信息

| 项目 | 检查结果 |
|------|---------|
| PR 标题格式 | ✅ 符合规范 |
| PR 描述完整性 | ⚠️ 缺少 Test Plan |
| 关联 Issue | ✅ Closes #88 |
| 变更规模 | ✅ 12 文件 / +248 -89 行 |

---

### 2. 提交信息检查

| 提交 SHA | 提交信息 | 检查结果 |
|---------|---------|---------|
| `a1b2c3d` | feat(auth): 添加 JWT 生成 | ✅ 合规 |
| `e4f5g6h` | fix login bug | ❌ 不合规 |
| `i7j8k9l` | WIP | ❌ 不合规 |

**问题详情：**

**[1] 提交 e4f5g6h: "fix login bug"**
- 问题：缺少规范 type 格式，描述过于模糊
- 建议：`fix(login): 修复登录时密码验证逻辑错误`

**[2] 提交 i7j8k9l: "WIP"**
- 问题：临时 WIP 提交不应出现在正式 PR 中
- 建议：`git rebase -i HEAD~2` 将相关提交合并，并写一个规范的提交信息

---

### 3. 分支命名

| 分支名 | 检查结果 |
|-------|---------|
| `feature/user-auth` | ✅ 符合规范 |

---

### 4. 变更文件审核

✅ 规模合理（12 文件，248 行净增加）

⚠️ **发现 1 个潜在问题：**
- `config/app.env`：`.env` 文件被提交，请确认是否包含敏感配置

---

### 5. 改进建议

1. 完善 PR 描述，添加 Test Plan（必须）
2. 修复 2 条不规范的提交信息（建议通过 rebase 整理）
3. 确认 `.env` 文件是否应该提交，建议加入 `.gitignore`

---

**总体评级：** ⚠️ 需要改进（2个必须修复，1个建议）
```

---

## 六、配合 CI 的自动化质量门禁

可将以下检查逻辑集成到 CI 流程，作为 PR 合并前的质量门禁：

```bash
# 在 CI 中检查最新提交的 commit message
git log --oneline -10

# 统计 PR 变更规模
gitlink-cli pr +files --id $PR_ID --format json | \
  jq '{files: [.data[].filename] | length, additions: [.data[].additions] | add, deletions: [.data[].deletions] | add}'
```

---

## 注意事项

- ✅ **提交规范的推行应循序渐进**，先建议后强制
- ✅ **WIP 提交**：允许在开发分支中存在，但合并到主干前需 squash/rebase
- ⚠️ **历史遗留**：老代码的提交不在检查范围，只检查本次 PR 新增的提交
- ✅ **团队约定优先**：如团队有自己的规范，以团队规范为准，本技能提供通用参考
