# gitlink-notification-bridge 使用样例

## 样例 1：飞书群机器人配置（基础场景）

**日期**：2026-06-26
**用户**：developer
**CLI 版本**：gitlink-cli 0.1.18
**场景**：将 GitLink 仓库的代码推送事件推送到飞书开发群

### 执行流程

```bash
# Step 1: 获取飞书群机器人 Webhook URL
# （在飞书群聊中操作）
# 1. 打开群设置 → 群机器人 → 添加机器人
# 2. 选择「自定义机器人」
# 3. 复制 Webhook 地址
# → https://open.feishu.cn/open-apis/bot/v2/hook/abc123def456

# Step 2: 创建 Webhook（仅监听 master 和 main 分支）
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/abc123def456" \
  --type "feishu" \
  --events "push" \
  --branch-filter "master,main" \
  --active "true" \
  --format json

# Step 3: 验证配置
# 获取刚创建的 Webhook ID
gitlink-cli webhook +list --format json

# 发送测试事件
gitlink-cli webhook +test --id 12345 --format json

# 检查投递历史
gitlink-cli webhook +history --id 12345 --format json
```

### 预期输出

**创建成功响应：**

```json
{
  "id": 12345,
  "url": "https://open.feishu.cn/open-apis/bot/v2/hook/abc123def456",
  "type": "feishu",
  "events": ["push"],
  "branch_filter": "master,main",
  "active": true,
  "created_at": "2026-06-26T10:30:00Z"
}
```

**测试投递成功：**

```json
{
  "webhook_id": 12345,
  "test_event": "push",
  "delivery_status": "success",
  "http_status": 200,
  "response_time_ms": 245
}
```

**飞书群消息示例：**

```
📦 代码推送通知

仓库：myorg/myproject
分支：master
提交者：developer
提交数：3

最新提交：
• feat: 添加用户认证模块 (abc1234)
• fix: 修复登录页面样式 (def5678)
• docs: 更新 README (ghi9012)

查看详情：https://www.gitlink.org.cn/myorg/myproject
```

### 关键发现

| 项目 | 值 |
|------|-----|
| Webhook 类型 | `feishu` |
| 支持事件 | `push`, `pull_request_only`, `issues_only` 等 |
| 分支过滤 | 支持 `*`, `master`, `feature/*` 等通配符 |
| 投递延迟 | < 500ms（正常情况） |
| 消息格式 | 飞书卡片消息（自动适配） |

---

## 样例 2：钉钉群机器人配置（带签名验证）

**日期**：2026-06-26
**场景**：将 GitLink 仓库的 PR 和 Issue 事件推送到钉钉运维群，启用签名验证

### 执行流程

```bash
# Step 1: 获取钉钉群机器人 Webhook URL 和 Secret
# （在钉钉群聊中操作）
# 1. 群设置 → 智能群助手 → 添加机器人
# 2. 选择「自定义」
# 3. 安全设置选择「加签」
# 4. 复制 Secret：SECxxxxxxxxxxxxxxxxxxxx
# 5. 复制 Webhook：https://oapi.dingtalk.com/robot/send?access_token=xyz789

# Step 2: 创建 Webhook（多事件 + 签名）
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=xyz789" \
  --type "dingtalk" \
  --events "push,pull_request_only,issues_only" \
  --branch-filter "master,main,release-*" \
  --secret "SECxxxxxxxxxxxxxxxxxxxx" \
  --active "true" \
  --format json

# Step 3: 验证签名配置
gitlink-cli webhook +view --id 12346 --format json

# Step 4: 发送测试事件
gitlink-cli webhook +test --id 12346 --format json
```

### 预期输出

**创建成功响应：**

```json
{
  "id": 12346,
  "url": "https://oapi.dingtalk.com/robot/send?access_token=xyz789",
  "type": "dingtalk",
  "events": ["push", "pull_request_only", "issues_only"],
  "branch_filter": "master,main,release-*",
  "secret": "********",  // 已脱敏
  "active": true,
  "created_at": "2026-06-26T11:00:00Z"
}
```

**钉钉群消息示例（PR 事件）：**

```
🔀 Pull Request 通知

仓库：myorg/myproject
标题：feat: 添加用户认证模块
作者：developer
状态：待审核

PR #42
分支：feature/auth → master

查看详情：https://www.gitlink.org.cn/myorg/myproject/pulls/42
```

### 签名验证说明

钉钉机器人使用「加签」安全设置时，GitLink 会自动计算签名：

