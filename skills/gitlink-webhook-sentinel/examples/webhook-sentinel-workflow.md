# Webhook 健康巡检工作流示例

本文档展示一个完整的 Webhook 健康巡检工作流，涵盖列出 Webhook、检查投递历史、识别问题端点、安全审计和配置优化的全过程。

> **前置条件**：已完成 `gitlink-cli auth login` 认证。所有命令使用 `gitlink-cli`，不使用 `gh`。

---

## 场景描述

仓库 `whale_hihihi/test` 配置了多个 Webhook，需要：
1. 巡检所有 Webhook 的健康状态
2. 诊断投递失败的端点
3. 审计安全配置
4. 优化 Webhook 配置

---

## Step 1：列出所有 Webhook

```bash
# 获取仓库所有 Webhook
gitlink-cli webhook +list --owner whale_hihihi --repo test --format json
```

**示例输出分析**：

假设仓库有 3 个 Webhook：
- `#10` — `https://ci.example.com/webhook` (Gitea, active)
- `#11` — `https://hooks.slack.com/services/T00/B00/xxx` (Slack, active)
- `#12` — `http://dev.local:3000/hook` (Gitea, active)

**初步发现**：
- Webhook #12 使用 HTTP 而非 HTTPS（安全风险）
- 共 3 个 Webhook，全部处于激活状态

---

## Step 2：逐个检查投递历史

```bash
# 检查 Webhook #10 的投递历史
gitlink-cli webhook +history --id 10 --format json

# 检查 Webhook #11 的投递历史
gitlink-cli webhook +history --id 11 --format json

# 检查 Webhook #12 的投递历史
gitlink-cli webhook +history --id 12 --format json
```

**示例分析**：

| Webhook | 总投递 | 成功 | 失败 | 成功率 | 状态 |
|---------|--------|------|------|--------|------|
| #10 CI | 50 | 48 | 2 | 96% | Healthy |
| #11 Slack | 30 | 25 | 5 | 83% | Warning |
| #12 Dev | 10 | 3 | 7 | 30% | Critical |

---

## Step 3：深入诊断异常端点

### 诊断 Webhook #11（Slack）

```bash
# 查看详细配置
gitlink-cli webhook +view --id 11 --format json
```

分析投递失败记录，发现响应码为 `403`。参考失败码表：
- `403` = 签名验证失败或 IP 白名单未通过

**结论**：Slack Webhook Secret 可能配置不正确。

### 诊断 Webhook #12（Dev Local）

```bash
# 查看详细配置
gitlink-cli webhook +view --id 12 --format json
```

分析投递失败记录，发现响应码为 `504`（网关超时）和 `Timeout`（连接超时）。

**结论**：本地开发服务器响应过慢或网络不通。

### 发送测试验证

```bash
# 对 Webhook #11 发送测试
gitlink-cli webhook +test --id 11 --format json

# 对 Webhook #12 发送测试
gitlink-cli webhook +test --id 12 --format json
```

---

## Step 4：安全审计

对每个 Webhook 逐项检查：

| 检查项 | #10 CI | #11 Slack | #12 Dev |
|--------|--------|-----------|---------|
| Secret 配置 | Yes | No | Yes |
| HTTPS | Yes | Yes | No |
| Content-Type | json | json | json |
| 事件范围 | push | push,issues_only,pull_request_only | push,create,delete |
| 分支过滤 | master | * | * |

**审计发现**：
- **Critical**：#11 未配置 Secret（可被伪造）
- **Critical**：#12 使用 HTTP（明文传输）
- **Warning**：#11 和 #12 的 `branch_filter` 为 `*`（监听所有分支）

---

## Step 5：配置优化建议

### Webhook #10（CI）— 配置合理
- 事件订阅和分支过滤合理，无需调整

### Webhook #11（Slack）— 需要修复
- 添加 Secret
- 收窄分支过滤

```bash
# 修复：添加 Secret 并限制分支过滤
gitlink-cli webhook +update --id 11 --secret "xJk9$mK2pL5qR8vW" --branch-filter "master,main"
```

### Webhook #12（Dev Local）— 建议禁用或删除
- 开发环境 Webhook，使用 HTTP 且不稳定
- 建议禁用或删除

```bash
# 选项 A：禁用
gitlink-cli webhook +update --id 12 --active false

# 选项 B：删除（需用户确认）
gitlink-cli webhook +delete --id 12
```

---

## Step 6：生成完整报告

巡检完成后，输出以下报告：

```markdown
## Webhook 健康巡检报告 — whale_hihihi/test

### 总览

| 指标 | 数值 |
|------|------|
| Webhook 总数 | 3 |
| 活跃 Webhook | 3 |
| 健康端点 | 1 |
| Warning 端点 | 1 |
| Critical 端点 | 1 |
| 整体成功率 | 76/90 (84.4%) |

### 端点健康状态

| ID | URL | 类型 | 状态 | 成功率 | 最近失败原因 |
|----|-----|------|:----:|:------:|-------------|
| 10 | https://ci.example.com/webhook | gitea | Healthy | 96% | — |
| 11 | https://hooks.slack.com/... | slack | Warning | 83% | 403 Forbidden |
| 12 | http://dev.local:3000/hook | gitea | Critical | 30% | 504 Gateway Timeout |

### 安全审计

| ID | Secret | HTTPS | 分支过滤 | 风险等级 |
|----|:------:|:-----:|----------|:--------:|
| 10 | Yes | Yes | master | Healthy |
| 11 | No | Yes | * | At Risk |
| 12 | Yes | No | * | At Risk |

### 异常端点详情

#### Webhook #11 — Slack 通知
- **问题**：Secret 未配置 + 分支过滤过宽
- **连续失败**：最近 5 次投递中 5 次返回 403
- **修复命令**：
  ```bash
  gitlink-cli webhook +update --id 11 --secret "<secret>" --branch-filter "master,main"
  ```

#### Webhook #12 — Dev Local
- **问题**：HTTP 明文传输 + 目标服务不稳定
- **连续失败**：最近 7 次投递中 7 次超时
- **修复建议**：确认服务是否仍在使用，若已废弃建议删除
  ```bash
  gitlink-cli webhook +delete --id 12
  ```

### 优化操作汇总

| 优先级 | 操作 | 命令 |
|--------|------|------|
| P0 | 为 Slack Webhook 添加 Secret | `gitlink-cli webhook +update --id 11 --secret "..."` |
| P1 | 禁用/删除 Dev Local Webhook | `gitlink-cli webhook +delete --id 12` |
| P2 | Slack Webhook 收窄分支过滤 | `gitlink-cli webhook +update --id 11 --branch-filter "master,main"` |

---
*由 gitlink-webhook-sentinel Skill 自动生成*
```

---

## 后续验证

修复操作完成后，再次执行测试验证：

```bash
# 验证修复后的 Webhook #11
gitlink-cli webhook +test --id 11 --format json

# 确认 Webhook #12 已删除
gitlink-cli webhook +list --format json
```

预期结果：
- Webhook #11 测试投递返回 200
- Webhook 列表中不再包含 #12
