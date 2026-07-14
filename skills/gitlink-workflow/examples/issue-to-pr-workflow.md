# Issue 到 PR 端到端工作流

**场景**：从创建 Issue 开始，到代码合并、Issue 关闭的完整自动化流程。适用于比赛子赛题三：端到端工作流。

> **比赛要求**：链式调用 ≥3 个 CLI 命令/Skills，实现端到端自动化。

## 前置条件

- 已配置 `gitlink-cli` 并登录 GitLink 平台
- 有目标仓库的写入权限
- 本地 Git 环境已配置

## 工作流步骤

### Step 1：创建 Issue

创建一个新 Issue，描述需要解决的问题。

```bash
gitlink-cli issue +create \
  --owner <owner> \
  --repo <repo> \
  --title "修复登录页面样式问题" \
  --body "## 问题描述\n登录按钮在移动端显示不完整\n\n## 期望结果\n按钮应完整显示并居中" \
  --format json
```

**预期输出：**
```json
{
  "id": 123,
  "subject": "修复登录页面样式问题",
  "project": {...},
  "status": "open"
}
```

**记录 Issue ID**：`123`（后续步骤使用）

---

### Step 2：创建开发分支

基于主分支创建新的开发分支。

```bash
gitlink-cli branch +create \
  --owner <owner> \
  --repo <repo> \
  --name fix-login-style \
  --from main \
  --format json
```

**预期输出：**
```json
{
  "name": "fix-login-style",
  "sha": "abc123...",
  "protected": false
}
```

---

### Step 3：本地开发与提交

在本地进行代码修改并推送到远程分支。

```bash
# 切换到新分支
git checkout fix-login-style

# 进行代码修改（示例：修改 CSS 文件）
echo ".login-btn { width: 100%; text-align: center; }" >> styles/login.css

# 提交更改
git add styles/login.css
git commit -m "fix: 修复登录按钮移动端显示问题 #123"

# 推送到远程
git push origin fix-login-style
```

**注意**：提交信息中包含 `#123` 可关联 Issue。

---

### Step 4：创建 Pull Request

创建 PR 将修改合并到主分支。

```bash
gitlink-cli pr +create \
  --owner <owner> \
  --repo <repo> \
  --title "fix: 修复登录按钮移动端显示问题" \
  --body "## 变更说明\n- 修复登录按钮在移动端显示不完整的问题\n- 添加响应式样式\n\n## 关联 Issue\nCloses #123" \
  --head fix-login-style \
  --base main \
  --format json
```

**预期输出：**
```json
{
  "id": 456,
  "title": "fix: 修复登录按钮移动端显示问题",
  "state": "open",
  "head": "fix-login-style",
  "base": "main"
}
```

**记录 PR ID**：`456`

---

### Step 5：获取 PR 变更摘要

使用工作流命令获取 PR 的结构化摘要。

```bash
gitlink-cli workflow +pr-summary \
  --owner <owner> \
  --repo <repo> \
  --number 456 \
  --format markdown
```

**预期输出：**
```markdown
# PR #456 摘要

## 变更文件
- `styles/login.css` (+5/-2)

## 变更类型
- Style: 样式修复

## 建议
- 建议在不同设备上测试
- 考虑添加单元测试
```

---

### Step 6：代码审查（可选）

添加 Review 评论或批准 PR。

```bash
# 添加 Review 评论
gitlink-cli api POST /:owner/:repo/pulls/456/reviews \
  --body '{"body":"LGTM! 代码变更清晰，已验证移动端显示正常。","event":"APPROVE"}'
```

---

### Step 7：合并 PR

将 PR 合并到主分支。

```bash
gitlink-cli pr +merge \
  --owner <owner> \
  --repo <repo> \
  --id 456 \
  --method merge \
  --format json
```

**预期输出：**
```json
{
  "id": 456,
  "state": "merged",
  "merged_at": "2026-06-29T10:00:00Z"
}
```

---

### Step 8：关闭 Issue

PR 合并后，关闭关联的 Issue。

```bash
gitlink-cli issue +close \
  --owner <owner> \
  --repo <repo> \
  --id 123 \
  --format json
```

**预期输出：**
```json
{
  "id": 123,
  "status": "closed",
  "closed_at": "2026-06-29T10:05:00Z"
}
```

---

### Step 9：清理分支（可选）

删除已合并的开发分支。

```bash
gitlink-cli branch +delete \
  --owner <owner> \
  --repo <repo> \
  --name fix-login-style \
  --format json
```

---

## 工作流总结

| 步骤 | 命令 | 操作类型 |
|------|------|----------|
| 1 | `issue +create` | 创建 Issue |
| 2 | `branch +create` | 创建分支 |
| 3 | Git 本地操作 | 代码修改 |
| 4 | `pr +create` | 创建 PR |
| 5 | `workflow +pr-summary` | 获取摘要 |
| 6 | `api POST .../reviews` | 代码审查 |
| 7 | `pr +merge` | 合并 PR |
| 8 | `issue +close` | 关闭 Issue |
| 9 | `branch +delete` | 清理分支 |

**CLI 命令数量**：8 个（满足 ≥3 个的比赛要求）

---

## 自动化脚本示例

将上述步骤封装为 Shell 脚本：

```bash
#!/bin/bash
# issue-to-pr.sh - Issue 到 PR 端到端工作流脚本

OWNER=$1
REPO=$2
ISSUE_TITLE=$3
BRANCH_NAME=$4

# Step 1: 创建 Issue
ISSUE=$(gitlink-cli issue +create --owner $OWNER --repo $REPO \
  --title "$ISSUE_TITLE" --body "待补充详细描述" --format json)
ISSUE_ID=$(echo $ISSUE | jq -r '.id')
echo "Created Issue #$ISSUE_ID"

# Step 2: 创建分支
gitlink-cli branch +create --owner $OWNER --repo $REPO \
  --name $BRANCH_NAME --from main --format json
echo "Created branch: $BRANCH_NAME"

# Step 3-4: 本地开发后创建 PR
# ... (需要用户手动完成代码修改)

# Step 7: 合并 PR
# PR_ID=...
# gitlink-cli pr +merge --owner $OWNER --repo $REPO --id $PR_ID

# Step 8: 关闭 Issue
gitlink-cli issue +close --owner $OWNER --repo $REPO --id $ISSUE_ID
echo "Closed Issue #$ISSUE_ID"
```

---

## 比赛提交材料

本工作流适用于 **GitLink × CCF开源创新大赛 - 子赛题三：端到端工作流**。

提交内容：
1. 本文档：`skills/gitlink-workflow/examples/issue-to-pr-workflow.md`
2. 可选：自动化脚本 `scripts/issue-to-pr.sh`
3. 演示视频/截图：展示完整工作流执行过程

---

## 相关链接

- [比赛主页](https://www.gitlink.org.cn/competitions/track1_2026GitLinkCli)
- [gitlink-cli 仓库](https://www.gitlink.org.cn/Gitlink/gitlink-cli)
- [Issue #6: 只要创建Issue就让Agent开始干活](https://www.gitlink.org.cn/Gitlink/gitlink-cli/issues/6)
