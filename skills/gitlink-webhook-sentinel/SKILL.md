---
name: gitlink-webhook-sentinel
version: 1.0.0
description: "Webhook 监控哨兵：监控 Webhook 投递成功率、检测端点问题、验证安全配置、分析失败原因。当用户需要排查 Webhook 集成问题或监控 Webhook 健康状态时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli webhook --help"
---

# gitlink-webhook-sentinel（Webhook 监控哨兵）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 工作流概览

本 Skill 提供一套完整的 Webhook 监控与诊断工作流，覆盖从健康巡检到故障排查、安全审计和配置优化的全过程。

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| 1 健康巡检 | 列出所有 Webhook，检查投递历史，识别失败端点 | 采集数据并生成健康报告 |
| 2 故障诊断 | 深入分析失败投递记录，检查响应码和 Payload | 定位根因并给出修复建议 |
| 3 安全审计 | 验证 Webhook Secret、Content-Type 配置、端点安全性 | 检出安全隐患并生成审计报告 |
| 4 配置优化 | 审查事件订阅、分支过滤器、配置合理性 | 提出优化建议并协助调整 |

---

## 命令参考

| 命令 | 说明 |
|------|------|
| `gitlink-cli webhook +list` | 列出仓库所有 Webhook |
| `gitlink-cli webhook +view --id <id>` | 查看单个 Webhook 详情 |
| `gitlink-cli webhook +create` | 创建 Webhook |
| `gitlink-cli webhook +update --id <id>` | 更新 Webhook |
| `gitlink-cli webhook +delete --id <id>` | 删除 Webhook |
| `gitlink-cli webhook +history --id <id>` | 查看 Webhook 投递历史（任务列表） |
| `gitlink-cli webhook +test --id <id>` | 发送测试事件到 Webhook 端点 |

### 支持的 Webhook 类型

| 类型 | 说明 |
|------|------|
| `gitea` | Gitea 原生格式（默认） |
| `slack` | Slack 通知 |
| `discord` | Discord 通知 |
| `dingtalk` | 钉钉机器人 |
| `telegram` | Telegram Bot |
| `msteams` | Microsoft Teams |
| `feishu` | 飞书机器人 |
| `matrix` | Matrix 协议 |
| `jianmu` | 建木 CI |
| `softbot` | SoftBot |

### 支持的事件类型

| 事件 | 说明 |
|------|------|
| `push` | 推送代码 |
| `create` | 创建分支/标签 |
| `delete` | 删除分支/标签 |
| `issues_only` | Issue 创建/更新 |
| `issue_assign` | Issue 分配 |
| `issue_label` | Issue 标签变更 |
| `issue_comment` | Issue 评论 |
| `pull_request_only` | PR 创建/更新 |
| `pull_request_assign` | PR 分配 |
| `pull_request_comment` | PR 评论 |

### create/update 参数

| 参数 | 短选项 | 说明 | 默认值 |
|------|--------|------|--------|
| `--url` | `-u` | Webhook 端点 URL | （必填） |
| `--events` | `-e` | 逗号分隔的事件列表 | （必填） |
| `--type` | `-t` | Webhook 类型 | `gitea` |
| `--content-type` | | 内容格式：`json` 或 `form` | `json` |
| `--http-method` | | HTTP 方法：`GET` 或 `POST` | `POST` |
| `--secret` | `-s` | Webhook 签名密钥 | |
| `--branch-filter` | | 分支过滤通配符 | `*` |
| `--active` | | 是否激活：`true` 或 `false` | `true` |

---

## 工作流 1：Webhook 健康巡检

**场景**：定期巡检仓库所有 Webhook 的运行状态，识别投递失败的端点。

### Step 1：获取所有 Webhook 列表

```bash
# 列出仓库所有 Webhook
gitlink-cli webhook +list --format json
```

分析返回数据，关注以下字段：
- `id`：Webhook ID
- `url`：端点地址
- `active`：是否激活
- `type`：Webhook 类型
- `events`：订阅的事件列表
- `branch_filter`：分支过滤规则

### Step 2：逐个检查投递历史

```bash
# 查看每个 Webhook 的投递任务记录
gitlink-cli webhook +history --id <webhook_id> --format json
```

