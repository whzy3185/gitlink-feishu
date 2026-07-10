# Issue 管理完整工作流示例

**场景**：项目维护者需要批量管理 Issue：创建、分类、关闭、评论。

## 前置条件

- `gitlink-cli` 已安装并登录

## 工作流步骤

### Step 1：列出项目 Issue

```bash
# 查看所有开放的 Issue
gitlink-cli issue +list --owner myorg --repo myproject --state open --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    {
      "number": 42,
      "subject": "登录页面报错 500",
      "status_id": 1,
      "priority_id": 2,
      "author": "user_a",
      "created_at": "2026-05-20T10:00:00+08:00"
    }
  ]
}
```

### Step 2：创建 Issue

```bash
gitlink-cli issue +create \
  --owner myorg --repo myproject \
  --title "Bug: 搜索结果排序异常" \
  --body "## 复现步骤\n1. 打开搜索页面\n2. 输入关键词\n3. 点击搜索\n\n## 预期结果\n结果按相关度排序\n\n## 实际结果\n结果顺序随机"
```

### Step 3：批量分类 Issue

```bash
# 查看可用标签
gitlink-cli issue +tags --owner myorg --repo myproject

# 批量更新 Issue 标签（加 --dry-run 预览）
gitlink-cli issue +batch-update --ids 10,20,30 --tag-ids 1,3 --dry-run

# 确认后执行
gitlink-cli issue +batch-update --ids 10,20,30 --tag-ids 1,3
```

### Step 4：添加评论和关闭 Issue

```bash
# 为 Issue 添加评论
gitlink-cli issue +comment --number 42 --body "已修复，请更新到 v1.2.0 验证"

# 关闭单个 Issue
gitlink-cli issue +close --number 42

# 批量关闭已解决的 Issue
gitlink-cli issue +batch-close --owner myorg --repo myproject --numbers 42,43,44 --dry-run
gitlink-cli issue +batch-close --owner myorg --repo myproject --numbers 42,43,44
```

---

## 完整命令速览

```bash
gitlink-cli issue +list --state open --format json
gitlink-cli issue +create --title "..." --body "..."
gitlink-cli issue +view --number <n> --format json
gitlink-cli issue +update --number <n> --title "..." --tag-ids <ids>
gitlink-cli issue +comment --number <n> --body "..."
gitlink-cli issue +close --number <n>
gitlink-cli issue +batch-close --numbers 1,2,3
gitlink-cli issue +batch-update --ids 1,2,3 --status closed --dry-run
```
