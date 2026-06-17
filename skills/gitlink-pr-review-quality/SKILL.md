---
name: gitlink-pr-review-quality
version: 1.0.0
description: "PR 审查效率看板：分析 PR 审查响应时间、审查者分布、沉默 PR 识别和审查健康度评分，输出优化建议。当用户需要了解 PR 审查是否成为瓶颈、审查响应速度、审查者覆盖度时触发。"
metadata:
requires:
bins: ["gitlink-cli"]
cliHelp: "gitlink-cli pr --help"
---

# gitlink-pr-review-quality（PR 审查效率看板）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读分析操作，不会修改任何 PR 或仓库数据。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

本技能量化 PR 审查流程的效率，提供以下分析能力：

1. **审查响应时间** — 从 PR 创建到首次 Review 的平均等待时长
2. **沉默 PR 识别** — 超过 N 天无人审查的 PR 清单
3. **审查者分布** — 各 Reviewer 的审查次数、评论密度、平均响应速度
4. **审查健康度评分** — A/B/C/D 四级评分 + 优化建议
5. **趋势分析** — 近期审查效率是否改善或恶化

---

## 一、数据采集：获取 PR 和审查记录

### 1.1 获取所有打开的 PR

```bash
# 获取所有打开的 PR（如果数量过大，需翻页处理）
gitlink-cli pr +list --state open --owner <owner> --repo <repo> --limit 50 --page 1 --format json
```

**AI 必须提取的关键字段**：
- `id`：PR ID
- `issue.id` 查询评论时用到的id
- `issue.journals_count` 日志数量（包括评论和操作日志，如果值为1，则说明只有一条创建日志，跳过审查记录步骤）
- `title`：PR 标题
- `pr_created_unix`：PR 创建时间（**计算响应时间的起点**）
- `status`：状态值 （open / merged / closed）
- `issue.author.login / issue.author.name`：PR 作者

### 1.2 获取 PR 的审查记录

```bash
# 获取某个 PR 的审查记录列表
gitlink-cli api GET v1/<owner>/<repo>/issues/<issue.id>/journals
```

**返回数据结构**：

| 字段 | 类型 | 说明 |
|------|------|------|
| journals | array | 审查记录列表 |
| journals[].id | integer | 审查 ID |
| journals[].is_journal_detail | bool | 是否为操作日志（内容分为操作、评论2种） |
| journals[].user | object | 审查者信息（id/name/login） |
| journals[].created_at | string | 审查创建时间 |
| journals[].notes | string | 审查评论内容 |
| journals[].operate_content | string | 操作日志 |

> **注意**：评论列表按创建时间正序排列，取第一条非作者评论即为首次响应。

### 1.3 获取已合并/已关闭的 PR（用于趋势分析）

```bash
# 获取最近已合并的 PR（分析审查效率趋势）
gitlink-cli pr +list --state merged --owner <owner> --repo <repo> --limit 50 --page 1 --format json

# 获取最近已关闭的 PR
gitlink-cli pr +list --state closed --owner <owner> --repo <repo> --limit 50 --page 1 --format json
```

---

## 二、核心指标计算

### 2.1 审查响应时间

**首次审查响应时间** = 第一条 Review/评论的 `created_at` - PR 的 `pr_created_unix`

```
计算流程：
├─ 获取 PR 的 pr_created_unix
├─ 获取该 PR 审查记录
├─ 过滤掉 PR 作者本人的评论/审查（非作者响应才算审查响应）
├─ 取最早的非作者评论/审查时间作为首次响应时间
│  └─ → 首次响应 = min(journals[非作者].created_at)
└─ 如果没有任何非作者响应 → 标记为"沉默 PR"
```

**响应时间等级定义**：

| 等级 | 响应时间 | 标签 | 说明 |
|------|---------|------|------|
| 🟢 极速 | ≤ 1 天 | 极速 | 审查者反应迅速，流程顺畅 |
| 🟡 正常 | 1-3 天 | 正常 | 响应在合理范围内 |
| 🟠 较慢 | 3-7 天 | 较慢 | 需关注，可能有审查瓶颈 |
| 🔴 严重滞后 | > 7 天 | 严重滞后 | 审查流程明显受阻 |