对每个 Webhook，统计：
- 总投递次数
- 成功次数（HTTP 2xx 响应）
- 失败次数（HTTP 4xx/5xx 响应或超时）
- 投递成功率

### Step 3：发送测试事件验证活跃端点

```bash
# 对可疑或长时间无投递记录的 Webhook 发送测试
gitlink-cli webhook +test --id <webhook_id> --format json
```

### Step 4：生成健康报告

按以下模板输出巡检报告：

```markdown
## Webhook 健康巡检报告 — <owner>/<repo>

### 总览

| 指标 | 数值 |
|------|------|
| Webhook 总数 | <total> |
| 活跃 Webhook | <active_count> |
| 未激活 Webhook | <inactive_count> |
| 健康端点 | <healthy_count> |
| 异常端点 | <unhealthy_count> |
| 整体成功率 | <success_rate>% |

### 端点健康状态

| ID | URL | 类型 | 状态 | 成功率 | 最近失败原因 |
|----|-----|------|:----:|:------:|-------------|
| <id> | <url> | <type> | Healthy/Warning/Critical | xx% | <reason> |

### 异常端点详情

#### Webhook #<id> — <url>
- **类型**：<type>
- **事件订阅**：<events>
- **分支过滤**：<branch_filter>
- **最近投递状态**：<last_delivery_status>
- **连续失败次数**：<consecutive_failures>
- **建议操作**：<recommendation>

---
*由 gitlink-webhook-sentinel Skill 自动生成*
```

---

## 工作流 2：故障诊断

**场景**：当某个 Webhook 投递失败时，深入分析失败原因。

### Step 1：获取 Webhook 详情

```bash
# 查看目标 Webhook 完整配置
gitlink-cli webhook +view --id <webhook_id> --format json
```

重点检查：
- `url`：端点地址是否正确、是否可达
- `content_type`：与目标系统期望的格式是否一致
- `http_method`：是否与端点期望的方法一致
- `active`：是否处于激活状态
- `secret`：是否已配置签名密钥

### Step 2：获取投递历史

```bash
# 获取投递任务列表（包含每次投递的详细信息）
gitlink-cli webhook +history --id <webhook_id> --format json
```

分析每次投递记录：
- 响应状态码（`response_status_code`）
- 响应内容（如有）
- 投递时间
- 是否成功

### Step 3：根据状态码定位问题

参考「常见失败响应码参考表」进行匹配分析，确定根因类别。

### Step 4：发送测试验证

```bash
# 发送测试事件以复现/验证问题
gitlink-cli webhook +test --id <webhook_id> --format json
```

### Step 5：输出诊断报告

```markdown
## Webhook 故障诊断报告 — #<id> <url>

### 诊断摘要

| 项目 | 详情 |
|------|------|
| Webhook ID | <id> |
| 端点 URL | <url> |
| 类型 | <type> |
| 问题类别 | <category> |
| 严重程度 | Critical/Warning/Info |

### 根因分析

**主要原因**：<primary_cause>

**证据**：
- <evidence_1>
- <evidence_2>

### 修复建议

1. **立即操作**：<immediate_fix>
2. **后续验证**：<verification_step>
3. **长期优化**：<long_term_improvement>

### 相关投递记录

| 时间 | 状态码 | 结果 |
|------|--------|------|
| <time> | <code> | Success/Failed |
```

---

## 工作流 3：安全审计

**场景**：审查 Webhook 配置的安全性，检查 Secret 配置、Content-Type 和端点安全性。

### Step 1：获取所有 Webhook 配置

```bash
# 列出所有 Webhook
gitlink-cli webhook +list --format json

# 逐个查看详细配置
gitlink-cli webhook +view --id <webhook_id> --format json
```

### Step 2：安全检查清单

逐项审查以下安全指标：

| 检查项 | 安全标准 | 风险等级 |
|--------|----------|:--------:|
| Secret 配置 | 所有生产 Webhook 必须配置签名密钥 | Critical |
| URL 协议 | 必须使用 HTTPS，禁止 HTTP | Critical |
| Content-Type | 推荐 `json`，避免 `form`（结构化更强） | Warning |
| URL 暴露信息 | URL 中不应包含 Token、密钥等敏感信息 | Critical |
| 事件范围 | 仅订阅必要事件，避免过度订阅 | Warning |
| 分支过滤 | 生产环境应设置分支过滤，不建议 `*` | Info |
| 激活状态 | 已废弃的 Webhook 应设为未激活或删除 | Info |

