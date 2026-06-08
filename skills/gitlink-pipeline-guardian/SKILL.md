---
name: gitlink-pipeline-guardian
version: 1.0.0
description: "流水线健康守护：监控 CI/CD 流水线状态、分析失败模式、识别慢速构建、生成健康度评分报告。当用户需要排查流水线故障或优化构建效率时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pipeline --help"
---

# gitlink-pipeline-guardian（流水线健康守护）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读巡检与分析工具。涉及 pipeline 的启用/禁用/删除/运行操作，需经用户确认后方可执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 定位与产物

面向项目维护者的流水线健康守护工具，核心能力：

1. **流水线健康巡检** — 全面扫描流水线运行状态，生成 0-100 健康度评分报告
2. **故障根因分析** — 针对失败构建深入分析日志，定位错误模式并给出修复建议
3. **性能优化分析** — 分析构建耗时分布，识别瓶颈阶段并提出优化方案
4. **流水线配置审计** — 审计流水线配置规范性，推荐最佳实践

与 `gitlink-ci-health`（关注 CI/CD 基础设施层，使用 `ci +builds`）不同，本 Skill 关注的是**流水线工作流层面**（Pipeline），使用 `pipeline +runs`、`pipeline +logs`、`pipeline +results` 等命令，覆盖从构建编排到执行结果的完整链路。

### 命令矩阵

| 命令 | 用途 | 涉及工作流 |
|------|------|-----------|
| `pipeline +list` | 列出平台流水线 | 巡检、配置审计 |
| `pipeline +runs` | 列出流水线运行记录 | 巡检、故障分析、性能分析 |
| `pipeline +view` | 查看流水线详情 | 巡检、配置审计 |
| `pipeline +logs` | 查询流水线运行日志 | 故障分析 |
| `pipeline +results` | 查看流水线运行结果 | 巡检、故障分析、性能分析 |
| `ci +builds` | 查看 CI 构建列表 | 巡检（补充） |
| `ci +log` | 查看 CI 构建日志 | 故障分析（补充） |
| `ci +restart` | 重启 CI 构建 | 故障修复（需确认） |

---

## 工作流 1：流水线健康巡检

**触发场景**：用户说"流水线怎么样""Pipeline 健康度""流水线报告""构建状态一览"。

### Step 1：获取流水线列表

```bash
gitlink-cli pipeline +list --owner-id <owner_id> --page 1 --limit 20 --format json
```

记录所有流水线的 `id`、`name`、`status`。

### Step 2：获取近期运行记录

对每个流水线获取运行历史：

```bash
gitlink-cli pipeline +runs --owner <owner> --repo <repo> --ref master --format json
```

提取每次运行的：
- `status` — 运行状态（success / failure / running / pending / cancelled）
- `started_at` / `finished_at` — 时间信息
- `duration` — 耗时
- `workflow` — 触发的工作流文件
- `branch` / `ref` — 触发分支

### Step 3：获取运行结果详情

对最近的关键运行获取结果：

```bash
gitlink-cli pipeline +results --owner <owner> --repo <repo> --run-id <run_id> --format json
```

### Step 4：计算健康度评分

按以下评分模板计算总分（满分 100）。

#### 健康度评分模板

| 维度 | 权重 | 满分 | 评分标准 |
|------|------|------|----------|
| **可用性** | — | 20 | 流水线全部可用=20，部分禁用按比例扣分，全部不可用=0 |
| **成功率** | — | 25 | ≥95%=25，≥90%=22，≥80%=18，≥70%=12，≥60%=8，<60%=4，无数据=0 |
| **稳定性** | — | 20 | 近10次全部成功=20，8-9次=16，6-7次=12，4-5次=8，<4次=4 |
| **性能** | — | 15 | 平均耗时 ≤2min=15，≤5min=12，≤10min=9，≤20min=6，>20min=3 |
| **频率** | — | 10 | 每天有构建=10，2-3天=8，每周=5，更少=2，无运行=0 |
| **规范性** | — | 10 | 有命名规范=3，有失败通知=3，有缓存策略=2，有并行配置=2，无=0 |

**总评分级**：

| 分数范围 | 等级 | 状态 |
|----------|------|------|
| 90-100 | A | 健康 |
| 75-89 | B | 良好 |
| 60-74 | C | 需关注 |
| 40-59 | D | 需改进 |
| 0-39 | F | 严重 |

### Step 5：生成健康巡检报告

按输出模板（见文末）生成报告。

---

## 工作流 2：故障根因分析

**触发场景**：用户说"这个构建为什么失败""流水线报错了""帮我排查一下 pipeline 失败"。

### Step 1：定位失败运行

```bash
# 获取近期运行，找到状态为 failure 的记录
gitlink-cli pipeline +runs --owner <owner> --repo <repo> --format json
```

### Step 2：获取运行结果

```bash
gitlink-cli pipeline +results --owner <owner> --repo <repo> --run-id <failed_run_id> --format json
```

确定失败的阶段（stage）和步骤（step）。

### Step 3：获取失败日志