### 2.2 沉默 PR 识别

```
沉默 PR 定义：打开的 PR 中，超过 N 天没有任何非作者 Review 或评论。

识别流程：
1. 从 PR 列表中筛选 status == open
2. 对每个 PR 检查是否存在非作者的 Review 或评论
3. 如果不存在任何非作者响应 → 计算沉默天数 = 当前日期 - PR.pr_created_unix
4. 按沉默天数排序，优先展示最严重的沉默 PR

默认阈值：7 天（可在对话中自定义）
```

### 2.3 审查者分布统计

```
审查者统计流程：
1. 遍历所有已获取 Review 的 PR
2. 对每个 Review，提取审查者 login 和审查状态
3. 按 Reviewer login 聚合统计：
   - 审查总数
   - approved 比率
   - rejected 比率
   - 平均评论长度（content 字段的字符数）
   - 平均响应速度（首次审查的响应时间均值）

4. 输出审查者排行榜 + 覆盖度分析
```

**审查者覆盖度**：

| 指标 | 计算 | 健康标准 |
|------|------|---------|
| 审查者集中度 | 审查次数最多的 1 人占比 | < 40% 为健康 |
| 审查者数量 | 有 Review 记录的不同人数 | ≥ 3 人为健康 |
| 审查者缺席率 | 无 Review 的 PR 占比 | < 20% 为健康 |

### 2.4 审查健康度评分

**评分模型**：

```
审查健康度评分 = f(响应速度, 覆盖度, 沉默率)

维度权重：
- 响应速度（40%）：平均首次响应时间 → 映射到 0-100 分
- 覆盖度（30%）：审查者集中度 + 缺席率 → 映射到 0-100 分
- 沉默率（30%）：沉默 PR 占总 open PR 比率 → 映射到 0-100 分

评分等级：
├─ A（≥ 80 分）：审查流程高效，无明显瓶颈
├─ B（60-79 分）：审查流程正常，有少量可优化点
├─ C（40-59 分）：审查流程存在瓶颈，需要改进
└─ D（< 40 分）：审查流程严重受阻，亟需干预
```

---

## 三、分析报告格式

### 3.1 审查效率看板（主报告）

```markdown
## 📊 PR 审查效率看板 — <owner>/<repo>

**分析时间：** 2026-06-16
**数据范围：** 当前所有打开 + 最近 30 天已合并的 PR

---

### 🏥 审查健康度评分：B（68 分）

| 维度 | 得分 | 权重 | 加权分 | 评价 |
|------|------|------|--------|------|
| 响应速度 | 72 | 40% | 28.8 | 🟡 平均 2.3 天首次响应 |
| 覆盖度 | 65 | 30% | 19.5 | 🟠 3 人审查，1 人占 52% |
| 沉默率 | 55 | 30% | 16.5 | 🟠 18% 的 PR 无审查 |

---

### ⏱️ 审查响应时间分布

| 响应等级 | PR 数量 | 占比 | 平均天数 |
|---------|---------|------|---------|
| 🟢 极速（≤1天） | 8 | 40% | 0.6 |
| 🟡 正常（1-3天） | 6 | 30% | 1.8 |
| 🟠 较慢（3-7天） | 3 | 15% | 4.5 |
| 🔴 严重滞后（>7天） | 1 | 5% | 12 |
| ⚫ 沉默（无响应） | 2 | 10% | — |

**加权平均首次响应时间：2.3 天**

---

### 🧍 审查者排行榜

| 审查者 | 审查数 | approved | rejected | 平均评论长度 | 平均响应速度 |
|--------|--------|----------|----------|-------------|-------------|
| reviewer-a | 15 | 12 (80%) | 3 (20%) | 120 字 | 1.2 天 |
| reviewer-b | 8 | 6 (75%) | 2 (25%) | 85 字 | 3.5 天 |
| reviewer-c | 3 | 3 (100%) | 0 (0%) | 45 字 | 0.8 天 |

**审查者集中度：** reviewer-a 占 52% — ⚠️ 过高（建议 > 40%）
**审查者缺席率：** 18% — ⚠️ 偏高（建议 < 20%）

---

### 🤐 沉默 PR 清单

| # | PR | 标题 | 作者 | 创建日期 | 沉默天数 |
|---|-----|------|------|---------|---------|
| 1 | #42 | 重构用户认证模块 | dev-x | 2026-05-20 | 27 |
| 2 | #55 | 添加缓存层 | dev-y | 2026-06-01 | 15 |

---

### 📈 趋势分析

| 指标 | 本月 | 上月 | 变化趋势 |
|------|------|------|---------|
| 平均首次响应 | 2.3 天 | 3.1 天 | 🟢 改善 (-0.8 天) |
| 沉默 PR 占比 | 18% | 25% | 🟢 改善 (-7%) |
| 审查者集中度 | 52% | 48% | 🟠 恶化 (+4%) |

---

### 💡 优化建议

1. **[高优先级]** 培养更多审查者：当前 reviewer-a 承担 52% 审查量，一旦其忙碌/离职将严重影响流程
2. **[中优先级]** 立即审查沉默 PR #42 和 #55：已分别沉默 27/15 天
3. **[低优先级]** 建议设定 3 天审查 SLA：当前平均 2.3 天已达标，但需要制度保障防止恶化

---
*由 gitlink-pr-review-quality Skill 自动生成*
```