### Step 3：输出安全审计报告

```markdown
## Webhook 安全审计报告 — <owner>/<repo>

### 审计概要

| 指标 | 数值 |
|------|------|
| 审计 Webhook 数 | <total> |
| 通过检查 | <passed> |
| 存在风险 | <at_risk> |
| Critical 风险 | <critical_count> |
| Warning 风险 | <warning_count> |

### 检查结果

| ID | URL | Secret | HTTPS | Content-Type | 事件范围 | 风险等级 |
|----|-----|:------:|:-----:|:------------:|----------|:--------:|
| <id> | <url> | Yes/No | Yes/No | json/form | <events> | Healthy/At Risk |

### Critical 问题

#### Webhook #<id> — 未配置 Secret
- **影响**：任何人都可以伪造 Webhook Payload，存在安全风险
- **修复**：
  ```bash
  gitlink-cli webhook +update --id <id> --secret "<strong_secret>"
  ```
- **注意**：更新 Secret 后需同步更新接收端的验签逻辑

#### Webhook #<id> — 使用 HTTP 端点
- **影响**：Payload 以明文传输，可被中间人截获
- **修复**：将 URL 更改为 HTTPS 端点
  ```bash
  gitlink-cli webhook +update --id <id> --url "https://..."
  ```

### Warning 问题

<按严重程度列出所有 Warning 级别问题>

### 改进建议

1. <suggestion_1>
2. <suggestion_2>
```

---

## 工作流 4：Webhook 配置优化

**场景**：审查现有 Webhook 的配置合理性，优化事件订阅和分支过滤规则。

### Step 1：获取当前配置

```bash
# 获取所有 Webhook 列表
gitlink-cli webhook +list --format json

# 查看每个 Webhook 的详细配置
gitlink-cli webhook +view --id <webhook_id> --format json
```

### Step 2：配置审查

对每个 Webhook 逐项审查：

**事件订阅优化**：
- 是否订阅了从未触发过的事件？建议移除
- 是否遗漏了必要的事件？建议补充
- 同一端点是否存在多个 Webhook 重复订阅？建议合并

**分支过滤优化**：
- `branch_filter` 为 `*` 时：确认是否确实需要监听所有分支
- 生产环境建议限定为 `master,main,release-*` 等模式
- 开发环境可放宽为 `*` 或 `feature/*`

**类型与格式优化**：
- 目标系统类型是否正确（`gitea` / `slack` / `dingtalk` 等）
- `content_type` 是否与目标系统匹配（推荐 `json`）
- `http_method` 是否正确

**冗余清理**：
- 是否有指向已下线服务的 Webhook？应删除
- 是否有长期未激活的 Webhook？确认是否仍需要

### Step 3：生成优化建议

```markdown
## Webhook 配置优化报告 — <owner>/<repo>

### 优化概要

| 指标 | 数值 |
|------|------|
| 审查 Webhook 数 | <total> |
| 需要优化 | <needs_optimization> |
| 配置合理 | <well_configured> |
| 建议删除 | <recommend_deletion> |

### 优化建议明细

#### Webhook #<id> — <url>

**当前配置**：
- 事件：<current_events>
- 分支过滤：<current_branch_filter>
- 类型：<current_type>
- Content-Type：<current_content_type>

**优化建议**：
| 项目 | 当前值 | 建议值 | 原因 |
|------|--------|--------|------|
| events | <current> | <suggested> | <reason> |
| branch-filter | <current> | <suggested> | <reason> |

**执行命令**：
```bash
gitlink-cli webhook +update --id <id> --events "push,pull_request_only" --branch-filter "master,main"
```

### 冗余 Webhook 清理

| ID | URL | 原因 | 操作 |
|----|-----|------|------|
| <id> | <url> | <reason> | 删除/禁用 |
```

### Step 4：执行优化（需用户确认）

**CRITICAL — 所有修改操作前务必确认用户意图。**

