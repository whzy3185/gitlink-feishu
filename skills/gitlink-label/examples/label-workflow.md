# 标签管理完整工作流示例

**场景**：项目维护者需要创建和管理 Issue 标签体系。

## 工作流步骤

### Step 1：查看现有标签

```bash
gitlink-cli label +list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    { "id": 1, "name": "bug", "color": "#FF0000" },
    { "id": 2, "name": "enhancement", "color": "#00FF00" },
    { "id": 3, "name": "documentation", "color": "#0075CA" }
  ]
}
```

### Step 2：创建标签

```bash
# 创建 bug 标签
gitlink-cli label +create --name "bug" --color "#FF0000"

# 创建新人友好标签
gitlink-cli label +create --name "good first issue" --color "#7057FF"

# 创建优先级标签
gitlink-cli label +create --name "priority: high" --color "#FF6600"
```

### Step 3：更新和删除标签

```bash
# 更新标签颜色
gitlink-cli label +update --id 1 --color "#E74C3C"

# 删除标签（⚠️ 需确认）
gitlink-cli label +delete --id 5
```

---

## 完整命令速览

```bash
gitlink-cli label +list --owner <owner> --repo <repo> --format json
gitlink-cli label +create --name <name> --color <hex>
gitlink-cli label +update --id <id> --name <name> --color <hex>
gitlink-cli label +delete --id <id>
```
