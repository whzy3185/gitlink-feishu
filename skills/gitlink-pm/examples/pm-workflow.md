# 项目管理（PM）完整工作流示例

**场景**：项目经理需要查看看板、Sprint、周报等项目管理信息。

## 工作流步骤

### Step 1：查看看板

```bash
# 查看项目看板（自动从 git remote 解析）
gitlink-cli pm +boards
```

### Step 2：查看 Sprint Issue 列表

```bash
# 查看 Sprint 中的 Issue
gitlink-cli pm +sprints
```

### Step 3：查看周报

```bash
# 生成本周周报
gitlink-cli pm +weekly
```

### Step 4：查看 PM Issue 标签和流水线

```bash
# 查看 PM Issue 标签
gitlink-cli pm +tags

# 查看 PM 流水线
gitlink-cli pm +pipelines
```

### Step 5：查看 Action 运行记录

```bash
# 查看最近的 Action 运行记录
gitlink-cli pm +actions --page 1 --limit 10

# 翻页查看更多
gitlink-cli pm +actions --page 2 --limit 10
```

### Step 6：通过 Raw API 灵活查询

```bash
# 按项目 ID 查看看板
gitlink-cli api GET /pm/dashboards --query 'project_id=123'

# 查询 Sprint Issue
gitlink-cli api GET /pm/sprint_issues --query 'project_id=123'

# 查询周报
gitlink-cli api GET /pm/weekly_issues --query 'project_id=123'
```

---

## 完整命令速览

```bash
gitlink-cli pm +boards
gitlink-cli pm +sprints
gitlink-cli pm +weekly
gitlink-cli pm +tags
gitlink-cli pm +pipelines
gitlink-cli pm +actions --page <n> --limit <n>
```
