---
name: gitlink-research-progress
version: 1.0.0
description: "科研进度智能跟踪与预警：监控科研仓库的提交节奏、Issue 解决进度、里程碑完成度、PR 吞吐与贡献者活跃度，自动生成《科研进度周报 + 风险预警》，用红/黄/绿灯标记停滞与风险。当课题组/PI/导师需要掌握科研项目进展、识别延期与停滞风险、生成阶段汇报时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-progress（科研进度智能跟踪与预警）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能为只读分析型；如需把预警落成 Issue/评论，须先确认用户意图。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能概述

本技能面向**课题组负责人（PI）、导师与科研团队**，回答：**「这个（些）科研项目现在进展如何？哪里有延期或停滞风险？」**

它把科研仓库的协作数据转化为**进度与风险信号**，自动产出一份可直接用于组会/阶段汇报的**《科研进度周报 + 风险预警》**。

### 五维进度信号

| 信号 | 衡量 | 风险方向 |
|------|------|----------|
| 1. 提交节奏（Velocity） | 近 N 周提交频率与趋势 | 提交骤降 = 停滞 |
| 2. Issue 进度（Issue Burndown） | 开/闭 Issue 数、解决速度、陈旧 Issue | 开口持续扩大 = 失控 |
| 3. 里程碑完成度（Milestone） | 里程碑到期/完成比例 | 临期未完成 = 延期 |
| 4. PR 吞吐（Throughput） | 开放/合并 PR、停留时长 | PR 长期挂起 = 阻塞 |
| 5. 团队活跃与 Bus Factor | 活跃贡献者数、贡献集中度 | 单点依赖 = 高风险 |

---

## 一、确定监控范围与采集元数据

### 1.1 单仓库或多仓库

```bash
# 单个科研仓库
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 课题组/组织名下多个仓库（逐个跟踪后汇总）
gitlink-cli repo +list --user <login> --format json
gitlink-cli repo +list --category manage --format json
```

记录 `created_at`、`updated_at`、`default_branch`、`open_issues_count`（若有）。

---

## 二、五维信号采集

### 2.1 信号一 · 提交节奏（Velocity）

```bash
# 取提交列表（按时间），用于统计近 N 周提交分布
gitlink-cli api GET "/:owner/:repo/commits?page=1&limit=50" --format json
```

统计：
- 近 4 周 / 8 周每周提交数 → 画 ASCII 趋势条
- 与上一周期环比（↑/↓/持平）
- 最近一次提交距今天数（`days_since_last_commit`）

**风险规则：**
```
🔴 最近一次提交 > 30 天   或   近 2 周提交为 0
🟡 周提交量环比下降 > 50%
🟢 提交稳定或上升
```

### 2.2 信号二 · Issue 进度（Burndown）

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open   --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json
```

统计：
- 开放 / 已关闭 Issue 数；近 4 周新增 vs 关闭（净增 = 新增 − 关闭）
- 陈旧 Issue：开放且 `updated_at` 超过 30/60 天
- 平均解决时长（closed 的 created→closed 估算）

**风险规则：**
```
🔴 近 4 周净增 Issue 持续为正且 > 5   或   陈旧 Issue 占比 > 50%
🟡 净增为正但可控
🟢 净增 ≤ 0（在收敛）
```

### 2.3 信号三 · 里程碑完成度（Milestone）

```bash
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
```

统计每个里程碑：`due_date`、开放/关闭 Issue 数、完成率 = 已关闭 / 总数。

**风险规则：**
```
🔴 里程碑已过期（today > due_date）且完成率 < 80%
🟡 距到期 < 14 天且完成率 < 60%
🟢 按期推进
```

### 2.4 信号四 · PR 吞吐（Throughput）

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open   --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --format json
```

统计：开放 PR 数、近 4 周合并 PR 数、开放 PR 平均停留天数、是否有 PR 挂起 > 14 天。

**风险规则：**
```
🔴 存在开放 PR 停留 > 30 天   或   近 4 周 0 合并但有活跃开发
🟡 PR 平均停留 14–30 天
🟢 PR 流转顺畅
```

### 2.5 信号五 · 团队活跃与 Bus Factor

```bash
# 主数据源：直接返回每人的 contribution_perc（贡献占比），是 Bus Factor 的可靠依据
gitlink-cli repo +contributors      --owner <owner> --repo <repo> --format json

# 可选补充（按代码行）：部分仓库后端会返回 [-1] 失败，属正常，失败则忽略、以上面为准
gitlink-cli repo +contributor-stats --owner <owner> --repo <repo> --format json
```

统计：
- 活跃贡献者数（结合提交记录判断近 90 天是否活跃）
- 贡献集中度：取 `repo +contributors` 返回的 Top1 `contribution_perc` → **Bus Factor 估计**（占比越高，单点风险越大）