### 3.2 单个 PR 审查详情（可选）

```markdown
## 📋 PR #42 审查详情

| 指标 | 数据 |
|------|------|
| 标题 | 重构用户认证模块 |
| 作者 | dev-x |
| 创建时间 | 2026-05-20 |
| 首次响应 | ⚫ 无响应（沉默 27 天） |
| Review 数量 | 0 |
| 评论数量 | 0 |
| 审查者 | — |
| 审查状态 | 🔴 严重滞后 |

**问题**：此 PR 涉及认证模块重构（高风险变更），却没有任何审查，存在安全隐患。
```

---

## 四、执行步骤总览

### 4.1 完整分析流程

```bash
# Step 1：获取所有打开的 PR（分页）
gitlink-cli pr +list --state open --owner <owner> --repo <repo> --limit 50 --page 1 --format json

# Step 2：获取最近已合并的 PR（用于趋势分析，可选）
gitlink-cli pr +list --state merged --owner <owner> --repo <repo> --limit 50 --page 1 --format json

# Step 3：对每个 PR 获取审查记录和评论列表
gitlink-cli api GET v1/<owner>/<repo>/issues/<issue.id>/journals

# Step 4：AI 计算所有指标（响应时间、沉默 PR、审查者分布）

# Step 5：输出审查效率看板报告

# Step 6：输出优化建议
```

### 4.2 仅沉默 PR 模式

如果用户只想查看哪些 PR 没被审查：

```bash
# Step 1：获取所有打开的 PR
gitlink-cli pr +list --state open --owner <owner> --repo <repo> --limit 50 --page 1 --format json

# Step 2：对每个 PR 检查是否有 Review
gitlink-cli api GET v1/<owner>/<repo>/issues/<issue.id>/journals

# Step 3：筛选出无 Review 且无非作者评论的 PR

# Step 4：输出沉默 PR 清单
```

### 4.3 仅审查者分析模式

如果用户只想看审查者分布：

```bash
# Step 1：获取所有打开和已合并的 PR
gitlink-cli pr +list --state open --owner <owner> --repo <repo> --limit 50 --page 1 --format json
gitlink-cli pr +list --state merged --owner <owner> --repo <repo> --limit 50 --page 1 --format json

# Step 2：对每个 PR 获取 Review 记录
gitlink-cli api GET v1/<owner>/<repo>/issues/<issue.id>/journals

# Step 3：AI 按审查者聚合统计

# Step 4：输出审查者排行榜
```

---

## 五、可配置参数