```
签名算法：HmacSHA256(timestamp + "\n" + secret, secret)
请求头：X-Timestamp, X-Sign
```

**注意事项：**
- Secret 必须与钉钉机器人设置中的「加签密钥」完全一致
- 签名验证失败会返回 HTTP 403
- 建议先测试无签名配置，确认可用后再启用签名

---

## 样例 3：多群通知配置

**场景**：为不同团队配置不同的通知策略

### 执行流程

```bash
# 开发群 - 监听所有分支的所有事件
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/dev-group" \
  --type "feishu" \
  --events "push,pull_request_only,pull_request_comment,issues_only,issue_comment" \
  --branch-filter "*" \
  --active "true" \
  --format json

# 运维群 - 仅监听生产分支的推送和分支操作
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/ops-group" \
  --type "feishu" \
  --events "push,create,delete" \
  --branch-filter "master,main,release-*,hotfix-*" \
  --active "true" \
  --format json

# 产品群 - 仅监听 Issue 相关事件
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/product-group" \
  --type "feishu" \
  --events "issues_only,issue_comment,issue_assign,issue_label" \
  --branch-filter "*" \
  --active "true" \
  --format json

# 查看所有 Webhook
gitlink-cli webhook +list --format json
```

### 预期输出

**Webhook 列表：**

```json
[
  {
    "id": 12347,
    "url": "https://open.feishu.cn/open-apis/bot/v2/hook/dev-group",
    "type": "feishu",
    "events": ["push", "pull_request_only", "pull_request_comment", "issues_only", "issue_comment"],
    "branch_filter": "*",
    "active": true
  },
  {
    "id": 12348,
    "url": "https://open.feishu.cn/open-apis/bot/v2/hook/ops-group",
    "type": "feishu",
    "events": ["push", "create", "delete"],
    "branch_filter": "master,main,release-*,hotfix-*",
    "active": true
  },
  {
    "id": 12349,
    "url": "https://open.feishu.cn/open-apis/bot/v2/hook/product-group",
    "type": "feishu",
    "events": ["issues_only", "issue_comment", "issue_assign", "issue_label"],
    "branch_filter": "*",
    "active": true
  }
]
```

---

## 样例 4：混合平台配置（飞书 + 钉钉）

**场景**：同时推送到飞书和钉钉群

### 执行流程

```bash
# 飞书群
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/feishu-group" \
  --type "feishu" \
  --events "push,pull_request_only" \
  --branch-filter "master,main" \
  --active "true" \
  --format json

# 钉钉群
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=dingtalk-token" \
  --type "dingtalk" \
  --events "push,pull_request_only" \
  --branch-filter "master,main" \
  --active "true" \
  --format json

# 验证两个平台都收到通知
gitlink-cli webhook +test --id 12350 --format json  # 飞书
gitlink-cli webhook +test --id 12351 --format json  # 钉钉
```

---

## 样例 5：故障排查 - 群聊未收到消息

**场景**：配置后群聊未收到通知

### 诊断流程

```bash
# Step 1: 检查 Webhook 是否激活
gitlink-cli webhook +view --id 12345 --format json

# 输出示例：
{
  "id": 12345,
  "active": false,  // ← 问题：未激活
  "url": "https://open.feishu.cn/open-apis/bot/v2/hook/abc123",
  "type": "feishu",
  "events": ["push"]
}

# Step 2: 激活 Webhook
gitlink-cli webhook +update \
  --id 12345 \
  --active "true" \
  --format json

# Step 3: 检查投递历史
gitlink-cli webhook +history --id 12345 --format json

# 输出示例：
{
  "deliveries": [
    {
      "id": "d001",
      "event_type": "push",
      "delivered_at": "2026-06-26T10:35:00Z",
      "http_status": 200,  // ← 成功
      "response_time_ms": 180
    }
  ]
}

# Step 4: 发送测试事件
gitlink-cli webhook +test --id 12345 --format json
```

### 常见问题诊断表

| 问题 | 检测方式 | 解决方案 |
|------|----------|----------|
| Webhook 未激活 | `active: false` | `--active "true"` |
| URL 错误 | 投递历史 HTTP 404 | 重新获取正确的 Webhook URL |
| 签名验证失败 | 投递历史 HTTP 403 | 确保 Secret 与群机器人配置一致 |
| 事件未触发 | 分支过滤不匹配 | 更新 `--branch-filter` |
| 事件类型不匹配 | 事件订阅不包含 | 更新 `--events` |

