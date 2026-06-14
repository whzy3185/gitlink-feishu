# 流水线健康巡检工作流示例

本文档展示如何使用 `gitlink-pipeline-guardian` Skill 对仓库 `whale_hihihi/gitlink-cli` 执行完整的流水线健康巡检，生成健康度评分报告。

> **前置条件**：已完成 `gitlink-cli auth login` 认证。详见 [`../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

---

## Step 1：获取流水线列表

查看仓库下所有流水线的基本信息。

```bash
gitlink-cli pipeline +list --owner-id 123 --page 1 --limit 20 --format json
```

预期输出：

```json
{
  "pipelines": [
    {"id": 7, "name": "CI Build", "status": "active"},
    {"id": 8, "name": "Deploy Staging", "status": "active"},
    {"id": 9, "name": "Deploy Production", "status": "disabled"},
    {"id": 10, "name": "Nightly Tests", "status": "active"}
  ]
}
```

记录：共 4 条流水线，3 条可用，1 条已禁用（Deploy Production）。

---

## Step 2：获取近期运行记录

获取仓库的流水线运行历史。

```bash
gitlink-cli pipeline +runs --owner whale_hihihi --repo gitlink-cli --ref master --format json
```

预期输出：

```json
{
  "runs": [
    {"id": 101, "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 185, "started_at": "2026-06-08T09:00:00Z"},
    {"id": 100, "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 192, "started_at": "2026-06-08T06:00:00Z"},
    {"id": 99,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "failure",  "duration": 67,  "started_at": "2026-06-07T18:00:00Z"},
    {"id": 98,  "pipeline_id": 8, "workflow": "deploy-staging.yml", "ref": "master", "status": "success",  "duration": 340, "started_at": "2026-06-07T15:00:00Z"},
    {"id": 97,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 178, "started_at": "2026-06-07T12:00:00Z"},
    {"id": 96,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 201, "started_at": "2026-06-07T09:00:00Z"},
    {"id": 95,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "failure",  "duration": 45,  "started_at": "2026-06-06T18:00:00Z"},
    {"id": 94,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 188, "started_at": "2026-06-06T12:00:00Z"},
    {"id": 93,  "pipeline_id": 8, "workflow": "deploy-staging.yml", "ref": "master", "status": "success",  "duration": 355, "started_at": "2026-06-06T10:00:00Z"},
    {"id": 92,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 195, "started_at": "2026-06-06T06:00:00Z"},
    {"id": 91,  "pipeline_id": 10, "workflow": "nightly.yml", "ref": "master", "status": "success", "duration": 620, "started_at": "2026-06-05T02:00:00Z"},
    {"id": 90,  "pipeline_id": 7, "workflow": "ci.yml", "ref": "master", "status": "success",  "duration": 180, "started_at": "2026-06-05T09:00:00Z"}
  ]
}
```

统计摘要：
- 总运行：12 次
- 成功：10 次
- 失败：2 次
- 成功率：83.3%

---

## Step 3：获取失败运行的结果详情

对失败的 run 99 和 run 95 获取结果。

```bash
gitlink-cli pipeline +results --owner whale_hihihi --repo gitlink-cli --run-id 99 --format json
```

预期输出：

```json
{
  "run_id": 99,
  "status": "failure",
  "jobs": [
    {"index": 0, "name": "lint", "status": "success", "duration": 15},
    {"index": 1, "name": "test", "status": "failure", "duration": 52},
    {"index": 2, "name": "build", "status": "skipped", "duration": 0}
  ]
}
```

确定失败阶段：`test`（index=1）。

---

## Step 4：获取失败日志

获取 run 99 的 test 阶段日志。

```bash
gitlink-cli pipeline +logs --owner whale_hihihi --repo gitlink-cli --run-id 99 --id 7 --index 1 --format json
```

预期输出：

```json
{
  "logs": "--- Running test suite...\n=== FAIL: TestAuthLogin (0.32s)\n    auth_test.go:45: expected status 200, got 500\n    auth_test.go:46: server returned internal error\n=== FAIL: TestAPICall (0.18s)\n    api_test.go:112: connection refused to localhost:8080\nFAIL\nexit code 1"
```

错误模式匹配：
- `connection refused` → **F-TEST-002**（集成测试失败）
- `expected status 200, got 500` → **F-TEST-001**（单元测试断言失败）

---

## Step 5：查看流水线详情（配置审计）

查看各流水线的配置情况。

```bash
gitlink-cli pipeline +view --owner whale_hihihi --repo gitlink-cli --id 7 --format json
```

预期输出：

```json
{
  "id": 7,
  "name": "CI Build",
  "description": "Main CI pipeline for build, test and lint",
  "workflows": ["ci.yml"],
  "triggers": ["push", "pull_request"],
  "status": "active"
}
```

---

## Step 6：补充 CI 构建数据

```bash
gitlink-cli ci +builds --owner whale_hihihi --repo gitlink-cli --format json
```

预期输出：

```json
{
  "builds": [
    {"id": 201, "status": "success", "branch": "master", "created_at": "2026-06-08T09:01:00Z"},
    {"id": 200, "status": "success", "branch": "master", "created_at": "2026-06-08T06:01:00Z"},
    {"id": 199, "status": "failed",  "branch": "master", "created_at": "2026-06-07T18:01:00Z"}
  ]
}
```

CI 数据与 Pipeline 数据交叉验证一致。

---

## Step 7：计算健康度评分

### 评分计算过程

**可用性（满分 20）**：
- 3 条可用 / 4 条总数 = 75%
- 得分：15/20

**成功率（满分 25）**：
- 10 成功 / 12 总运行 = 83.3%
- 区间 ≥80% → 得分：18/25

**稳定性（满分 20）**：
- 近 10 次：8 次成功
- 区间 8-9 次 → 得分：16/20

**性能（满分 15）**：
- CI 构建平均耗时：(185+192+67+178+201+45+188+195+180)/9 ≈ 159 秒 ≈ 2.7 分钟
- 区间 ≤5min → 得分：12/15

**频率（满分 10）**：
- 最近 7 天有 10 次运行，每天均有构建
- 得分：10/10

**规范性（满分 10）**：
- 有命名规范（CI Build / Deploy Staging / Nightly Tests）：3 分
- 失败通知：未知（保守记 0 分）
- 缓存策略：未知（保守记 0 分）
- 并行配置：未知（保守记 0 分）
- 得分：3/10

### 汇总

| 维度 | 得分 | 满分 |
|------|------|------|
| 可用性 | 15 | 20 |
| 成功率 | 18 | 25 |
| 稳定性 | 16 | 20 |
| 性能 | 12 | 15 |
| 频率 | 10 | 10 |
| 规范性 | 3 | 10 |
| **总分** | **74** | **100** |

**等级：C（需关注）**

---

## 完整报告

以下是生成的完整巡检报告。

---

# 流水线健康巡检报告：whale_hihihi/gitlink-cli

> 巡检时间：2026-06-08 10:30:00
> 仓库：whale_hihihi/gitlink-cli
> 流水线数量：4（3 可用 / 1 禁用）

---

## 一、健康度总览

| 指标 | 数值 | 评分 |
|------|------|------|
| 可用性 | 3/4 条流水线可用 | 15/20 |
| 成功率 | 83.3%（10/12） | 18/25 |
| 稳定性 | 近 10 次 8 次成功 | 16/20 |
| 性能 | 平均耗时 2.7 分钟 | 12/15 |
| 频率 | 每天有构建 | 10/10 |
| 规范性 | 通过 1/4 项 | 3/10 |
| **总分** | | **74/100（等级 C — 需关注）** |

## 二、运行趋势

最近 20 次运行状态：
```
S S F S S S F S S S S S
```
（S=成功  F=失败  R=运行中  -=待执行  C=已取消）

| 时间段 | 总运行 | 成功 | 失败 | 成功率 |
|--------|--------|------|------|--------|
| 最近 7 天 | 10 | 8 | 2 | 80% |
| 7-14 天 | 2 | 2 | 0 | 100% |

## 三、性能概览

| 指标 | 数值 |
|------|------|
| 平均耗时 | 2 分 39 秒 |
| 中位数耗时 | 3 分 5 秒 |
| P90 耗时 | 5 分 40 秒 |
| 最快构建 | 45 秒（#95 — 失败，提前终止） |
| 最慢构建 | 10 分 20 秒（#91 — Nightly Tests） |

### 按流水线分组

| 流水线 | 运行次数 | 平均耗时 | 成功率 |
|--------|----------|----------|--------|
| CI Build (#7) | 9 | 3 分 5 秒 | 77.8% |
| Deploy Staging (#8) | 2 | 5 分 48 秒 | 100% |
| Nightly Tests (#10) | 1 | 10 分 20 秒 | 100% |

## 四、故障摘要

| 运行 ID | 工作流 | 分支 | 状态 | 耗时 | 错误模式 |
|---------|--------|------|------|------|----------|
| #99 | ci.yml | master | failure | 1 分 7 秒 | F-TEST-001 单元测试断言失败，F-TEST-002 集成测试连接拒绝 |
| #95 | ci.yml | master | failure | 45 秒 | F-TEST-002 集成测试连接拒绝（未获取详细日志） |

### 根因分析

两次失败均发生在 `test` 阶段，共同特征为 `connection refused to localhost:8080`，表明测试依赖的本地服务在构建环境中未正确启动。这是**集成测试环境配置问题**（F-TEST-002），而非代码逻辑错误。

## 五、改进建议

### 高优先级

1. **修复集成测试环境**：两次失败均为测试服务连接失败。建议在 CI 工作流中添加服务启动步骤，确保 localhost:8080 在测试运行前可用。可使用 `docker-compose` 或内联脚本启动依赖服务。

2. **启用 Deploy Production 流水线**：当前 Deploy Production（#9）处于禁用状态。如已不再需要，建议删除以减少管理负担；如仍需要，建议评估后重新启用。

### 中优先级

3. **配置失败通知**：当前未检测到失败通知机制。建议配置 Webhook 或邮件通知，确保构建失败时维护者能及时响应。

4. **添加缓存策略**：CI 构建平均 3 分钟，其中依赖安装可能占比较高。建议启用依赖缓存（如 Go module cache、npm cache）以缩短构建时间。

### 低优先级

5. **并行化测试阶段**：Nightly Tests 耗时超过 10 分钟，可考虑将测试拆分为多个并行任务以缩短总耗时。

6. **启用 Deploy Production 后配置审批门禁**：确保生产部署需人工确认，避免自动部署引入风险。

---

*报告由 gitlink-pipeline-guardian 生成*
