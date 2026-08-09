# 新手友好度评分规则

本技能用一套可解释的规则为每个 Issue 打出 0-100 的新手友好度分，不依赖大模型。

## 评分模型

基准分 **50**，在此基础上根据信号加减：

| 信号 | 分值 | 说明 |
|------|:----:|------|
| 命中新手友好标签 | +35 | `good first issue` / `beginner` / `新手` / `easy` 等 |
| 命中高难度标签 | -30 | `hard` / `complex` / `advanced` / `困难` 等 |
| 命中易上手关键词 | +7/词（上限 +20） | `typo` / `docs` / `文档` / `test` / `翻译` 等 |
| 命中高难度关键词 | -10/词（上限 -25） | `refactor` / `架构` / `并发` / `性能` / `安全` 等 |
| 描述长度适中（30-600） | +5 | 适中描述更易上手 |
| 描述过长（>1500） | -8 | 往往较复杂 |
| 讨论过多（评论 >15） | -8 | 可能存在分歧 |

最终分数裁剪到 [0, 100]。

## 难度等级

| 友好度分 | 难度等级 |
|:--------:|:--------:|
| ≥ 75 | 入门 |
| 55 - 74 | 较易 |
| 40 - 54 | 中等 |
| < 40 | 进阶 |

友好度 ≥ 55 的 Issue 被列为新手候选（`is_good_first = true`）。

## 信号词表（节选）

**新手友好标签**：good first issue、good-first-issue、first-timers-only、beginner、beginner-friendly、easy、starter、新手、新手友好、入门、简单

**易上手关键词**：typo、document、docs、readme、comment、translation、rename、format、lint、test、example、i18n、文档、注释、拼写、翻译、示例、格式

**高难度关键词**：refactor、architecture、performance、concurrency、race、security、deadlock、memory leak、breaking change、重构、架构、性能、并发、安全、死锁、内存

## 可调整性

词表与阈值集中在 `scripts/newcomer.py` 顶部常量（`GOOD_FIRST_LABELS` / `EASY_KEYWORDS` / `HARD_KEYWORDS` / `HARD_LABELS`），可按项目习惯调整。
