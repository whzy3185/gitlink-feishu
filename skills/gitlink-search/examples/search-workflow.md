# 搜索操作完整工作流示例

**场景**：开发者需要在 GitLink 上搜索仓库、用户和 Issue。

## 工作流步骤

### Step 1：搜索仓库

```bash
gitlink-cli search +repos --keyword "machine learning" --limit 10
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    {
      "full_name": "AI-Lab/ml-framework",
      "description": "轻量级机器学习框架",
      "language": "Python",
      "forks_count": 156,
      "stars_count": 1024
    }
  ]
}
```

### Step 2：搜索用户

```bash
gitlink-cli search +users --keyword "zhangsan"
```

### Step 3：搜索 Issue

```bash
# 基本搜索
gitlink-cli search +issues --owner myorg --repo myproject --keyword "登录失败"

# 搜索已关闭的 Issue
gitlink-cli search +issues --keyword "bug" --category closed

# 按标签和负责人筛选
gitlink-cli search +issues --keyword "性能" --assignee 42 --tag 1,2

# 按时间排序
gitlink-cli search +issues --keyword "需求" --sort-by created_on --sort-dir asc
```

---

## 完整命令速览

```bash
gitlink-cli search +repos --keyword "<keyword>" --limit <n>
gitlink-cli search +users --keyword "<username>"
gitlink-cli search +issues --owner <owner> --repo <repo> --keyword "<keyword>" --category opened
```
