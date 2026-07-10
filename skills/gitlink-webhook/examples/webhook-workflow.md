# Webhook 管理完整工作流示例

**场景**：项目维护者需要配置 Webhook 来实现自动化通知。

## 工作流步骤

### Step 1：查看 Webhook 列表

```bash
gitlink-cli webhook +list --owner myorg --repo myproject --format json
```

### Step 2：创建 Webhook

```bash
gitlink-cli webhook +create \
  --owner myorg --repo myproject \
  --url "https://example.com/webhook" \
  --events "push,pull_request,issues" \
  --secret "my-webhook-secret"
```

### Step 3：测试和更新 Webhook

```bash
# 查看 Webhook 详情
gitlink-cli webhook +view --id 10 --format json

# 更新 Webhook URL
gitlink-cli webhook +update --id 10 --url "https://new.example.com/webhook"

# 删除 Webhook（⚠️ 需确认）
gitlink-cli webhook +delete --id 10
```

---

## 完整命令速览

```bash
gitlink-cli webhook +list --owner <owner> --repo <repo> --format json
gitlink-cli webhook +create --url <url> --events <events> --secret <secret>
gitlink-cli webhook +view --id <id> --format json
gitlink-cli webhook +update --id <id> --url <url>
gitlink-cli webhook +delete --id <id>
```
