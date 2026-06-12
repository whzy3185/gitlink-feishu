# 架构与数据流 — PR 看门人闭环

本工作流采用「**采集 → 路由 → 裁决 → 回写/善后**」四段式流水线，把 `gitlink-gatekeeper` 的 Policy-as-Code 门禁包成一条可复现闭环。所有数值/字段/算法以已收录的 [`gitlink-gatekeeper` Skill REFERENCE](../../../../skills/gitlink-gatekeeper/REFERENCE.md) 为准。

## 设计目标

- **可复现**：同策略 + 同 PR 上下文 → 同评分卡 + 同裁决（确定性算法，SSOT 第 3–5 节）。
- **可审计**：评分卡逐维列分 + 备注，硬门禁逐条列出命中原因，裁决可追溯到具体规则与 `gatekeeper.yaml` 版本。
- **安全默认**：默认 dry-run，写操作需显式 `--apply`；绝不默认自动合并（SSOT 第 8 节）。
- **低门槛**：仅依赖 `gitlink-cli` 与 Python 标准库（含内置 YAML 子集解析器，无第三方包）。
- **边界清晰**：采集、路由、裁决、回写四段各自独立，便于单测与替换（如换一套评分维度只动裁决段）。

## ASCII 流程图

> 下面是数据流占位图：左侧为 `gitlink-cli` 采集，中间为本脚本的确定性处理，右侧为回写/善后的写操作（仅 `--apply` 时执行）。

```
                        ┌──────────────────────────── gatekeeper_workflow.py ────────────────────────────┐
                        │                                                                                 │
   gitlink-cli (读)     │   step 1: 路由                step 2: 裁决              step 3: 回写 + 善后        │   gitlink-cli (写, 仅 --apply)
 ─────────────────────  │  ───────────────────       ──────────────────      ─────────────────────────  │  ─────────────────────────────
                        │                                                                                 │
  pr +view  ──┐         │                          ┌─ review_findings(40) ─┐                              │
  pr +files ──┼──▶ 采集 │  changed_files            │  test_coverage (20)   │   verdict                    │   pr +comment  ─▶ 评分卡评论
  ci +builds  │   归一化 │     │                     │  pr_hygiene    (15)   │──▶ ── PASS ───────┐           │
  api .../    │         │     ▼                     │  commit_quality(15)   │   ── COMMENT ───┐ │           │   label +create ─▶ 裁决标签
   commits  ──┘         │  owner-rules.yaml         │  ci_status     (10)   │   ── REQ_CHG ─┐ │ │           │      (+ 挂 issue_tag_ids
        ▲               │  (glob → reviewer)        └───────────┬───────────┘             │ │ │           │       via Raw API POST
        │               │     │                                 ▼                         │ │ │           │       /:owner/:repo/issues/:id)
   gatekeeper.yaml ─────┼─────┼──────────────▶  hard_gates 判定 ─┴─▶ total 0..100 ─▶ 阈值  │ │ │           │
   (Policy-as-Code)     │     ▼                  (SSOT 第4节)        (SSOT 第3节)  (第5节) │ │ │           │   issue +create ─▶ tracking issue
        │               │  suggested_reviewers ──────────────────────────────────────────┘ │ │           │      (仅 REQUEST_CHANGES)
   findings.json ───────┼──▶ review_findings 注入                                            │ │           │
   (可选, AI 审查)       │                                                                    │ │           │   pr +merge ─▶ 合并 (受限:
                        │                            ┌── outputs/*_scorecard.md ◀────────────┘ │           │      PASS + auto_merge + --apply)
                        │  本地产物落盘 (总是) ───────┤                                          │           │
                        │                            └── outputs/*_summary.json ◀──────────────┘           │
                        └─────────────────────────────────────────────────────────────────────────────────┘

   dry-run（默认）：右侧写操作仅打印「将要执行的命令」，不实际调用 → 安全。
   --apply        ：右侧写操作真正执行；其中合并需同时满足 PASS + 策略 auto_merge=true + --apply。
```

## 四段职责

### ① 采集（collect_pr_context）
调只读 `gitlink-cli` 命令拿到 PR 元信息、变更文件、CI 状态、commits（端点未开放时降级，不阻断）。输出统一归一化为内部结构，兼容 GitLink Envelope 的多种字段名。

### ② 路由（route_reviewers）
读 `owner-rules.yaml`，对每个变更文件按 glob 顺序匹配（首个命中生效，顺序即优先级），产出 `reviewer → 文件清单`；未命中文件归 `default_reviewers`。结果写进评分卡的「Suggested reviewers」分区。**只产出建议，不调用任何写操作**——是否真正分配由维护者决定。

### ③ 裁决（score_dimensions / evaluate_hard_gates / decide_verdict）
- 五维加权评分（权重和=100，SSOT 第 3 节），可选注入 AI findings 影响 `review_findings`。
- 硬门禁逐项判定（SSOT 第 4 节），任一命中即 `hard_gate_failed`。
- 裁决判定树（SSOT 第 5 节）：硬门禁失败 → REQUEST_CHANGES；否则按总分与 `pass`/`request_changes` 阈值落三态。
- 渲染评分卡（SSOT 第 6 节模板）。

### ④ 回写 + 善后（build_*_command + execute_write）
按裁决构造写操作计划：评分卡评论、裁决标签、（REQUEST_CHANGES 时）tracking issue、（受限）合并。dry-run 只打印计划；`--apply` 才逐条执行并记录结果到 `summary.json`。

## 为什么选这条链路

子赛题三要求用现有命令 / Skill 组合形成完整解决方案，且串联不少于 3 步。本链路：
1. 串联了 **4 个只读采集命令** + **最多 4 个写命令**，远超 3 步下限。
2. 形成从「数据获取」到「治理动作落地」的端到端闭环，并能接入 CI（REQUEST_CHANGES 返回码 2）。
3. 复用本作品自研的 `label` 命令组（子赛题一）与 gatekeeper 策略（子赛题二），三个子赛题在同一作品内闭环，相互增强。
