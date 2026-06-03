---
name: gitlink-contributor-insight
version: 1.0.0
description: "贡献者活跃度分析：分析仓库贡献者的活跃度、贡献趋势和工作节奏，生成贡献者洞察报告。当用户需要分析贡献者活跃度、查看团队贡献趋势、评估成员参与度时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli user --help"
---

# gitlink-contributor-insight（贡献者活跃度分析）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 为只读操作，不会修改任何仓库。无需用户额外确认即可执行。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

面向开源社区管理者和维护者的贡献者分析工具：

1. **项目概览** — 获取仓库贡献者规模
2. **贡献热力图分析** — 通过 `user +heatmap` 查看贡献节奏
3. **统计数据提取** — 通过 `user +stats` 获取个人统计
4. **趋势分析** — 通过 `user +trends` 查看项目趋势
5. **洞察报告** — 生成贡献者活跃度排名和团队健康度评估

---

## 工作流：贡献者分析全流程

### Step 1：获取项目贡献者列表

```bash
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

> 如果仓库贡献者数量较多（>15），按 `repo +contributors` 返回中 `commits_count` 降序排列，取前 10 位分析。如 `commits_count` 缺失，按返回的自然顺序取前 10 位，报告中注明"基于返回顺序 Top 10"。

提取每个贡献者的 `login`（用户名）。

### Step 2：获取仓库基本信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

提取 `contributor_users_count`、`full_name`、`description`。

### Step 3：逐位贡献者深度分析

对每位贡献者执行以下命令：

```bash
# 热力图（最近一年的贡献日历）
gitlink-cli user +heatmap --login <username> --format json

# 统计信息（PR/Issue/Commit 数量）
gitlink-cli user +stats --login <username> --format json

# 项目趋势
gitlink-cli user +trends --login <username> --format json
```

从返回数据中提取：

| 维度 | 来源命令 | 分析要点 |
|------|----------|----------|
| 贡献频率 | `+heatmap` | 最近 1/3/6/12 个月有贡献的天数，判断是"持续贡献者"还是"间歇参与者" |
| 贡献产出 | `+stats` | PR 数、Issue 数、Commit 数，区分"代码贡献者"和"问题反馈者" |
| 活跃趋势 | `+trends` | 贡献量是上升/稳定/下降，识别"上升期贡献者"和"逐渐淡出者" |

> ⚠️ **控制 API 调用**：贡献者 >15 人时，仅分析 Step 1 中按 `commits_count` 排序后的前 10 位。每人最多 3 次 API 调用（heatmap + stats + trends，共 ≤30 次）。

### Step 4：贡献者分级与分类

#### 4.1 活跃度分级

| 级别 | 判定标准 |
|------|----------|
| 🔥 **核心贡献者** | 最近 30 天有贡献 + 总贡献 PR > 10 |
| 🌟 **活跃贡献者** | 最近 60 天有贡献 + 总贡献 > 5 |
| 🌱 **新兴贡献者** | 最近 90 天首次出现 + 贡献频率上升 |
| 💤 **休眠贡献者** | 最近 90 天无贡献 + 历史有贡献 |

#### 4.2 贡献类型分类

| 类型 | 判定 |
|------|------|
| **代码贡献者** | PR/Commit 数量占比最高 |
| **问题反馈者** | Issue 数量占比最高 |
| **全能贡献者** | PR 和 Issue 数量均衡 |

### Step 5：生成贡献者洞察报告

---

## 输出模板

```markdown
# 👥 贡献者洞察报告：{{仓库名}}

> 分析时间：{{当前时间}}
> 仓库：{{full_name}}
> 总贡献者：{{contributor_users_count}} 人，本次分析：{{analyzed_count}} 人

---

## 一、团队概览

| 指标 | 数值 |
|------|------|
| 总贡献者 | {{contributor_users_count}} |
| 核心贡献者 | {{core_count}} |
| 活跃贡献者 | {{active_count}} |
| 新兴贡献者 | {{new_count}} |
| 休眠贡献者 | {{dormant_count}} |
| 近 30 天活跃率 | {{active_30d_rate}}% |

