# PR 代码审查完整工作流示例

**场景**：团队成员提交了一个 PR，需要进行代码审查。

## 前置条件

- `gitlink-cli` 已安装并登录
- 用户拥有 PR 所在仓库的读取权限

## 工作流步骤

### Step 1：获取 PR 上下文

```bash
# 查看 PR 列表，找到待审查的 PR
gitlink-cli pr +list --state open --format json

# 获取特定 PR 详情
gitlink-cli pr +view --id 42 --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "id": 42,
    "title": "feat: add user authentication module",
    "body": "实现了基于 JWT 的用户认证模块，包含登录、注册、Token 刷新功能。",
    "state": "open",
    "author": "developer_a",
    "created_at": "2026-05-18T10:30:00+08:00",
    "source_branch": "feat/auth-module",
    "target_branch": "master"
  }
}
```

### Step 2：获取变更文件

```bash
gitlink-cli pr +files --id 42 --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    { "filename": "src/auth/login.py", "status": "added", "additions": 120, "deletions": 0 },
    { "filename": "src/auth/token.py", "status": "added", "additions": 85, "deletions": 0 },
    { "filename": "src/config.py", "status": "modified", "additions": 5, "deletions": 2 },
    { "filename": "tests/test_auth.py", "status": "added", "additions": 200, "deletions": 0 },
    { "filename": "requirements.txt", "status": "modified", "additions": 3, "deletions": 0 }
  ]
}
```

### Step 3：获取 Diff 内容

```bash
gitlink-cli pr +diff --id 42 --format json
```

### Step 4：逐文件审查

对每个变更文件，分析代码质量。以下是审查结果示例：

```markdown
## PR #42 代码审查报告

### 🔴 Critical

1. **JWT Secret 硬编码** — `src/config.py:15`
   > `JWT_SECRET = "my-secret-key-123"` 硬编码在源码中，存在泄露风险。建议：
   > - 使用环境变量：`JWT_SECRET = os.getenv("JWT_SECRET")`
   > - 或使用配置文件（不提交到版本控制）

2. **SQL 注入风险** — `src/auth/login.py:42`
   > `cursor.execute(f"SELECT * FROM users WHERE username = '{username}'")` 直接拼接用户输入，存在 SQL 注入风险。建议使用参数化查询：
   > ```python
   > cursor.execute("SELECT * FROM users WHERE username = ?", (username,))
   > ```

### 🟡 Warning

1. **密码明文存储** — `src/auth/login.py:88`
   > 密码直接存储到数据库，建议使用 `bcrypt` 或 `argon2` 进行哈希处理。

2. **缺少输入验证** — `src/auth/login.py:15`
   > `login()` 函数没有对 `username` 和 `password` 进行长度和格式校验。建议：
   > ```python
   > if len(username) < 3 or len(username) > 50:
   >     raise ValueError("用户名长度应在 3-50 个字符之间")
   > ```

### 🔵 Suggestion

1. **Token 过期时间可配置** — `src/auth/token.py:30`
   > `ACCESS_TOKEN_EXPIRE_MINUTES = 30` 建议改为从环境变量读取，方便不同环境配置。

2. **测试可增加边界用例** — `tests/test_auth.py`
   > 现有测试覆盖了正常流程，建议补充：
   > - 空用户名/密码
   > - 超长输入
   > - Token 过期处理
   > - 并发登录场景

### ✅ Positive

- 完整的测试覆盖（200 行测试代码，覆盖主要功能路径）
- 清晰的模块划分（login / token 职责分离）
- 有类型注解，代码可读性好
```

### Step 5：提交 Review

```bash
# 提交整体 Review 评论
gitlink-cli api POST /Gitlink/forgeplus/pulls/42/reviews --body '{
  "body": "## PR #42 代码审查报告\n\n### 🔴 Critical\n\n1. **JWT Secret 硬编码** — `src/config.py:15`\n   JWT_SECRET 硬编码在源码中。建议使用 `os.getenv(\"JWT_SECRET\")`。\n\n2. **SQL 注入风险** — `src/auth/login.py:42`\n   直接拼接用户输入到 SQL 查询。建议使用参数化查询。\n\n### 🟡 Warning\n\n1. **密码明文存储** — 建议使用 bcrypt 哈希处理。\n\n### 总体评价\n\n代码整体结构清晰，测试覆盖良好。建议修复 Critical 问题后合并。",
  "event": "COMMENT"
}'
```

### Step 6：输出审查摘要

```markdown
## 📋 审查摘要 — PR #42 feat: add user authentication module

| 指标 | 数据 |
|------|------|
| 审查文件数 | 5 |
| 变更行数 | +413 / -2 |
| Critical 问题 | 2 |
| Warning | 2 |
| Suggestion | 2 |

### 主要发现
1. **[Critical]** JWT Secret 硬编码在源码中
2. **[Critical]** SQL 查询存在注入风险
3. **[Warning]** 密码明文存储

### 总体评价
代码结构良好，测试覆盖完整。修复两个安全关键问题后即可合并。
```

---

## 完整命令速览

```bash
# 获取 PR 详情
gitlink-cli pr +view --id <id> --format json

# 获取变更文件
gitlink-cli pr +files --id <id> --format json

# 获取 Diff
gitlink-cli pr +diff --id <id> --format json

# 提交 Review
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{"body":"...","event":"COMMENT"}'
```
