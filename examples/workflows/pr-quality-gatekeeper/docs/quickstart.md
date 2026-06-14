# 最短复现路径（3 步）

## 1. 安装与认证

```bash
npm install -g @gitlink-ai/cli      # ≥0.2.0（label 命令与 gitlink-gatekeeper Skill 已内置）
gitlink-cli auth login              # 或 export GITLINK_TOKEN=<私人令牌>
gitlink-cli auth status             # 确认已登录
```

## 2. 对任意真实 PR 出评分卡（dry-run，零写入）

在本目录（`examples/workflows/pr-quality-gatekeeper/`）下：

```bash
python3 scripts/gatekeeper_workflow.py \
  --owner Gitlink --repo gitlink-cli --pr <PR号> \
  --policy ../../../skills/gitlink-gatekeeper/examples/gatekeeper.yaml \
  --owner-rules owner-rules.example.yaml \
  --output-dir outputs
```

产物：`outputs/<owner>_<repo>_pr<id>_scorecard.md`（评分卡）+ `_summary.json`（结构化摘要）。
退出码：`0` = PASS/COMMENT，`2` = REQUEST_CHANGES（可直接当 CI 门禁用），`1` = 运行错误。

不带 `--policy` 也能跑（脚本内置同值默认策略）；想看含扣分的评分卡，加 `--findings findings.example.json`。

## 3. 可选进阶

- **真实回写**（评论 + 裁决标签 + tracking issue）：加 `--apply`。请先在自有 fork 演练；自动裁决只用建议性 `common` 评论，绝不替人 approve/reject，绝不自动合并。
- **全仓体检**（只读批扫全部 open PR，出治理报告）：

```bash
python3 scripts/gatekeeper_sweep.py \
  --owner Gitlink --repo gitlink-cli \
  --policy ../../../skills/gitlink-gatekeeper/examples/gatekeeper.yaml \
  --owner-rules owner-rules.example.yaml \
  --output-dir sweep-out --date-label $(date +%F)
```

- **CI 接入**：见 [`../ci-example/`](../ci-example/)（Gitea Actions，PR 触发自动门禁）。
- **改门禁松紧**：复制一份 `gatekeeper.yaml` 改 `weights/hard_gates/thresholds`，字段说明见 [Skill REFERENCE](../../../../skills/gitlink-gatekeeper/REFERENCE.md)。

## 验证自己改动没破坏确定性

```bash
python3 tests/test_scoring.py    # 同输入 → 同分 → 同裁决 的回归护栏
```
