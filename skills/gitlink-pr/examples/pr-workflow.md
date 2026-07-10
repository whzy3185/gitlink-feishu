# PR 管理完整工作流示例

**场景**：贡献者从 Fork 仓库到创建 PR、审查、合并的完整流程。

## 前置条件

- `gitlink-cli` 已安装并登录
- 已 Fork 目标仓库

## 工作流步骤

### Step 1：Fork 仓库并创建分支

```bash
# Fork 目标仓库
gitlink-cli repo +fork --owner TargetOrg --repo target-repo

# Clone 自己的 Fork
git clone https://www.gitlink.org.cn/MyUser/target-repo.git
cd target-repo
git remote add upstream https://www.gitlink.org.cn/TargetOrg/target-repo.git

# 创建功能分支
git checkout -b feat/new-search
# ... 编写代码 ...
git add -A && git commit -m "feat: add full-text search"
git push origin feat/new-search
```

### Step 2：创建 PR

```bash
gitlink-cli pr +create \
  --owner TargetOrg --repo target-repo \
  --head MyUser:feat/new-search --base master \
  --title "feat: add full-text search functionality" \
  --body "## 变更说明\n\n- 新增全文搜索功能\n- 支持中英文分词\n- 添加搜索结果高亮\n\n## 测试\n- [x] 单元测试通过\n- [x] 手动验证搜索功能"
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "id": 88,
    "number": 88,
    "title": "feat: add full-text search functionality",
    "state": "open",
    "source_branch": "MyUser:feat/new-search",
    "target_branch": "master"
  }
}
```

### Step 3：查看和审查 PR

```bash
# 查看 PR 详情
gitlink-cli pr +view --id 88 --format json

# 获取变更文件列表
gitlink-cli pr +files --id 88 --format json

# 获取 Diff
gitlink-cli pr +diff --id 88 --format json

# 添加 Review 评论
gitlink-cli api POST /TargetOrg/target-repo/pulls/88/reviews --body '{
  "body": "代码结构清晰，测试覆盖完整。建议合并。",
  "event": "APPROVE"
}'
```

### Step 4：合并 PR

```bash
# 查看开放的 PR 列表
gitlink-cli pr +list --state open --format json

# 合并 PR（使用 squash 方式）
gitlink-cli pr +merge --id 88 --method squash
```

---

## 完整命令速览

```bash
gitlink-cli pr +list --state open --format json
gitlink-cli pr +view --id <id> --format json
gitlink-cli pr +files --id <id> --format json
gitlink-cli pr +diff --id <id> --format json
gitlink-cli pr +create --owner <owner> --repo <repo> --head <user>:<branch> --base master --title "..."
gitlink-cli pr +merge --id <id> --method merge
gitlink-cli api POST /:owner/:repo/pulls/:id/reviews --body '{"body":"...","event":"COMMENT"}'
```
