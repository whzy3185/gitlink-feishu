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