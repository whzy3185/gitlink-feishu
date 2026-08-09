# Release 管理完整工作流示例

**场景**：项目维护者需要创建新版本发布，包含自动生成的 Release Notes。

## 前置条件

- `gitlink-cli` 已安装并登录
- 项目有已合并的 PR 和关闭的 Issue

## 工作流步骤

### Step 1：查看当前版本

```bash
# 列出已有的 Release
gitlink-cli release +list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "releases": [
      {
        "id": 15,
        "tag_name": "v1.2.0",
        "name": "v1.2.0 - 性能优化版",
        "body": "## ✨ 新功能\n...",
        "created_at": "2026-05-01"
      }
    ]
  }
}
```

### Step 2：查看提交历史和变更

```bash
# 查看自上次发版以来的提交
gitlink-cli repo +commits --owner myorg --repo myproject --limit 30 --format json

# 查看已合并的 PR
gitlink-cli pr +list --state merged --format json

# 查看已关闭的 Issue
gitlink-cli issue +list --state closed --format json
```

### Step 3：创建 Release

```bash
gitlink-cli release +create \
  --owner myorg --repo myproject \
  --tag v1.3.0 \
  --name "v1.3.0 - 用户体验升级" \
  --body "## v1.3.0 (2026-06-01)

### ✨ 新功能
- feat: 新增搜索历史功能 (#156)
- feat: 支持暗黑模式 (#178)

### 🐛 Bug 修复
- fix: 修复登录超时问题 (#165)
- fix: 修复文件上传进度显示错误 (#172)

### 🤝 贡献者
感谢 @zhangsan @lisi 的贡献！" \
  --target master
```

### Step 4：验证和通知

```bash
# 验证发布成功
gitlink-cli release +list --owner myorg --repo myproject --format json

# 获取 version_id 用于查看详情
gitlink-cli release +view --owner myorg --repo myproject --id <version_id>

# 通知相关 Issue
gitlink-cli issue +comment --number 165 --body "🎉 此问题已在 v1.3.0 中修复，请更新验证。"
gitlink-cli issue +comment --number 172 --body "🎉 此问题已在 v1.3.0 中修复，请更新验证。"
```

---

## 完整命令速览

```bash
gitlink-cli release +list --owner <owner> --repo <repo> --format json
gitlink-cli release +view --owner <owner> --repo <repo> --id <version_id>
gitlink-cli release +create --tag v1.3.0 --name "..." --body "..." --target master
gitlink-cli release +delete --id <version_id>
```