```bash
gitlink-cli pipeline +logs --owner <owner> --repo <repo> --run-id <failed_run_id> --id <pipeline_id> --index <job_index> --format json
```

> **注意**：`pipeline +logs` 需要提供 `--run-id`、`--id`（流水线 ID）和 `--index`（作业索引）。先通过 `pipeline +results` 获取这些参数。

### Step 4：补充 CI 日志（如需要）

```bash
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli ci +log --build <build_id> --format json
```

### Step 5：分析错误模式

将日志中的错误信息与**常见失败模式目录**（见文末）匹配，定位根因。

### Step 6：输出分析报告

包含：
- 失败构建基本信息（ID、分支、时间）
- 错误日志关键片段
- 匹配的失败模式
- 根因分析结论
- 修复建议（具体的代码或配置修改方案）

---

## 工作流 3：性能优化分析

**触发场景**：用户说"构建太慢了""优化一下流水线""构建耗时分析"。

### Step 1：收集运行耗时数据

```bash
gitlink-cli pipeline +runs --owner <owner> --repo <repo> --format json
```

提取每次运行的 `duration`，计算：
- 平均耗时
- 中位数耗时
- 最大 / 最小耗时
- P90 / P95 耗时

### Step 2：获取各阶段结果

对多次运行获取结果，对比各阶段耗时：

```bash
gitlink-cli pipeline +results --owner <owner> --repo <repo> --run-id <run_id> --format json
```

### Step 3：识别瓶颈阶段

按阶段统计平均耗时，找出耗时最长的 TOP 3 阶段。

### Step 4：慢速构建分析

识别超过平均耗时 1.5 倍的构建，分析可能的慢速原因：
- 依赖安装阶段过长 → 未使用缓存
- 测试阶段过长 → 未并行执行
- 构建阶段过长 → 未增量构建
- 部署阶段过长 → 资源不足

### Step 5：输出优化建议

包含：
- 当前耗时统计（表格）
- 瓶颈阶段排名
- 慢速构建列表及原因
- 具体优化建议（带预期收益估算）

---

## 工作流 4：流水线配置审计

**触发场景**：用户说"流水线配置合不合理""审计一下 pipeline""流水线最佳实践检查"。

### Step 1：获取流水线详情

```bash
gitlink-cli pipeline +list --owner-id <owner_id> --format json
```

对每个流水线：

```bash
gitlink-cli pipeline +view --owner <owner> --repo <repo> --id <pipeline_id> --format json
```

### Step 2：审计检查清单

逐项检查以下配置规范：

| 检查项 | 审计标准 | 状态 |
|--------|----------|------|
| **命名规范** | 流水线名称清晰、有业务含义 | 通过/不通过 |
| **触发条件** | 配置了合理的触发分支和事件 | 通过/不通过 |
| **超时设置** | 各阶段设置了合理的超时时间 | 通过/不通过 |
| **重试策略** | 关键阶段配置了重试 | 通过/不通过 |
| **缓存配置** | 依赖安装阶段使用了缓存 | 通过/不通过 |
| **并行执行** | 无依赖的阶段配置了并行 | 通过/不通过 |
| **通知配置** | 配置了失败通知机制 | 通过/不通过 |
| **环境变量** | 敏感信息通过 Secret 管理 | 通过/不通过 |
| **版本锁定** | 依赖版本已锁定（非 latest） | 通过/不通过 |
| **清理策略** | 配置了构建产物清理 | 通过/不通过 |

### Step 3：生成审计报告

按检查清单生成报告，标注通过率和不通过项的改进建议。

---

## 常见失败模式目录

以下为流水线构建中的常见失败模式，用于故障根因分析时的模式匹配。

### 编译/构建错误

| 模式 ID | 模式名称 | 关键特征 | 常见原因 |
|---------|----------|----------|----------|
| F-COMP-001 | 依赖下载失败 | `npm ERR!`、`Could not resolve`、`download failed` | 网络问题、私有源不可达、版本不存在 |
| F-COMP-002 | 编译语法错误 | `SyntaxError`、`compilation error`、`parse error` | 代码语法问题、语言版本不兼容 |
| F-COMP-003 | 内存不足 | `OOM`、`out of memory`、`heap`、`137 exit code` | 构建资源不足、内存泄漏 |
| F-COMP-004 | 磁盘空间不足 | `No space left`、`ENOSPC`、`disk full` | 构建缓存堆积、产物过大 |
| F-COMP-005 | 版本不兼容 | `version mismatch`、`incompatible`、`unsupported version` | 运行时版本与代码不匹配 |

### 测试错误

| 模式 ID | 模式名称 | 关键特征 | 常见原因 |
|---------|----------|----------|----------|
| F-TEST-001 | 单元测试失败 | `FAIL`、`AssertionError`、`expected but got` | 代码逻辑错误、测试用例过时 |
| F-TEST-002 | 集成测试失败 | `connection refused`、`timeout`、`ECONNREFUSED` | 服务依赖不可用、环境配置错误 |
| F-TEST-003 | 测试超时 | `timeout`、`exceeded`、`Deadline exceeded` | 测试死锁、外部依赖响应慢 |
| F-TEST-004 | 测试覆盖率不达标 | `coverage`、`threshold`、`below` | 代码缺少测试覆盖 |