```bash
# 更新 Webhook 配置
gitlink-cli webhook +update --id <id> --events "<optimized_events>" --branch-filter "<filter>"

# 删除确认废弃的 Webhook（需用户明确同意）
gitlink-cli webhook +delete --id <id>

# 更新后发送测试验证
gitlink-cli webhook +test --id <id> --format json
```

---

## 常见失败响应码参考表

| HTTP 状态码 | 含义 | 可能原因 | 排查方向 |
|:-----------:|------|----------|----------|
| `200` | 成功 | 正常投递 | 无需处理 |
| `301/302` | 重定向 | 端点 URL 已变更 | 更新为最终目标 URL |
| `400` | 请求无效 | Payload 格式错误、Content-Type 不匹配 | 检查 `content_type` 配置，验证 Payload 结构 |
| `401` | 未认证 | 目标端点要求认证但未提供 | 检查 URL 是否需要 Basic Auth，或在 URL 中嵌入认证参数 |
| `403` | 禁止访问 | 签名验证失败、IP 白名单未通过 | 检查 Secret 配置是否与接收端一致 |
| `404` | 未找到 | 端点 URL 错误或已下线 | 确认 URL 是否正确，服务是否在运行 |
| `408` | 请求超时 | 接收端处理过慢 | 优化接收端逻辑，或联系服务方 |
| `422` | 无法处理 | Payload 结构不符合目标系统预期 | 检查 `type` 配置是否匹配目标系统 |
| `429` | 请求过多 | 触发目标系统限流 | 降低事件触发频率或联系服务方提高限额 |
| `500` | 服务器内部错误 | 接收端服务异常 | 联系目标系统维护方 |
| `502` | 网关错误 | 目标服务器上游故障 | 检查目标服务是否正常，稍后重试 |
| `503` | 服务不可用 | 目标服务维护或过载 | 等待恢复后重试 |
| `504` | 网关超时 | 接收端响应时间过长 | 优化接收端处理逻辑，或设置异步处理 |
| Timeout | 连接超时 | 网络不通、DNS 解析失败、防火墙阻断 | 检查网络连通性、DNS 配置、防火墙规则 |

---

## Raw API 参考

Webhook 相关的 GitLink API 端点：

```bash
# 列出所有 Webhook
gitlink-cli api GET /v1/:owner/:repo/webhooks --format json

# 查看 Webhook 详情
gitlink-cli api GET /v1/:owner/:repo/webhooks/:id --format json

# 创建 Webhook
gitlink-cli api POST /v1/:owner/:repo/webhooks --body '{
  "type": "gitea",
  "active": true,
  "content_type": "json",
  "http_method": "POST",
  "url": "https://example.com/webhook",
  "secret": "your-secret",
  "branch_filter": "master",
  "events": ["push", "pull_request_only"]
}'

# 更新 Webhook
gitlink-cli api PUT /v1/:owner/:repo/webhooks/:id --body '{...}'

# 删除 Webhook
gitlink-cli api DELETE /v1/:owner/:repo/webhooks/:id

# 获取投递任务历史
gitlink-cli api GET /v1/:owner/:repo/webhooks/:id/hooktasks --format json

# 发送测试事件
gitlink-cli api POST /v1/:owner/:repo/webhooks/:id/tests
```

---

## 注意事项

- `webhook +history` 返回的是该 Webhook 的投递任务列表（hooktasks），包含每次投递的状态和响应信息
- `webhook +test` 会向目标端点发送一个测试 Payload，请确保目标服务能处理测试事件
- 更新 Webhook 的 Secret 时，接收端的验签逻辑需同步更新
- 删除 Webhook 是不可逆操作，执行前务必确认
- 对于使用 `--format json` 的命令，建议配合 `jq` 工具进行数据过滤和统计

---

## 相关 Skill 交叉引用

| Skill | 关联场景 |
|-------|----------|
| [`gitlink-shared`](../gitlink-shared/SKILL.md) | 认证、全局参数、安全规则基础 |
| [`gitlink-workflow`](../gitlink-workflow/SKILL.md) | AI 自动化工作流 |
| [`gitlink-health`](../gitlink-health/SKILL.md) | 项目整体健康度分析（Webhook 可作为子维度） |
