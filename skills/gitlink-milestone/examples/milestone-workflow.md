# 里程碑管理完整工作流示例

**场景**：项目维护者需要创建里程碑来规划版本迭代。

## 工作流步骤

### Step 1：查看里程碑列表

```bash
gitlink-cli milestone +list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    {
      "id": 5,
      "name": "v1.3.0",
      "description": "用户体验升级",
      "due_date": "2026-07-01",
      "open_issues": 3,
      "closed_issues": 12
    }
  ]
}
```

### Step 2：创建里程碑

```bash
gitlink-cli milestone +create \
  --owner myorg --repo myproject \
  --name "v1.4.0" \
  --description "性能优化与稳定性提升" \
  --due-date "2026-08-01"
```

### Step 3：关联 Issue 到里程碑

```bash
# 将 Issue 分配到里程碑
gitlink-cli issue +update --number 156 --milestone 6

# 批量分配
gitlink-cli issue +batch-update --ids 156,157,158 --milestone 6
```

### Step 4：更新和关闭里程碑

```bash
# 更新里程碑描述
gitlink-cli milestone +update --id 6 --description "更新后的描述"

# 版本发布后关闭里程碑
gitlink-cli milestone +close --id 5
```

---

## 完整命令速览

```bash
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
gitlink-cli milestone +create --name <name> --description "..." --due-date <date>
gitlink-cli milestone +update --id <id> --description "..."
gitlink-cli milestone +close --id <id>
```
