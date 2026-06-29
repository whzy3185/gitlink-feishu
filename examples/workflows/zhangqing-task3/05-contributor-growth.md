# 贡献者成长排行榜 — ylly/gitlink-cli

> 工作流⑤（贡献者成长体系）产物 · 数据源：git log + issue +list + pr +list（merged）
> 生成时间：2026-06-29 · 数据范围：仓库全历史（已过滤测试号 15972095207 / 2403_89190320 / fsafasff）

## 综合排行榜

| 排名 | 贡献者 | Commits | 合并 PR | Issue | 综合积分 |
|:----:|--------|:-------:|:-------:|:-----:|:--------:|
| 1 | wbtiger | 42 | 0 | 0 | 42 |
| 2 | ylly | 5 | 0 | 14 | 33 |
| 3 | ZxR (ZxR123-Z) | 10 | 2 | 2 | 24 |
| 4 | whzy | 9 | 0 | 0 | 9 |
| 5 | wangyue789 | 8 | 0 | 0 | 8 |
| 6 | wbavon | 8 | 0 | 0 | 8 |
| 7 | Tiger | 5 | 0 | 0 | 5 |
| 8 | zhangqing | 4 | 0 | 0 | 4 |
| 9 | Mengz | 3 | 0 | 0 | 3 |
| 10 | Jiachen Li | 2 | 0 | 0 | 2 |
| 11 | Leo77 | 2 | 0 | 0 | 2 |
| 12 | yangsai01 | 2 | 0 | 0 | 2 |
| 13 | Zhang Jinnan | 1 | 0 | 0 | 1 |
| 14 | ljc0426 | 1 | 0 | 0 | 1 |

> 积分公式：commits×1 + 合并PR×5 + Issue×2（PR 权重高，因合并工作量大）

## 徽章授予方案

| 徽章 | 授予标准 | 获得者 |
|------|---------|--------|
| ⭐ 星级贡献者 | commits ≥ 10 或 合并 PR ≥ 2 | wbtiger, ZxR (ZxR123-Z) |
| 🔥 活跃贡献者 | commits ≥ 5 或 Issue ≥ 5 | ylly, whzy, wangyue789, wbavon, Tiger |
| 🌱 贡献者 | 有任意提交 | zhangqing, Mengz, Jiachen Li, Leo77, yangsai01, Zhang Jinnan, ljc0426 |

## 颁奖动作清单（待执行，写入操作需用户确认）

**Step 1: 创建徽章 label**（仓库现无这些 label）
- label +create --name "星级贡献者" --color #FFD700
- label +create --name "活跃贡献者" --color #FF6B35
- label +create --name "贡献者" --color #87C95F

**Step 2: 给 Top 贡献者发祝贺 Issue 评论**
对 wbtiger / ZxR（ZxR123-Z）等 Top 贡献者相关的 Issue 添加祝贺评论。
（GitLink 无"用户主页评论"能力，改在 Issue 评论或新建颁奖 Issue 演示）

**Step 3: 可选 — 新建颁奖 Issue**
创建一个 Issue「🏆 v0.2.0-beta.1 贡献者排行榜公布」，body 含本排行榜，用 @mention 提及 Top 贡献者，并打上「星级贡献者」label。

---

## Skill 蓝图对照（按 SKILL.md 行号）

### 取数阶段 ← `gitlink-insight/SKILL.md` 工作流 3（贡献者洞察）

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 贡献者列表 | 第 173-183 行 采集数据 | `contributors` + merged PR + user info | ✅ commits 走 `git log`（contributors API 返回 HTML，降级方案）+ merged PR + Issue 作者聚合 |
| 角色与活跃度 | 第 185-203 行 输出格式 | 排行 + 活跃度分布 | ✅ 综合积分公式 commits×1+PR×5+Issue×2，按贡献分层 |
| 三层分布 | 第 194-197 行 | 核心/活跃/新增 | ✅ 对应「星级/活跃/贡献者」三级徽章 |

### 发奖阶段 ← `gitlink-issue-triage/SKILL.md` 工作流 1 + 通用 label 操作

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 查现有标签 | 第 80-84 行 Step 5 | `label +list` 命中则复用 | ✅ 仓库原无徽章 label，3 个均新建 |
| 创建徽章 | 第 86-89 行 | `label +create --name --color` | ✅ 星级 394180 / 活跃 394181 / 贡献者 394182 |
| 颁奖 Issue | 第 101-117 行报告格式 | 表格 + @mention | ✅ Issue #17 已建并打「星级贡献者」label |

### Skill 串联数

| 步骤 | 使用的 Skill/命令 | 角色 |
|------|------------------|------|
| 1 | git log + issue +list + pr +list（insight 工作流 3 数据源） | 取数 |
| 2 | label +create（issue-triage 工作流 1 Step 5） | 建徽章 |
| 3 | issue +create + issue +update --label（issue-triage 工作流 1 Step 6） | 颁奖 |

**3 步串联，满足 PDF「≥3 命令/Skill 串联」要求。**