用户可在对话中指定以下参数调整分析行为：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `silent_threshold_days` | 7 | 超过此天数无审查即标记为沉默 PR |
| `analysis_range_days` | 30 | 趋势分析的回溯天数（最近 N 天的已合并 PR） |
| `top_reviewers_limit` | 10 | 审查者排行榜最多展示人数 |
| `include_closed_prs` | false | 是否将已关闭 PR 纳入分析（默认仅分析 open PR） |
| `exclude_author_comments` | true | 响应时间计算时是否排除 PR 作者本人的评论 |

### 配置示例

```
用户："分析 PR 审查效率，沉默阈值设为 5 天，看看最近 60 天的趋势"

AI 应解析为：
- silent_threshold_days = 5
- analysis_range_days = 60
- top_reviewers_limit = 10（默认）
- include_closed_prs = false（默认）
- exclude_author_comments = true（默认）
```

---

## 六、评分映射规则

### 6.1 响应速度得分映射

| 平均首次响应时间 | 得分 |
|----------------|------|
| ≤ 0.5 天 | 100 |
| ≤ 1 天 | 90 |
| ≤ 2 天 | 80 |
| ≤ 3 天 | 70 |
| ≤ 5 天 | 55 |
| ≤ 7 天 | 40 |
| > 7 天 | 20 |

### 6.2 覆盖度得分映射

| 审查者集中度 | 审查者缺席率 | 得分 |
|------------|------------|------|
| < 30% 且 < 10% | 100 |
| < 40% 且 < 20% | 80 |
| 40-60% 或 20-30% | 60 |
| > 60% 或 > 30% | 40 |
| > 80% 或 > 50% | 20 |

### 6.3 沉默率得分映射

| 沉默 PR 占比 | 得分 |
|-------------|------|
| 0% | 100 |
| < 5% | 90 |
| < 10% | 80 |
| < 20% | 60 |
| < 30% | 40 |
| ≥ 30% | 20 |

---

## 七、常见场景示例

### 场景 1：审查效率全景分析

```
用户："帮我看看 myzc 项目 PR 审查效率怎么样"

AI 执行：
1. gitlink-cli pr +list --state open --owner yangsai --repo myzc --limit 50 --page 1 --format json
2. 对每个 open PR：
   - gitlink-cli api GET v1/<owner>/<repo>/issues/<issue.id>/journals
3. 计算首次响应时间、沉默 PR、审查者分布
4. 输出审查效率看板报告 + 优化建议
```

### 场景 2：找出没人看的 PR

```
用户："哪些 PR 一直没人审查？"

AI 执行：
1. gitlink-cli pr +list --state open --owner <owner> --repo <repo> --limit 50 --page 1 --format json
2. 对每个 open PR 检查 Review 记录
3. 筛选无 Review + 无非作者评论的 PR
4. 按沉默天数排序输出清单
5. 对高风险 PR（涉及安全/核心模块）特别标注
```

### 场景 3：审查者负载分析

```
用户："看看谁是审查主力，有没有审查瓶颈"

AI 执行：
1. 获取所有 PR + Review 数据
2. 按 Reviewer 聚合统计审查数量和响应速度
3. 计算审查者集中度和缺席率
4. 输出审查者排行榜 + 风险预警
5. 建议培养更多审查者（如果集中度过高）
```

### 场景 4：特定时间范围的趋势

```
用户："对比这个月和上个月的审查效率"

AI 执行：
1. 获取本月和上月已合并的 PR
2. 分别计算两组 PR 的平均响应时间
3. 计算沉默率变化
4. 输出趋势对比表 + 结论
```

---

## 八、注意事项

- ✅ **本 Skill 为只读分析**：不会修改任何 PR、Review 或仓库数据
- ✅ **排除 PR 作者本人**：响应时间计算中，PR 作者的评论不算审查响应，只有非作者才算
- ✅ **数据量处理**：如果 PR 数量 > 50，需要分页获取，避免遗漏
- ✅ **PR 状态筛选**：默认仅分析 open PR，趋势分析时才纳入 merged/closed PR
- ⚠️ **审查记录可能为空**：新仓库或小型项目可能没有任何 Review 记录，此时报告应标注"数据不足"
- ⚠️ **pr状态筛选**：请注意 pr +list 中的`--state`没有问题，无需使用 `pull_request_staus` 进行过滤