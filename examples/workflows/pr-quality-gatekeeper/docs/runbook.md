# 运行手册 — PR 看门人闭环

本手册覆盖 `scripts/gatekeeper_workflow.py` 的前置条件、运行步骤、参数、预期输出与回滚。数值/字段以已收录的 [`gitlink-gatekeeper` Skill REFERENCE](../../../../skills/gitlink-gatekeeper/REFERENCE.md) 为准。

## 1. 前置条件

- 已安装 `gitlink-cli` 且在 `PATH` 中（或用 `--cli-bin` 指定路径）。
- 已完成登录：`gitlink-cli auth login`（Token 有效期 7 天，过期重新登录；详见 [gitlink-shared](../../../../skills/gitlink-shared/SKILL.md)）。
- 对目标仓库有读权限；要 `--apply` 回写评论/标签/建 issue 时需写权限。
- Python 3.8+（脚本纯标准库，无需 `pip install`）。

验证登录态：

```bash
gitlink-cli auth status
gitlink-cli pr +view -i <pr_id> --owner <owner> --repo <repo> --format json   # 确认目标 PR 可读
```

## 2. 配置

编辑 [`../config.example.yaml`](../config.example.yaml)（或复制一份），填好 `owner`/`repo`/`pr` 与策略、路由表路径。命令行参数会覆盖配置同名字段，相对路径以配置文件所在目录为基准。

按需调整 [`../owner-rules.example.yaml`](../owner-rules.example.yaml)：把占位 reviewer 用户名替换成本仓库维护者，按「具体规则在前」排列 glob。

按需选择策略预设（均在 [`../../../../skills/gitlink-gatekeeper/examples/`](../../../../skills/gitlink-gatekeeper/examples/)）：
- `gatekeeper.yaml`：均衡基线（= SSOT 内置默认）。
- `gatekeeper.strict.yaml`：严格预设。
- `gatekeeper.lenient.yaml`：宽松预设。
- 不指定 `--policy` 且配置无 `policy` 字段时，回退脚本内置默认策略（与 `gatekeeper.yaml` 等价）。

## 3. 运行步骤

### 步骤 A：dry-run 预览（安全默认，必做）

```bash
python3 scripts/gatekeeper_workflow.py --config config.example.yaml --pr <pr_id>
```

此模式**不写任何东西**，只采集 + 评分 + 打印将要执行的写命令 + 落盘本地产物。先看评分卡与计划是否符合预期。

### 步骤 B：注入 AI 审查发现（可选）

`review_findings` 维度默认按 0 发现计分。若已有 AI 代码审查（如 `gitlink-code-review` Skill）产出，整理成 JSON 注入：

```json
{ "findings": [
  { "severity": "blocker", "message": "硬编码密钥", "file": "internal/auth/refresh.go", "line": 12 },
  { "severity": "minor",   "message": "缺超时上下文", "file": "internal/auth/handler.go", "line": 40 }
] }
```

```bash
python3 scripts/gatekeeper_workflow.py --config config.example.yaml --pr <pr_id> --findings findings.json
```

`severity` 取值：`blocker` / `major` / `minor` / `nit`（其余忽略）。

### 步骤 C：apply 执行写操作

确认 dry-run 计划无误后，加 `--apply`：

```bash
python3 scripts/gatekeeper_workflow.py --config config.example.yaml --pr <pr_id> --apply
```

将依次执行（按裁决）：回写评分卡评论 → 确保裁决标签存在 →（仅 REQUEST_CHANGES）创建 tracking issue。
**合并不会自动发生**：仅当策略 `behavior.auto_merge: true` 且裁决为 `PASS` 且本次带 `--apply` 时，才追加 `pr +merge`。默认 `auto_merge: false`。

## 4. 参数速查

| 参数 | 说明 | 默认 |
|------|------|------|
| `--config` | 工作流配置 YAML（owner/repo/pr/policy/owner_rules/findings） | 无 |
| `--owner` / `--repo` / `--pr` | 覆盖配置中的目标 | 取自 config |
| `--policy` | `gatekeeper.yaml` 路径 | 内置默认策略 |
| `--owner-rules` | `owner-rules.yaml` 路径 | 取自 config |
| `--findings` | AI 审查发现 JSON | 空（0 发现） |
| `--cli-bin` | `gitlink-cli` 可执行路径 | `gitlink-cli` |
| `--skip-ci` | 跳过 CI 采集（`ci_status` 记 `unknown`） | 否 |
| `--output-dir` | 本地产物目录 | `outputs` |
| `--apply` | **执行写操作**；不传则仅预览 | 否（dry-run） |

## 5. 预期输出

- 终端：三段进度（路由 / 裁决 / 回写）+ 评分概览 + 计划或执行结果 + 最终裁决。
- 文件：
  - `outputs/<owner>_<repo>_pr<id>_scorecard.md` — 评分卡（SSOT 第 6 节模板）。
  - `outputs/<owner>_<repo>_pr<id>_summary.json` — 结构化摘要（路由、各维得分、硬门禁、裁决、`planned_writes`、`executed`、产物路径）。
- 退出码：`PASS`/`COMMENT` → `0`；`REQUEST_CHANGES` → `2`（可作 CI 门禁）；可预期错误（缺配置 / 未登录 / CLI 缺失）→ `1`。

样例评分卡见 [`../../../../skills/gitlink-gatekeeper/examples/scorecard-sample.md`](../../../../skills/gitlink-gatekeeper/examples/scorecard-sample.md)。

## 6. 回滚

dry-run 不产生任何远端副作用，无需回滚（本地产物可直接删 `outputs/`）。

`--apply` 后如需撤销：

| 已做的写操作 | 回滚方式 |
|--------------|----------|
| 回写的评分卡评论 | 评论走 issue journals，在 PR 页面手动删除该评论即可；脚本不提供删除命令（避免误删他人评论） |
| 创建的裁决标签定义 | `gitlink-cli label +delete -i <label_id> --owner <o> --repo <r>`（先 `label +list` 查 id） |
| 创建的 tracking issue | `gitlink-cli issue +close -n <number> --owner <o> --repo <r>`（关闭而非删除，保留审计痕迹） |
| 已合并的 PR | **不可自动回滚**。这也是默认 `auto_merge: false` 的原因；合并前务必人工确认。如确需撤销，按仓库常规流程 revert commit |

> 安全提示：任何 `--apply` 写操作前，脚本会在 dry-run 计划里完整复述将执行的命令。生产仓库建议先 dry-run，再 `--apply`。
