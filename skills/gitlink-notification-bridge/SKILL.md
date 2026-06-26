---
name: gitlink-notification-bridge
version: 1.0.0
description: "跨平台通知桥接：将 GitLink 仓库事件实时推送到飞书/钉钉群机器人，实现跨平台协作通知。当用户需要配置 GitLink 与飞书/钉钉集成时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli webhook --help"
---

# gitlink-notification-bridge（跨平台通知桥接）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。
> **执行样例：** 参见 [`EXAMPLES.md`](EXAMPLES.md)

---

## 功能概述

帮助用户快速配置 GitLink 仓库与飞书/钉钉群机器人的集成，实现：

1. **实时推送** — 代码推送、PR、Issue 等事件实时推送到群聊
2. **多平台支持** — 支持飞书（Feishu）和钉钉（DingTalk）
3. **事件过滤** — 可选择订阅特定事件类型
4. **分支过滤** — 可限定监听特定分支的推送
5. **安全验证** — 支持 Webhook Secret 签名验证

---

## 支持的平台

| 平台 | Webhook 类型 | 官方文档 |
|------|-------------|----------|
| **飞书** | `feishu` | [飞书机器人开发文档](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN) |
| **钉钉** | `dingtalk` | [钉钉机器人开发文档](https://open.dingtalk.com/document/robots/custom-robot-access) |

---

## 工作流：快速配置通知桥接

### Step 1：获取群机器人 Webhook URL

#### 飞书群机器人

1. 打开飞书群聊 → 点击群设置 → 群机器人 → 添加机器人
2. 选择「自定义机器人」
3. 设置机器人名称和描述
4. 复制 Webhook 地址（格式：`https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx`）
5. （可选）设置签名密钥，增强安全性

#### 钉钉群机器人

1. 打开钉钉群聊 → 点击群设置 → 智能群助手 → 添加机器人
2. 选择「自定义」机器人
3. 设置机器人名称
4. 安全设置选择「加签」，复制密钥（Secret）
5. 复制 Webhook 地址（格式：`https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxx`）

---

### Step 2：配置 GitLink Webhook

使用 `gitlink-cli webhook +create` 命令创建 Webhook：

#### 飞书配置示例

```bash
# 基础配置（推送事件）
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx" \
  --type "feishu" \
  --events "push" \
  --branch-filter "master,main" \
  --active "true"

# 完整配置（多事件 + 签名）
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx" \
  --type "feishu" \
  --events "push,pull_request_only,issues_only" \
  --branch-filter "master,main,release-*" \
  --secret "your-feishu-secret" \
  --active "true"
```

#### 钉钉配置示例

```bash
# 基础配置（推送事件）
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxx" \
  --type "dingtalk" \
  --events "push" \
  --branch-filter "master,main" \
  --active "true"

# 完整配置（多事件 + 签名）
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxx" \
  --type "dingtalk" \
  --events "push,pull_request_only,issues_only" \
  --branch-filter "master,main,release-*" \
  --secret "your-dingtalk-secret" \
  --active "true"
```

---

### Step 3：验证配置

#### 发送测试事件

```bash
# 获取 Webhook ID
gitlink-cli webhook +list --format json

# 发送测试事件
gitlink-cli webhook +test --id <webhook_id> --format json
```

#### 检查投递历史

```bash
# 查看投递记录
gitlink-cli webhook +history --id <webhook_id> --format json
```

**预期结果：**
- 群聊中收到测试消息
- 投递历史显示 HTTP 200 状态码

---

## 事件类型说明

| 事件 | 说明 | 推荐场景 |
|------|------|----------|
| `push` | 代码推送 | 监听代码提交 |
| `pull_request_only` | PR 创建/更新 | 代码评审流程 |
| `pull_request_comment` | PR 评论 | 评审讨论通知 |
| `pull_request_assign` | PR 分配 | 任务分配通知 |
| `issues_only` | Issue 创建/更新 | 问题跟踪 |
| `issue_comment` | Issue 评论 | 问题讨论通知 |
| `issue_assign` | Issue 分配 | 任务分配通知 |
| `create` | 创建分支/标签 | 版本管理 |
| `delete` | 删除分支/标签 | 清理通知 |

**推荐组合：**

- **开发团队**：`push,pull_request_only,pull_request_comment`
- **运维团队**：`push,create,delete`
- **产品团队**：`issues_only,issue_comment`
- **全量监控**：`push,pull_request_only,issues_only,create,delete`

---

## 分支过滤规则

`--branch-filter` 参数支持通配符：

| 过滤规则 | 说明 | 示例 |
|----------|------|------|
| `*` | 所有分支（默认） | 监听所有分支 |
| `master` | 单个分支 | 仅监听 master |
| `master,main` | 多个分支 | 监听 master 和 main |
| `release-*` | 通配符匹配 | 匹配 release-1.0, release-2.0 |
| `feature/*` | 路径匹配 | 匹配 feature/login, feature/api |

**推荐配置：**

- **生产环境**：`master,main,release-*`
- **开发环境**：`*` 或 `feature/*,develop`
- **测试环境**：`test-*`

---

## 安全配置

### Webhook Secret（签名验证）

GitLink 支持 Webhook Secret 签名验证，防止伪造请求：

#### 飞书签名

```bash
# 创建时设置 Secret
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx" \
  --type "feishu" \
  --events "push" \
  --secret "your-strong-secret"

# 更新 Secret
gitlink-cli webhook +update \
  --id <webhook_id> \
  --secret "new-strong-secret"
```

**注意：** 飞书群机器人需在创建时启用「签名校验」，Secret 需与 GitLink 配置一致。

#### 钉钉签名

```bash
# 创建时设置 Secret（使用钉钉机器人的加签密钥）
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxx" \
  --type "dingtalk" \
  --events "push" \
  --secret "SECxxxxxxxxxxxxxxxxxxxx"
```

**注意：** 钉钉机器人的 Secret 在创建机器人时生成，需完整复制。

---

## 高级配置

### 多群通知

为不同群聊创建不同的 Webhook：

```bash
# 开发群 - 监听所有事件
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/dev-group" \
  --type "feishu" \
  --events "push,pull_request_only,issues_only" \
  --branch-filter "*"

# 运维群 - 仅监听生产分支
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/ops-group" \
  --type "feishu" \
  --events "push,create,delete" \
  --branch-filter "master,main,release-*"

# 产品群 - 仅监听 Issue
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/product-group" \
  --type "feishu" \
  --events "issues_only,issue_comment" \
  --branch-filter "*"
```

### 混合平台配置

同时推送到飞书和钉钉：

```bash
# 飞书群
gitlink-cli webhook +create \
  --url "https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx" \
  --type "feishu" \
  --events "push,pull_request_only" \
  --branch-filter "master,main"

# 钉钉群
gitlink-cli webhook +create \
  --url "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxx" \
  --type "dingtalk" \
  --events "push,pull_request_only" \
  --branch-filter "master,main"
```

---

## 故障排查

### 常见问题

#### 1. 群聊未收到消息

**检查步骤：**

```bash
# 1. 检查 Webhook 是否激活
gitlink-cli webhook +view --id <webhook_id> --format json

# 2. 检查投递历史
gitlink-cli webhook +history --id <webhook_id> --format json

# 3. 发送测试事件
gitlink-cli webhook +test --id <webhook_id> --format json
```

**可能原因：**
- Webhook 未激活（`active: false`）
- URL 错误或已失效
- 签名验证失败（Secret 不匹配）
- 事件未触发（分支过滤或事件订阅不匹配）

#### 2. 签名验证失败

**错误表现：** 投递历史显示 HTTP 403 或群聊提示签名错误

**解决方案：**

```bash
# 检查当前 Secret 配置
gitlink-cli webhook +view --id <webhook_id> --format json

# 更新 Secret（确保与群机器人配置一致）
gitlink-cli webhook +update \
  --id <webhook_id> \
  --secret "correct-secret"

# 重新测试
gitlink-cli webhook +test --id <webhook_id> --format json
```

#### 3. 消息格式异常

**可能原因：**
- Webhook 类型配置错误（`--type` 应为 `feishu` 或 `dingtalk`）
- Content-Type 不匹配（默认 `json`，通常无需修改）

**解决方案：**

```bash
# 检查类型配置
gitlink-cli webhook +view --id <webhook_id> --format json

# 更新类型
gitlink-cli webhook +update \
  --id <webhook_id> \
  --type "feishu"
```

#### 4. 特定分支未触发通知

**检查分支过滤规则：**

```bash
# 查看当前过滤规则
gitlink-cli webhook +view --id <webhook_id> --format json

# 更新过滤规则
gitlink-cli webhook +update \
  --id <webhook_id> \
  --branch-filter "master,main,feature/*"
```

---

## 管理操作

### 查看所有 Webhook

```bash
# 列出仓库所有 Webhook
gitlink-cli webhook +list --format json
```

### 更新配置

```bash
# 更新事件订阅
gitlink-cli webhook +update \
  --id <webhook_id> \
  --events "push,pull_request_only,issues_only"

# 更新分支过滤
gitlink-cli webhook +update \
  --id <webhook_id> \
  --branch-filter "master,main"

# 暂停通知（不删除）
gitlink-cli webhook +update \
  --id <webhook_id> \
  --active "false"
```

### 删除 Webhook

```bash
# ⚠️ 删除前确认
gitlink-cli webhook +delete --id <webhook_id>
```

---

## 输出模板

配置完成后，生成配置摘要：

```markdown
## GitLink 跨平台通知桥接配置完成

### 配置概要

| 项目 | 详情 |
|------|------|
| 仓库 | <owner>/<repo> |
| 平台 | 飞书 / 钉钉 |
| Webhook ID | <id> |
| 状态 | ✅ 已激活 |

### 订阅配置

| 配置项 | 值 |
|--------|-----|
| 事件类型 | <events> |
| 分支过滤 | <branch_filter> |
| 签名验证 | ✅ 已启用 / ❌ 未启用 |

### 测试结果

| 测试项 | 结果 |
|--------|------|
| 测试投递 | ✅ 成功 / ❌ 失败 |
| HTTP 状态码 | 200 |
| 群聊收到消息 | ✅ 是 / ❌ 否 |

### 后续操作

1. 触发实际事件验证（推送代码、创建 PR 等）
2. 监控投递历史：`gitlink-cli webhook +history --id <id>`
3. 如需调整，使用 `gitlink-cli webhook +update --id <id>`

---
*由 gitlink-notification-bridge Skill 自动生成*
```

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **创建/更新 Webhook 为写操作**，执行前确认参数
- ✅ **Secret 需与群机器人配置一致**，否则签名验证失败
- ⚠️ **飞书/钉钉 Webhook URL 需从群设置中获取**，无法手动构造
- ⚠️ **删除 Webhook 是不可逆操作**，建议先设为未激活测试
- ⚠️ **分支过滤区分大小写**，确保分支名称正确

---

## 相关 Skill 交叉引用

| Skill | 关联场景 |
|-------|----------|
| [`gitlink-shared`](../gitlink-shared/SKILL.md) | 认证、全局参数、安全规则基础 |
| [`gitlink-webhook-sentinel`](../gitlink-webhook-sentinel/SKILL.md) | Webhook 监控、故障诊断、安全审计 |
| [`gitlink-notification-digest`](../gitlink-notification-digest/SKILL.md) | GitLink 平台内通知摘要 |
