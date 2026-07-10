# 成员管理完整工作流示例

**场景**：项目维护者需要管理仓库的协作者和团队成员。

## 工作流步骤

### Step 1：查看仓库成员

```bash
gitlink-cli member +list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    { "id": 100, "name": "张三", "login": "zhangsan", "role": "Manager" },
    { "id": 101, "name": "李四", "login": "lisi", "role": "Developer" }
  ]
}
```

### Step 2：添加成员

```bash
# 添加仓库协作者
gitlink-cli member +add --owner myorg --repo myproject --user-id 102 --role Developer
```

### Step 3：修改成员角色

```bash
# 将成员提升为 Manager
gitlink-cli member +update --owner myorg --repo myproject --user-id 101 --role Manager
```

### Step 4：移除成员

```bash
# 移除仓库协作者（⚠️ 需确认）
gitlink-cli member +remove --owner myorg --repo myproject --user-id 101
```

---

## 完整命令速览

```bash
gitlink-cli member +list --owner <owner> --repo <repo> --format json
gitlink-cli member +add --user-id <id> --role <role>
gitlink-cli member +update --user-id <id> --role <role>
gitlink-cli member +remove --user-id <id>
```
