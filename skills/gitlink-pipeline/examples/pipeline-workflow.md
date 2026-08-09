# Pipeline 管理完整工作流示例

**场景**：开发者需要查看和管理 CI/CD Pipeline 工作流。

## 工作流步骤

### Step 1：查看 Pipeline 列表

```bash
# 列出平台 Pipeline
gitlink-cli pipeline +list --owner-id 123 --page 1 --limit 20
```

### Step 2：查看仓库 Pipeline 运行记录

```bash
# 列出仓库的 Pipeline 运行记录
gitlink-cli pipeline +runs --owner Gitlink --repo forgeplus --ref master --workflow build.yml
```

### Step 3：查看 Pipeline 详情和日志

```bash
# 查看 Pipeline 详情
gitlink-cli pipeline +view --owner Gitlink --repo forgeplus --id 7

# 查看 Pipeline 运行日志
gitlink-cli pipeline +logs --owner Gitlink --repo forgeplus --run-id 99 --id 7 --index 43

# 查看运行报告结果
gitlink-cli pipeline +results --owner Gitlink --repo forgeplus --run-id 99
```

### Step 4：管理 Pipeline

```bash
# 启用 Pipeline 工作流（⚠️ 需确认）
gitlink-cli pipeline +enable --owner Gitlink --repo forgeplus --id 7 --workflow build.yml --dry-run

# 禁用 Pipeline 工作流
gitlink-cli pipeline +disable --owner Gitlink --repo forgeplus --id 7 --workflow build.yml --dry-run

# 删除 Pipeline（⚠️ 危险操作）
gitlink-cli pipeline +delete --owner Gitlink --repo forgeplus --id 7 --dry-run
```

### Step 5：保存 Pipeline 可视化 YAML

```bash
# 保存 Pipeline 图形为 YAML
gitlink-cli pipeline +save-yaml --owner Gitlink --repo forgeplus \
  --id 7 --pipeline-json '{"nodes":[]}' --dry-run
```

---

## 完整命令速览

```bash
gitlink-cli pipeline +list --owner-id <id> --page <n> --limit <n>
gitlink-cli pipeline +runs --owner <owner> --repo <repo> --ref <branch> --workflow <name>
gitlink-cli pipeline +view --owner <owner> --repo <repo> --id <id>
gitlink-cli pipeline +logs --owner <owner> --repo <repo> --run-id <id> --id <id> --index <n>
gitlink-cli pipeline +run --owner <owner> --repo <repo> --ref <branch> --workflow <name>
gitlink-cli pipeline +enable --owner <owner> --repo <repo> --id <id> --workflow <name>
gitlink-cli pipeline +disable --owner <owner> --repo <repo> --id <id> --workflow <name>
gitlink-cli pipeline +delete --owner <owner> --repo <repo> --id <id> --dry-run
```