---

## 二、贡献者活跃度排行榜

| 排名 | 贡献者 | 级别 | 类型 | 近30天贡献 | 总PR | 总Issue | 趋势 |
|------|--------|------|------|-----------|------|---------|------|
| 1 | {{login}} | 🔥 | 代码 | {{d30}} 天 | {{pr_count}} | {{issue_count}} | ↑ |
| ... | ... | ... | ... | ... | ... | ... | ... |

---

## 三、重点贡献者分析

> 仅展示核心/活跃贡献者。

### 🔥 {{login}}（核心贡献者）

| 维度 | 数据 | 说明 |
|------|------|------|
| 最近 30 天贡献 | {{d30}} 天 | {{评价}} |
| 总 PR 数 | {{pr_count}} | |
| 总 Issue 数 | {{issue_count}} | |
| 贡献趋势 | {{trend_direction}} | {{trend_comment}} |

---

## 四、团队健康度评估

### 健康度指标

| 指标 | 状态 | 说明 |
|------|------|------|
| 核心贡献者占比 | {{core_ratio}}% | {{core_comment}} |
| 新老比例 | {{new_old_ratio}} | {{new_old_comment}} |
| 贡献频率稳定性 | {{stability}} | {{stability_comment}} |
| 知识分散度 | {{bus_factor}} | {{bus_factor_comment}} |

### 风险提示

<!-- 根据分析结果，从以下列表中选择匹配的风险项输出 -->

- ⚠️ **核心贡献者不足**（当 core_count < 3 时）：仅 {{core_count}} 位核心贡献者，存在单点依赖风险（Bus Factor = {{core_count}}）。
- ⚠️ **贡献者流失**（当 dormant_rate > 50% 时）：超过一半的贡献者已不活跃，需要关注社区留存。
- ⚠️ **缺少新鲜血液**（当 new_count == 0 时）：近期无新兴贡献者，建议通过 Good First Issue 等方式吸引新人。
- ✅ **团队健康**（当以上情况均不满足时）：贡献者结构合理，团队运转良好。

> 指标计算：
> - `core_ratio` = core_count / analyzed_count × 100
> - `dormant_rate` = dormant_count / analyzed_count × 100
> - `active_30d_rate` = (近30天至少一次贡献的人数) / analyzed_count × 100
> - `new_old_ratio`：新兴贡献者数 : 核心+活跃贡献者数 的比值
> - `bus_factor` = core_count（简化定义：核心贡献者数量最低值）
> - `stability`：判断标准为"贡献标准差"（各月贡献量波动小=高稳定性，波动大=低稳定性）

---

## 五、社区建设建议

1. **激励核心贡献者**：{{核心贡献者维护建议}}
2. **激活休眠贡献者**：{{休眠贡献者召回建议}}
3. **吸引新贡献者**：{{新贡献者吸引建议}}
4. **平衡贡献类型**：{{贡献类型平衡建议}}
```

---

## 异常场景处理

| 场景 | 处理方式 |
|------|----------|
| `repo +contributors` 返回空 | 标注"仓库暂无贡献者数据"，仅从 `repo +info` 获取 `contributor_users_count` |
| `user +heatmap` 返回空 | 标注"无热力图数据"，评分仅基于 stats 和 trends |
| `user +stats` / `+trends` 返回错误 | 跳过该维度，标注"数据不可用" |
| 贡献者 > 15 人 | 仅分析贡献量最高的前 10 位，报告中注明"基于 Top 10 分析" |

---

## 注意事项

- ✅ **所有命令使用 `--format json`**，确保可解析
- ✅ **本 Skill 为纯只读分析**，不会修改任何仓库
- ✅ **Owner/repo 优先从 `git remote` 自动解析**，无 git 上下文时询问用户
- ⚠️ **每人 3 次 API 调用**（heatmap + stats + trends），10 人即 30 次，注意控制分析人数
- ⚠️ **热力图数据可能稀疏**：部分贡献者数据不完整，标注"数据有限"
- ⚠️ **数据仅反映 GitLink 平台活动**：不包括 GitHub 或其他平台的数据
