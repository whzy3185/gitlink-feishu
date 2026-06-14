# PR 质量门禁工作流（pr-quality-gatekeeper）

把已收录的 [`gitlink-gatekeeper` Skill](../../../skills/gitlink-gatekeeper/SKILL.md)（Policy-as-Code 合并门禁）包成**可直接运行的端到端工作流**：

> **采集 → 路由 → 裁决 → 回写/善后**：读取一个真实 PR 的元信息/变更文件/commits/CI，按变更路径建议 reviewer，依 `gatekeeper.yaml` 策略算出**确定性 0–100 评分卡**与**三态裁决**（PASS / REQUEST_CHANGES / COMMENT），并（仅在 `--apply` 时）把评分卡评论、裁决标签、tracking issue 真实回写到 GitLink。

与仓库内已有能力的关系：`label` 命令（裁决标签）→ `gitlink-gatekeeper` Skill（裁决知识）→ **本工作流（可复现闭环）**，三层共用同一套策略文件，互为支撑而非重复。

## 交付物

- `scripts/gatekeeper_workflow.py`：单 PR 门禁闭环（纯标准库，Python ≥3.9，零第三方依赖）
- `scripts/gatekeeper_sweep.py`：**仓库级批量体检**——对全部 open PR 逐个 dry-run，产出治理报告
- `owner-rules.example.yaml`：变更路径 → reviewer 的路由表样例
- `config.example.yaml`：工作流配置样例（命令行参数可覆盖）
- `findings.example.json`：AI/人工审查发现注入样例（**来自对真实 PR diff 的真实审查**，行号可复核）
- `docs/architecture.md` · `docs/quickstart.md` · `docs/runbook.md` · `docs/verification.md`
- `ci-example/`：Gitea Actions 接入示例（PR 触发自动门禁，退出码 2 = REQUEST_CHANGES）
- `examples/demo-outputs/`：真实平台运行产物（PASS 90 评分卡 / 注入发现后的 55 分评分卡 / 113 个 open PR 的全仓体检报告）
- `tests/test_scoring.py`：确定性回归护栏（同输入 → 同分 → 同裁决）

## 快速运行（默认 dry-run，不写远端）

```bash
npm install -g @gitlink-ai/cli      # ≥0.2.0，自带 label 命令与 gitlink-gatekeeper Skill
gitlink-cli auth login

python3 scripts/gatekeeper_workflow.py \
  --owner <owner> --repo <repo> --pr <PR号> \
  --policy ../../../skills/gitlink-gatekeeper/examples/gatekeeper.yaml \
  --owner-rules owner-rules.example.yaml \
  --output-dir outputs
```

- 注入审查发现得到含扣分的评分卡：加 `--findings findings.example.json`
- 真实回写（评论 + 标签 + tracking issue）：加 `--apply`（请先在自有仓库演练）
- 全仓批量体检（只读，零写入）：

```bash
python3 scripts/gatekeeper_sweep.py \
  --owner <owner> --repo <repo> \
  --policy ../../../skills/gitlink-gatekeeper/examples/gatekeeper.yaml \
  --owner-rules owner-rules.example.yaml \
  --output-dir sweep-out --date-label $(date +%F)
```

更多见 [`docs/quickstart.md`](docs/quickstart.md) 与 [`docs/runbook.md`](docs/runbook.md)。

## 已在真实平台验证

全部证据见 [`docs/verification.md`](docs/verification.md)，要点：

| 验证 | 对象 | 结果 |
|------|------|------|
| dry-run | 本仓库真实 PR（pull_request_id 15222） | ✅ PASS 90/100，8 个变更文件路由正确 |
| 注入真实审查发现 | 同一 PR + `findings.example.json` | ❌ REQUEST_CHANGES 55/100（裁决翻转，确定性可复算） |
| `--apply` 真实回写 | 自有 fork 的演练 PR | 评分卡评论 + tracking issue + 裁决标签全部由 API 回执确认 |
| **全仓批量体检** | 本仓库**全部 113 个 open PR** | 113/113 成功：PASS 105 / COMMENT 6 / REQUEST_CHANGES 2，均分 88.5；96% 未关联 issue |
| 单测 | `tests/test_scoring.py` | 全绿（锁定四个权威裁决案例的分值与裁决） |

## 设计要点

- **确定性评分**：AI 只负责产出「发现列表」（可选注入），扣分与裁决由纯函数完成——同策略 + 同 PR → 同裁决，可逐位手算复现、可审计。
- **安全默认**：默认 dry-run 什么都不写；即便策略开了 `auto_merge`，也必须 `verdict == PASS` 且显式 `--apply` 才会合并；强语义的 approve/reject 始终留给人，自动裁决只以建议性 `common` 评论 + 标签呈现。
- **原生适配 GitLink**：PR 标题/描述取自 `pr +view` 的 `issue.subject/description`；标签挂载走「`label +list` 查 id → Raw API `POST /:owner/:repo/issues/<issue_id>`」；尊重 `common/approved/rejected` 三态 review。
- **零依赖、零常驻**：纯标准库脚本 + `gitlink-cli`，无需部署 webhook 服务或数据库，CI 一条 step 即可接入（见 `ci-example/`）；确定性意味着**大规模治理零 AI 成本**。

## 许可证

随仓库 [MulanPSL-2.0](../../../LICENSE)。