---

## 样例 6：故障排查 - 签名验证失败

**场景**：钉钉群机器人返回签名错误

### 诊断流程

```bash
# Step 1: 检查投递历史
gitlink-cli webhook +history --id 12346 --format json

# 输出示例：
{
  "deliveries": [
    {
      "id": "d002",
      "event_type": "push",
      "delivered_at": "2026-06-26T11:05:00Z",
      "http_status": 403,  // ← 签名验证失败
      "error": "signature verification failed"
    }
  ]
}

# Step 2: 检查当前 Secret 配置
gitlink-cli webhook +view --id 12346 --format json

# Step 3: 更新 Secret（确保与钉钉机器人「加签密钥」一致）
gitlink-cli webhook +update \
  --id 12346 \
  --secret "SECcorrectsecret123456" \
  --format json

# Step 4: 重新测试
gitlink-cli webhook +test --id 12346 --format json
```

### 签名验证要点

**飞书签名：**
- 在飞书群机器人设置中启用「签名校验」
- 设置自定义 Secret（建议 16-32 位随机字符串）
- GitLink 配置相同的 Secret

**钉钉签名：**
- 钉钉机器人创建时选择「加签」安全设置
- 系统自动生成 Secret（以 `SEC` 开头）
- 完整复制 Secret 到 GitLink 配置

---

## 样例 7：更新和删除 Webhook

**场景**：调整现有配置或清理不再使用的 Webhook

### 更新配置

```bash
# 更新事件订阅
gitlink-cli webhook +update \
  --id 12345 \
  --events "push,pull_request_only,issues_only" \
  --format json

# 更新分支过滤
gitlink-cli webhook +update \
  --id 12345 \
  --branch-filter "master,main,develop,feature/*" \
  --format json

# 暂停通知（不删除）
gitlink-cli webhook +update \
  --id 12345 \
  --active "false" \
  --format json

# 恢复通知
gitlink-cli webhook +update \
  --id 12345 \
  --active "true" \
  --format json
```

### 删除 Webhook

```bash
# ⚠️ 删除前确认
gitlink-cli webhook +view --id 12345 --format json

# 删除
gitlink-cli webhook +delete --id 12345 --format json

# 验证已删除
gitlink-cli webhook +list --format json
```

---

## 样例 8：通过 Agent 调用 Skill（自动配置）

**日期**：2026-06-26
**调用方式**：`Agent(subagent_type="general", prompt="请帮我配置 GitLink 仓库的飞书通知，监听 master 分支的推送事件。")`

### Agent 自主执行的命令序列

```
工具调用 1: Read  → ../gitlink-shared/SKILL.md              ← 遵循 Skill 前置条件
工具调用 2: Read  → ./SKILL.md                              ← 读取当前 Skill
工具调用 3: Bash  → gitlink-cli webhook +list --format json  ← 检查现有配置
工具调用 4: Bash  → gitlink-cli webhook +create ...          ← 创建 Webhook
工具调用 5: Bash  → gitlink-cli webhook +test --id ...       ← 发送测试
工具调用 6: Bash  → gitlink-cli webhook +history --id ...    ← 验证投递
```

### Agent 决策过程

Agent **正确遵循了 Skill 工作流**：

1. 先读取 `gitlink-shared/SKILL.md` 了解认证和安全规则
2. 读取当前 Skill 了解配置流程
3. 检查现有 Webhook 避免重复创建
4. 引导用户获取飞书群机器人 Webhook URL
5. 创建 Webhook 并验证配置
6. 生成配置摘要报告

### Agent 生成的配置摘要

```markdown
## GitLink 跨平台通知桥接配置完成

### 配置概要

| 项目 | 详情 |
|------|------|
| 仓库 | myorg/myproject |
| 平台 | 飞书 |
| Webhook ID | 12345 |
| 状态 | ✅ 已激活 |

### 订阅配置

| 配置项 | 值 |
|--------|-----|
| 事件类型 | push |
| 分支过滤 | master |
| 签名验证 | ❌ 未启用 |

### 测试结果

| 测试项 | 结果 |
|--------|------|
| 测试投递 | ✅ 成功 |
| HTTP 状态码 | 200 |
| 群聊收到消息 | ✅ 是 |

### 后续操作

1. 触发实际推送事件验证
2. 监控投递历史：`gitlink-cli webhook +history --id 12345`
3. 如需调整，使用 `gitlink-cli webhook +update --id 12345`

---
*由 gitlink-notification-bridge Skill 自动生成*
```

