# CI 构建管理完整工作流示例

**场景**：开发者需要查看 CI 构建状态、重启失败的构建。

## 工作流步骤

### Step 1：查看构建列表

```bash
# 列出最近的构建
gitlink-ci build list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    {
      "build_number": 156,
      "status": "success",
      "branch": "master",
      "trigger": "push",
      "duration": "3m 42s",
      "created_at": "2026-05-28T15:30:00+08:00"
    },
    {
      "build_number": 155,
      "status": "failure",
      "branch": "feature/new-api",
      "trigger": "push",
      "duration": "5m 10s"
    }
  ]
}
```

### Step 2：查看构建详情

```bash
# 查看特定构建的详细信息
gitlink-cli ci +builds --owner myorg --repo myproject --format json

# 查看构建日志（失败构建）
gitlink-cli ci +log --build 155
```

### Step 3：重启构建

```bash
# 重启失败的构建
gitlink-cli ci +restart --build 155
```

---

## 完整命令速览

```bash
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli ci +log --build <n>
gitlink-cli ci +restart --build <n>
```