**风险规则：**
```
🔴 Bus Factor = 1（Top1 占比 > 80%）   或   活跃贡献者 = 1
🟡 Top1 占比 60–80%
🟢 贡献相对分散
```

---

## 三、进度健康度评级

### 3.1 综合灯

将五个信号的红/黄/绿汇总为**项目总灯**：

```
🔴 红灯（高风险）：任一信号为 🔴，或 ≥3 个 🟡
🟡 黄灯（需关注）：1–2 个 🟡，无 🔴
🟢 绿灯（健康）：全部 🟢
```

### 3.2 提交节奏 ASCII 趋势

```
近 8 周周提交量：
W-7 ████████        12
W-6 ██████          9
W-5 █████████       14
W-4 ███             4
W-3 ██              3
W-2 █               1
W-1 ▏               0    ← 🔴 提交骤降
W-0 ▏               0
```

---

## 四、《科研进度周报 + 风险预警》报告模板

```markdown
## 📈 科研进度周报与风险预警

**项目：** <owner>/<repo>
**统计周期：** 2026-06-09 ~ 2026-06-15（近 8 周趋势）
**生成时间：** 2026-06-15
**项目总灯：** 🟡 黄灯（需关注）

---

### 一、一句话结论

> 核心算法仓库提交节奏明显放缓（近 2 周仅 1 次提交），里程碑 M2 距到期 10 天但完成率 55%，建议本周组会重点对齐。

### 二、五维信号面板

| 信号 | 指标 | 灯 |
|------|------|----|
| 提交节奏 | 近 2 周 1 次提交，环比 ↓78% | 🔴 |
| Issue 进度 | 开放 14 / 已闭 36，近 4 周净增 +3 | 🟡 |
| 里程碑 | M2 完成率 55%，距到期 10 天 | 🟡 |
| PR 吞吐 | 开放 2（最久 9 天），近 4 周合并 5 | 🟢 |
| 团队 / Bus Factor | 活跃 3 人，Top1 占比 64% | 🟡 |

### 三、提交节奏趋势

（插入第 3.2 节 ASCII 趋势图）

### 四、风险预警清单

| 等级 | 风险 | 证据 | 建议 |
|------|------|------|------|
| 🔴 | 主仓库近 2 周近乎停滞 | 最近提交 11 天前 | 组会确认是否受阻/缺人 |
| 🟡 | M2 里程碑有延期苗头 | 完成率 55%，10 天到期 | 砍范围或顺延，更新 due_date |
| 🟡 | 存在单点依赖 | Top1 提交占比 64% | 安排第二人熟悉核心模块 |
| 🟡 | 陈旧 Issue 累积 | 5 个 Issue >30 天未动 | 分诊：关闭/重排期 |

### 五、本周建议动作（Top 3）

1. 排查主仓库停滞原因（环境/卡点/人力），必要时拆任务。
2. 重新评估 M2 范围与到期日，更新里程碑。
3. 指定核心模块第二负责人，降低 Bus Factor。
```

---

## 五、执行步骤总览

```bash
# Step 1：确定范围
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# Step 2：提交节奏
gitlink-cli api GET "/:owner/:repo/commits?page=1&limit=50" --format json

# Step 3：Issue 进度
gitlink-cli issue +list --owner <owner> --repo <repo> --state open   --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json

# Step 4：里程碑
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json

# Step 5：PR 吞吐
gitlink-cli pr +list --owner <owner> --repo <repo> --state open   --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --format json

# Step 6：团队活跃 / Bus Factor（以 +contributors 为主，其 contribution_perc 即 Bus Factor 依据）
gitlink-cli repo +contributors      --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributor-stats --owner <owner> --repo <repo> --format json   # 可选，失败可忽略

# Step 7：AI 按第二节规则打灯、第四节模板出周报
```

---

## 注意事项

- ✅ **纯只读**：默认不写任何内容；落成 Issue/评论需用户明确同意。
- ✅ **多仓库汇总**：课题组多仓库时，逐仓采集后再出一张"组级总览表"（每行一个仓库 + 总灯）。
- ⚠️ **时间字段**：以 `created_at` / `updated_at` / commit date 估算周期；注意时区与"X 天前"等相对时间需换算。
- ⚠️ **分页**：Issue/PR/commit 可能分页，统计趋势时关注 `meta.total_count` 并按需翻页（趋势可只取近若干页近似）。
- ⚠️ **里程碑可选**：部分科研仓库不用里程碑，则该信号标"不适用"，不计入总灯。
- ⚠️ **contributor-stats 容错**：`repo +contributor-stats`（按代码行）在部分仓库会返回 `[-1]` 失败，属正常；Bus Factor 以 `repo +contributors` 的 `contribution_perc` 为准即可。
- ✅ **可定期运行**：建议每周固定时间运行，形成可对比的趋势序列；最终产出为 Markdown 周报。