---

## 异常场景速查

| 场景 | 检测方式 | 处理 |
|------|----------|------|
| Webhook URL 格式错误 | 创建时返回 400 | 确保格式正确（飞书：`https://open.feishu.cn/open-apis/bot/v2/hook/xxx`） |
| 不支持的 Webhook 类型 | `--type` 参数报错 | 使用 `feishu` 或 `dingtalk` |
| 分支过滤语法错误 | 创建时返回 400 | 使用逗号分隔，支持 `*` 通配符 |
| 事件类型不存在 | 创建时返回 400 | 参考支持的事件列表 |
| 投递超时 | 投递历史显示 timeout | 检查网络连接，重试 |
| 消息格式异常 | 群聊显示乱码 | 检查 `--type` 是否正确 |

---

## 版本兼容性说明

本 skill v1.0.0 基于 `gitlink-cli 0.1.18` 编写。

### 支持的 Webhook 类型

| 类型 | 说明 | GitLink 版本 |
|------|------|--------------|
| `feishu` | 飞书群机器人 | v0.1.0+ |
| `dingtalk` | 钉钉群机器人 | v0.1.0+ |
| `slack` | Slack | v0.1.0+ |
| `discord` | Discord | v0.1.0+ |
| `telegram` | Telegram | v0.1.0+ |
| `msteams` | Microsoft Teams | v0.1.0+ |
| `matrix` | Matrix | v0.1.0+ |

### 支持的事件类型

| 事件 | 说明 | GitLink 版本 |
|------|------|--------------|
| `push` | 代码推送 | v0.1.0+ |
| `create` | 创建分支/标签 | v0.1.0+ |
| `delete` | 删除分支/标签 | v0.1.0+ |
| `issues_only` | Issue 创建/更新 | v0.1.0+ |
| `issue_assign` | Issue 分配 | v0.1.0+ |
| `issue_label` | Issue 标签变更 | v0.1.0+ |
| `issue_comment` | Issue 评论 | v0.1.0+ |
| `pull_request_only` | PR 创建/更新 | v0.1.0+ |
| `pull_request_assign` | PR 分配 | v0.1.0+ |
| `pull_request_comment` | PR 评论 | v0.1.0+ |

当 CLI 版本更新后，重新验证可用命令：

```bash
gitlink-cli webhook --help
gitlink-cli webhook +create --help
```

---

## 最佳实践

### 1. 安全配置

- ✅ 生产环境建议启用签名验证
- ✅ Secret 使用 16-32 位随机字符串
- ✅ 定期轮换 Secret（建议每季度）
- ⚠️ 不要在公开仓库中暴露 Webhook URL

### 2. 事件订阅策略

| 团队类型 | 推荐事件 | 说明 |
|----------|----------|------|
| 开发团队 | `push,pull_request_only,pull_request_comment` | 关注代码变更和评审 |
| 运维团队 | `push,create,delete` | 关注分支和版本管理 |
| 产品团队 | `issues_only,issue_comment` | 关注需求和问题跟踪 |
| QA 团队 | `pull_request_only,issues_only` | 关注测试任务 |

### 3. 分支过滤策略

| 环境 | 推荐过滤规则 | 说明 |
|------|--------------|------|
| 生产环境 | `master,main,release-*` | 仅监听关键分支 |
| 开发环境 | `*` 或 `feature/*,develop` | 监听所有或开发分支 |
| 测试环境 | `test-*` | 仅监听测试分支 |

### 4. 监控和维护

```bash
# 定期检查投递成功率
gitlink-cli webhook +history --id <id> --format json | jq '.deliveries | map(select(.http_status != 200)) | length'

# 批量检查所有 Webhook 状态
gitlink-cli webhook +list --format json | jq '.[] | {id, active, type}'

# 清理未激活的 Webhook
gitlink-cli webhook +list --format json | jq '.[] | select(.active == false) | .id' | xargs -I {} gitlink-cli webhook +delete --id {}
```

---

## 相关资源

- [飞书机器人开发文档](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
- [钉钉机器人开发文档](https://open.dingtalk.com/document/robots/custom-robot-access)
- [GitLink Webhook API 文档](https://www.gitlink.org.cn/docs/api#webhook)
- [gitlink-webhook-sentinel Skill](../gitlink-webhook-sentinel/SKILL.md) - Webhook 监控和故障诊断
