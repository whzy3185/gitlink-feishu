# Wiki 管理完整工作流示例

**场景**：项目维护者需要管理项目 Wiki 文档。

## 工作流步骤

### Step 1：列出 Wiki 页面

```bash
gitlink-cli wiki +list --owner myorg --repo myproject
```

**输出示例：**
```json
{
  "ok": true,
  "data": [
    { "name": "Home", "title": "首页", "updated_at": "2026-05-20" },
    { "name": "Getting-Started", "title": "快速开始", "updated_at": "2026-05-18" },
    { "name": "API-Reference", "title": "API 参考", "updated_at": "2026-05-15" }
  ]
}
```

### Step 2：查看和创建 Wiki 页面

```bash
# 查看 Wiki 页面内容
gitlink-cli wiki +view --name Getting-Started

# 创建新 Wiki 页面
gitlink-cli wiki +create \
  --name "Deployment-Guide" \
  --content "# 部署指南\n\n## Docker 部署\n\n```bash\ndocker build -t myapp .\ndocker run -p 8080:8080 myapp\n```\n\n## 手动部署\n\n..."
```

### Step 3：更新和删除 Wiki 页面

```bash
# 更新 Wiki 页面
gitlink-cli wiki +update \
  --name "Getting-Started" \
  --content "# 快速开始（已更新）\n\n新增了环境变量配置说明..." \
  --message "更新环境变量配置"

# 删除 Wiki 页面（⚠️ 需确认）
gitlink-cli wiki +delete --name Old-Page
```

---

## 完整命令速览

```bash
gitlink-cli wiki +list --owner <owner> --repo <repo>
gitlink-cli wiki +view --name <page-name>
gitlink-cli wiki +create --name <name> --content "..."
gitlink-cli wiki +update --name <name> --content "..." --message "..."
gitlink-cli wiki +delete --name <name>
```
