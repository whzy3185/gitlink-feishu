# community-ops-sweep —— 社区运营增量对账 sweep

## 概述

一个 [Claude Code Workflow](../../../README.md) 实现的**增量对账式**社区运营自动化：
运行时给定一个 `since` 时间点（或默认读上次 checkpoint），处理该时间点后的所有**新增 Issue** 与**合并 PR**，按需更新社区周报与 Release Notes。

**一次调用、时间窗界定、5 项事件一网打尽**：R1 新 Issue triage → R2 建议 owner → R3 周报 → R4 Release Notes → R5 PR→Issue 关联闭环。

> 完整设计见 [`docs/superpowers/specs/2026-07-01-community-ops-sweep-design.md`](/docs/superpowers/specs/2026-07-01-community-ops-sweep-design.md)

## 架构

三层，JSON 文件交接（判断与计算分离）：

```
CC Workflow (JS) ……… 编排：按 phase 调度
   ├── LLM agent 层 …… 只做判断：分类 / PR 关联 / owner 建议 / 散文增强
   │       ↕ JSON 文件
   └── Python 引擎 …… 只做计算：采集 / 归一化 / id 解析 / 合并 / 守卫 / 渲染 / 写回
```

确定性引擎（`scripts/community_ops_sweep.py`）复用 `community-ops-automation` 的采集/归一化/渲染管线，真实字段名实测自 `baoerjun/gitlink-cli`。

## 前置条件

- **gitlink-cli** ≥ 0.2.0（`npm install -g @gitlink-ai/cli`）
- **Python** 3.9+（标准库，无第三方依赖）
- **gitlink-cli auth login**（认证）
- **Claude Code**（运行 workflow）

## 运行方式

### 在 Claude Code 中

```bash
# 作为 slash 命令（若已放入 .claude/workflows/）
/community-ops-sweep owner=baoerjun repo=gitlink-cli

# 或通过 Workflow 工具
# 带入参数：since 可选（ISO 时间或留空读 .last-sweep）、apply 默认 false
```

### 单独跑确定性引擎

```bash
# collect：采集 since 后的新 issue + 合并 PR
python scripts/community_ops_sweep.py collect --owner baoerjun --repo gitlink-cli --since 2026-06-01T00:00:00Z

# plan：读 candidates + LLM decisions → 写计划
python scripts/community_ops_sweep.py plan --candidates candidates.json --triage triage.json --owners owners.json --links links.json

# apply：dry-run 预览，--apply 才真写
python scripts/community_ops_sweep.py apply --plan plan.json --owner baoerjun --repo gitlink-cli --apply

# checkpoint：推进 .last-sweep + 追加 triage-log
python scripts/community_ops_sweep.py checkpoint --owner baoerjun --repo gitlink-cli
```

### 跑单测

```bash
python3 tests/test_sweep.py    # 18 个确定性逻辑回归
```

## 文件布局

```
community-ops-sweep/
  community-ops-sweep.wf.js           # CC workflow (JS)
  scripts/community_ops_sweep.py      # 确定性引擎
  routing-rules.example.yaml          # R2 路由规则（opt-in auto_assign）
  tests/test_sweep.py                 # 引擎单测
  README.md
```

## 安全

- 默认 **dry-run**，`--apply` 才写
- 标签**整体替换**发期望全集（增删统一，不丢标签）
- 关 issue 前自动守卫（已关跳过）
- **绝不自动合并 PR**
- 可逆写（关可 reopen、release 可 edit）→ 无二次确认