### 环境/配置错误

| 模式 ID | 模式名称 | 关键特征 | 常见原因 |
|---------|----------|----------|----------|
| F-ENV-001 | 环境变量缺失 | `undefined`、`not set`、`missing env` | Secret 未配置、变量名拼写错误 |
| F-ENV-002 | 权限不足 | `permission denied`、`403`、`unauthorized` | 凭证过期、角色权限不足 |
| F-ENV-003 | Docker 构建失败 | `Dockerfile`、`image not found`、`build failed` | 基础镜像不存在、Dockerfile 语法错误 |
| F-ENV-004 | 资源限制 | `rate limit`、`too many requests`、`429` | API 调用频率超限 |

### 部署错误

| 模式 ID | 模式名称 | 关键特征 | 常见原因 |
|---------|----------|----------|----------|
| F-DEPLOY-001 | 部署超时 | `deployment timeout`、`rollout stuck` | 镜像拉取慢、资源不足 |
| F-DEPLOY-002 | 部署验证失败 | `health check failed`、`unhealthy` | 应用启动失败、配置错误 |
| F-DEPLOY-003 | 回滚触发 | `rollback`、`reverted` | 部署后检测到故障自动回滚 |

---

## 输出模板：流水线健康巡检报告

```markdown
# 流水线健康巡检报告：{{仓库名}}

> 巡检时间：{{当前时间}}
> 仓库：{{full_name}}
> 流水线数量：{{pipeline_count}}

---

## 一、健康度总览

| 指标 | 数值 | 评分 |
|------|------|------|
| 可用性 | {{可用流水线}}/{{总数}} | {{availability_score}}/20 |
| 成功率 | {{success_rate}}%（{{success_count}}/{{total_count}}） | {{success_score}}/25 |
| 稳定性 | 近 10 次 {{recent_success}} 次成功 | {{stability_score}}/20 |
| 性能 | 平均耗时 {{avg_duration}} | {{performance_score}}/15 |
| 频率 | {{frequency_desc}} | {{frequency_score}}/10 |
| 规范性 | 通过 {{passed_checks}}/{{total_checks}} 项 | {{compliance_score}}/10 |
| **总分** | | **{{total_score}}/100（等级 {{grade}}）** |

## 二、运行趋势

最近 20 次运行状态：
{{status_bar}}
（S=成功  F=失败  R=运行中  -=待执行  C=已取消）

| 时间段 | 总运行 | 成功 | 失败 | 成功率 |
|--------|--------|------|------|--------|
| 最近 7 天 | {{w1_total}} | {{w1_success}} | {{w1_fail}} | {{w1_rate}}% |
| 7-14 天 | {{w2_total}} | {{w2_success}} | {{w2_fail}} | {{w2_rate}}% |
| 14-30 天 | {{w3_total}} | {{w3_success}} | {{w3_fail}} | {{w3_rate}}% |

## 三、性能概览

| 指标 | 数值 |
|------|------|
| 平均耗时 | {{avg_duration}} |
| 中位数耗时 | {{median_duration}} |
| P90 耗时 | {{p90_duration}} |
| 最快构建 | {{min_duration}}（#{{min_run_id}}） |
| 最慢构建 | {{max_duration}}（#{{max_run_id}}） |

## 四、故障摘要

> 如无失败运行，输出：**分析期内无失败运行，流水线运行健康。**

| 运行 ID | 工作流 | 分支 | 状态 | 耗时 | 错误模式 |
|---------|--------|------|------|------|----------|
| {{run_id}} | {{workflow}} | {{branch}} | {{status}} | {{duration}} | {{error_pattern}} |

## 五、改进建议

{{improvement_suggestions}}

---
*报告由 gitlink-pipeline-guardian 生成*
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| 无流水线 | 报告"仓库尚未配置流水线"，建议创建 `.gitea/workflows` 或通过 Web 界面创建 |
| 流水线全部禁用 | 标注"所有流水线已禁用"，可用性评分为 0 |
| 运行记录为空 | 标注"流水线无运行记录"，跳过成功率和性能评分 |
| `pipeline +logs` 返回空 | 标注"日志不可用"，基于 `pipeline +results` 进行有限分析 |
| 构建记录 < 5 次 | 标注"样本量不足，统计不具代表性" |
| 流水线数量 > 10 | 仅分析最近活跃的 10 条流水线 |

---

## 最佳实践

- 所有命令使用 `--format json`，确保输出可解析
- Owner/repo 优先从 `git remote` 自动解析
- 巡检类操作为只读，不会修改任何流水线配置
- 涉及启用/禁用/删除/运行操作需经用户确认后执行
- 批量获取运行记录时控制 API 调用频率，避免请求过快
- 日志分析时仅提取关键错误行，避免处理过多数据
- 将巡检报告保存为文件以便后续对比和追踪趋势
