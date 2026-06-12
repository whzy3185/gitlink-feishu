# 真实平台验证记录

> 全部针对 **GitLink 线上真实平台** 运行（`gitlink-cli` + Token 认证），非 mock。
> 他人仓库一律 dry-run（只读零写入）；写操作只在自有 fork 演练。
> 运行环境：macOS（Apple Silicon）· Python 3.9 · `@gitlink-ai/cli` 0.2.0（npm 官方发布版，零本地构建）。

## A. dry-run：真实 PR → PASS 90/100

对本仓库真实 PR（`pull_request_id 15222`，feat/org-team-projects，8 个变更文件）：

- 路由正确分流：README/docs/skill → doc-maintainer；`org.go` → go-reviewer；`org_test.go` → qa-reviewer
- 评分（确定性，可手算复现）：review 40/40 · test 20/20（1 src/1 test）· hygiene 10/15（desc✓/issue✗/size✓）· commit 15/15 · ci 5/10（unknown）= **90 → PASS**
- CI 取不到构建记录 → `unknown`：按策略**不触发**硬门禁（仅显式 `failing` 触发），只在 CI 维记半分
- 产物：[`../examples/demo-outputs/scorecard-pass-90.md`](../examples/demo-outputs/scorecard-pass-90.md)

## B. 注入真实审查发现 → REQUEST_CHANGES 55/100

同一 PR，注入 [`../findings.example.json`](../findings.example.json) 重跑：review_findings 40/40 → 5/40（1 major + 2 minor），总分 90 → 55，**裁决翻转为 REQUEST_CHANGES**。

**发现是真的，不是编的**——三条均来自对该 PR 真实 diff（head `bcc27bf`）的代码审查，标注 `shortcuts/org/org.go` 真实行号，任何人拉取该分支可逐条复核。其中 major：新增的 `parseBool` 只认字面 `"true"`，`--dry-run=1` 会被静默当 false，而该 flag 守护的是「批量移除团队全部项目」这一破坏性操作。

产物：[`../examples/demo-outputs/scorecard-findings-55.md`](../examples/demo-outputs/scorecard-findings-55.md)

## C. `--apply` 真实回写（自有 fork 演练）

在自有 fork 的演练 PR（故意「改源码不带测试」）上执行 `--apply`：

- 触发硬门禁 `require_tests_for_src_changes` → REQUEST_CHANGES 40/100
- GitLink API 回执确认三件写操作全部落地：
  1. 评分卡评论回写到 PR（comment id `472741`）
  2. 自动创建 tracking issue（id `143217`），汇总硬门禁 + 必修项 + 建议 reviewer，与 PR 双向回链
  3. 裁决标签挂载到 PR 背后 issue（`label +list` 查 id → Raw API `POST /:owner/:repo/issues/<issue_id>`）——依赖本仓库的 `label` 命令（0.2.0 起官方发布版自带）

## D. 全仓批量体检：113 个 open PR

`gatekeeper_sweep.py` 对本仓库**全部 113 个 open PR** 逐个 dry-run（只读、零写入、零 AI 成本），113/113 成功：

- 裁决分布：**PASS 105 · COMMENT 6 · REQUEST_CHANGES 2**；分数 min 70 / 中位 90 / 均值 88.5 / max 95
- 治理洞察：**96% 的 open PR 未关联 issue**；2 个 PR 触发 `require_tests_for_src_changes`（改源码不带测试）
- 完整报告（含全量明细表）：[`../examples/demo-outputs/sweep-report-2026-06-10.md`](../examples/demo-outputs/sweep-report-2026-06-10.md)
- 诚实口径：批扫不注入审查发现（review_findings 维未评、按满分计），CI 统一 `--skip-ci`（unknown 半分）——总分代表「除人工/AI 审查外的工程卫生分」，偏乐观

## E. 单元测试（确定性回归护栏）

```bash
$ python3 tests/test_scoring.py
OK
```

锁定四个权威裁决案例（PASS / REQUEST_CHANGES / COMMENT / 硬门禁直拒）的**总分与裁决**与 Skill 文档逐位一致；任何改动若破坏「同输入 → 同分 → 同裁决」，测试立即变红。

## 真实运行当场暴露过的问题（透明记录）

- GitLink 的 PR 标题/描述在 `pr +view` 返回的 `issue.subject/description`，而非 `pull_request` 子对象——离线 mock 测不到，真实平台运行才暴露并修复。
- npm 0.1.18 时代 `--apply` 的打标签步骤会报 `unknown command "label"`（彼时 `label` 命令尚未发布）；0.2.0 起官方发布版自带，整条闭环零本地构建跑通。
