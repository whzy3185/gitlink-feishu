---
name: gitlink-gatekeeper
version: 1.0.0
description: "Policy-as-Code 的 PR 合并门禁：按版本化的 gatekeeper.yaml 策略对一个 Pull Request 聚合多路信号、算出 0-100 评分卡、给出三态裁决（PASS / REQUEST_CHANGES / COMMENT），并把结论作为结构化评论回写、打标签。当用户需要判断 PR 能否合并、做合并门禁/质量闸门、生成可复现的 PR 裁决，或问『这个 PR 达标了吗』时触发。默认 dry-run，绝不自动合并。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli pr --help"
---

# gitlink-gatekeeper（Policy-as-Code PR 合并门禁）

**CRITICAL — 开始前必须先阅读 [`gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 GitLink API 注意事项（PR list state 过滤不准、主分支是 master、issue 更新需带回 subject/description 等）。**
**CRITICAL — 任何写操作（回写评论、打标签、合并）前，必须向用户复述将要做什么并得到确认；默认 dry-run 不写任何东西。绝不默认自动合并。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数；本 Skill 的全字段策略与算法细节见同目录 [`REFERENCE.md`](./REFERENCE.md)，故障排查见 [`TROUBLESHOOTING.md`](./TROUBLESHOOTING.md)。

---

## 定位与产物

gitlink-gatekeeper 是一个**可复现的 PR 合并门禁**：团队把合并标准写进版本化的 `gatekeeper.yaml`，gatekeeper 据此对一个 PR 算出**透明的 0-100 评分卡**，给出**三态裁决**，并把结论回写。与 `gitlink-code-review`（只产出主观评论、无阈值、无结论）相比，本 Skill 的核心是**可裁决、可审计、可复现**：同一策略 + 同一 PR → 同一裁决。

| 裁决 | emoji | 触发条件 | 回写标签 |
|------|:----:|----------|----------|
| PASS | ✅ | 无硬门禁失败 且 `total ≥ thresholds.pass` | `gatekeeper:pass` |
| REQUEST_CHANGES | ❌ | 命中任一硬门禁，或 `total < thresholds.request_changes` | `gatekeeper:needs-changes` |
| COMMENT | 💬 | 介于两阈值之间且无硬门禁失败 | `gatekeeper:review` |

> GitLink 原生 PR review 的 `status` 为 `common`/`approved`/`rejected`（`rejected` 即「请求修改」的等价）。gitlink-gatekeeper **刻意把所有自动裁决以建议性的 `common` 评论回写**（评分卡标题明确标注三态裁决），把强语义的 `approved`/`rejected` 留给人工授意；裁决状态通过标签（`gatekeeper:needs-changes` 等）承载——这正是本作品复用 `label` 命令组的原因。

---

## 工作流概览

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| ① 加载策略 | 读 `gatekeeper.yaml`，找不到则回退内置默认策略 | 解析 / 校验 / 回退 |
| ② 采集上下文 | 拉 PR 元信息、分支可合并性、变更文件、diff、commits、CI 状态 | 执行 CLI 命令采集数据 |
| ③ 产出发现 | 逐文件审查，按 severity 分级标记问题 | AI 分析，输出发现列表 |
| ④ 逐维评分 | 五维各算 `0..weight` 得分，相加得 `total` | 确定性计算（非主观） |
| ⑤ 硬门禁 | 逐项判定 `hard_gates`，命中即拦截 | 布尔判定 |
| ⑥ 裁决 | 由硬门禁 + total + 阈值得出三态 | 确定性计算 |
| ⑦ 渲染评分卡 | 按模板渲染 Markdown 评分卡 | 模板填充 |
| ⑧ 回写（受 `--apply`） | dry-run 仅打印；`--apply` 才回写评论 + 打标签 | 写操作，需确认 |
| ⑨ 安全规则 | 默认 dry-run，绝不自动合并 | 守门 |

---

## 详细工作流

### Step ① 加载策略 gatekeeper.yaml

策略默认从**仓库根目录** `gatekeeper.yaml` 读取，可用 `--policy <path>` 指定。找不到时**回退到下文内置默认策略**（务必在评分卡 footer 注明用的是哪个策略来源）。

加载后做两项校验（不通过则报错并停止）：
- `weights` 五项之和必须 **= 100**（`review_findings + test_coverage + pr_hygiene + commit_quality + ci_status`）。
- `thresholds.pass`、`thresholds.request_changes` 为 0-100 整数，且 `pass ≥ request_changes`。

**内置默认策略**（与 `examples/gatekeeper.yaml` 一致，回退时使用，policy 标记为 `<built-in default>@v1`）：

```yaml
version: 1
weights:
  review_findings: 40
  test_coverage:   20
  pr_hygiene:      15
  commit_quality:  15
  ci_status:       10
hard_gates:
  forbid_blocker_findings: true
  require_ci_pass: true
  require_tests_for_src_changes: true
  require_linked_issue: false
  max_changed_files: 80
severity_penalty: { blocker: 100, major: 25, minor: 5, nit: 1 }
thresholds: { pass: 85, request_changes: 60 }
labels:
  pass: "gatekeeper:pass"
  request_changes: "gatekeeper:needs-changes"
  comment: "gatekeeper:review"
source_globs: ["**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.rs", "**/*.java"]
test_globs:   ["**/*_test.go", "**/test_*.py", "**/*.test.*", "**/*.spec.*", "tests/**"]
behavior:
  dry_run_default: true
  post_comment: true
  apply_label: true
  auto_merge: false
  merge_method: squash
```

### Step ② 采集 PR 上下文（CLI 命令映射）

用以下命令采集一个 PR 的全部输入。**始终带 `--format json` 便于解析**；`--owner`/`--repo` 在 git 仓库目录下可自动从 remote 解析，否则显式传入。

| 步骤 | 数据 | 命令 |
|------|------|------|
| 聚合上下文 | 仓库、PR、文件、Review、Issue、标签 | `gitlink-cli workflow +review-context --number <id> --format json` |
| PR 元信息 | 标题/描述/作者/关联 issue | `gitlink-cli pr +view -i <id> --format json` |
| 合并准备状态 | 源/目标分支是否可合并 | `gitlink-cli pr +check-can-merge --head <head> --base <base> --dry-run`，确认后 `--yes` |
| 变更文件 | 文件路径列表 | `gitlink-cli pr +files -i <id> --format json` |
| Diff | 变更内容供 AI 审查 | `gitlink-cli pr +diff -i <id> --format json` |
| CI 状态 | 构建结果 | `gitlink-cli ci +builds --format json` |

> 实测注意：优先使用 `workflow +review-context` 获取聚合上下文；如需更细的 diff 再补充 `pr +diff`。CI 通过/失败需从 builds 返回的 `status` 字段判断；**无 build 记录时按「CI 未知」处理**（见 §3.5）。

```bash
PR=42
gitlink-cli workflow +review-context --number "$PR" --format json
gitlink-cli pr +view  -i "$PR" --format json   # title / body / 关联 issue
gitlink-cli pr +check-can-merge --head feature/pr --base master --dry-run
gitlink-cli pr +files -i "$PR" --format json   # changed files
gitlink-cli pr +diff  -i "$PR" --format json   # diff（供 AI 审查）
gitlink-cli ci +builds --format json
```

### Step ③ AI 按 severity 分级产出发现

对 diff 逐文件审查（检查项参考 `gitlink-code-review`：安全红线、错误处理、并发、资源管理、魔法数字、测试覆盖等），产出**结构化发现列表**，每条带：

```
{ severity: blocker|major|minor|nit, file: "<path>", line: <n>, message: "<问题描述>" }
```

severity 判定准则（与 §3.1 扣分表对应）：

| severity | 含义 | 典型例子 |
|----------|------|----------|
| `blocker` | 必须拦截，单条即清零本维度并触发硬门禁 | 硬编码密钥/Token、SQL/命令注入、路径遍历、不安全反序列化、XSS |
| `major` | 严重缺陷，强烈建议修 | 未处理错误/异常吞掉、资源泄漏、并发竞态、明显逻辑错误 |
| `minor` | 一般问题 | 风格偏离、轻微复杂度、缺注释、魔法数字 |
| `nit` | 吹毛求疵 | 命名小瑕疵、格式建议 |

> 控制信噪比：宁可少而准。同时记录 `Strengths`（做得好的点），用于评分卡 `✅ Strengths` 区。

### Step ④ 按 SSOT 算法逐维评分（确定性）

每维产出 `0..weights[dim]`，五维相加得 `total ∈ 0..100`。**这是确定性计算，不是再次主观打分**——同输入必同分。

**4.1 review_findings（默认 40）**
```
penalty = Σ severity_penalty[finding.severity]          # 对所有发现求和
score   = weights.review_findings * max(0, 1 - penalty / weights.review_findings)
```
扣分累计达本维度权重即扣到 0；blocker 单条（penalty=100）即清零本维度。

**4.2 test_coverage（默认 20）**
```
changed_src   = 变更文件中匹配 source_globs 的数量
changed_tests = 变更文件中匹配 test_globs 的数量
if changed_src == 0:        score = weights.test_coverage      # 无源码改动，不扣
elif changed_tests == 0:    score = 0
else: ratio = min(1, changed_tests / changed_src)
      score = round(weights.test_coverage * (0.5 + 0.5*ratio)) # 有测试至少拿一半
```

**4.3 pr_hygiene（默认 15，三项各占 1/3）**
- 描述非空且长度 ≥ 30 字符 → +1/3
- 关联了 Issue（PR body 含 `#<n>` 或 API 标记）→ +1/3
- 体量适中（`changed_files ≤ max_changed_files/2`）→ +1/3；超过一半但未超上限 → +1/6
```
score = round(weights.pr_hygiene * 命中比例)
```

**4.4 commit_quality（默认 15）**
```
conforming = 符合 Conventional Commits (type(scope): subject) 的 commit 数
total_c    = commit 总数
score      = round(weights.commit_quality * conforming / total_c)   # total_c=0 给满分
```

**4.5 ci_status（默认 10）**
```
CI 通过 → weights.ci_status
CI 失败 → 0
CI 未知/无 → round(weights.ci_status * 0.5)
```

### Step ⑤ 硬门禁评估

按 `hard_gates` 逐项判定，命中任一 → `hard_gate_failed = true`（记下命中的门禁名供评分卡 `⛔ Hard gate failures` 区）：

| 门禁 | 命中条件 |
|------|----------|
| `forbid_blocker_findings` | 为 true 且存在 blocker 发现 |
| `require_ci_pass` | 为 true 且 CI **明确失败**（`failing`）；`unknown`/无 build 记录**不触发**本门禁 |
| `require_tests_for_src_changes` | 为 true 且 `changed_src > 0` 且 `changed_tests == 0` |
| `require_linked_issue` | 为 true 且未关联 Issue |
| `max_changed_files` | `> 0` 且 `changed_files > max_changed_files` |

### Step ⑥ 裁决计算

```
if hard_gate_failed:                        verdict = REQUEST_CHANGES
elif total >= thresholds.pass:              verdict = PASS
elif total <  thresholds.request_changes:   verdict = REQUEST_CHANGES
else:                                       verdict = COMMENT
```

### Step ⑦ 渲染评分卡（回写到 PR 的 Markdown）

按下列模板填充（emoji：PASS=✅，REQUEST_CHANGES=❌，COMMENT=💬）。无硬门禁失败时省略 `⛔` 区；各发现区无内容时省略。

```markdown
## 🛡️ Gatekeeper Report — PR #<id> <title>

**Verdict: <emoji> <PASS|REQUEST_CHANGES|COMMENT>**  ·  Score: <total>/100  ·  policy: <policy_path>@v<version>

| Dimension | Weight | Score | Notes |
|-----------|:------:|:-----:|-------|
| Review findings | 40 | <s>/40 | <n blocker / n major / n minor / n nit> |
| Test coverage   | 20 | <s>/20 | <changed_src> src / <changed_tests> test files |
| PR hygiene      | 15 | <s>/15 | <desc / linked issue / size 命中情况> |
| Commit quality  | 15 | <s>/15 | <conforming>/<total> conventional |
| CI status       | 10 | <s>/10 | <passing/failing/unknown> |

### ⛔ Hard gate failures (<n>)
- `<gate>`: <说明>

### 🔴 Must fix (<n>)
- [<severity>] <message> — <file>:<line>

### 🟡 Should fix (<n>)
- ...

### 🔵 Nits (<n>)

### ✅ Strengths
- <做得好的点>

### Next steps
1. <按裁决给出的最高优先级行动>
---
*Generated by gitlink-gatekeeper · policy-as-code PR gate · re-run after changes*
```

### Step ⑧ 回写：dry-run（默认）vs --apply

**默认 dry-run**：不传 `--apply` 时，只把评分卡**打印给用户**，不回写、不打标签、不合并。
**`--apply` 才写**：写操作前**先向用户复述**「将向 PR #<id> 回写评分卡评论 + 打标签 `<label>`」，确认后执行。

回写评论（二选一，优先用 `pr +review` 因其自带 `--dry-run` 预览）：

```bash
# 方式 A：作为 PR review 回写（status 用 common，即 GitLink 的 COMMENT 等价）
gitlink-cli pr +review -i "$PR" --status common --content "$(cat scorecard.md)"

# 方式 B：作为普通 PR 评论回写
gitlink-cli pr +comment -i "$PR" --body "$(cat scorecard.md)"
```

> ⚠️ `pr +review --status` 只接受 `common` / `approved` / `rejected`。评分卡评论一律用 `common`（不要因为裁决=PASS 就 `approved`，APPROVE 是更强的批准语义，需用户显式授意）。裁决信息已写在评分卡标题里，状态由标签承载。

打标签（GitLink 无「直接给 PR 挂标签」的命令，标签挂在 PR 背后的 Issue 上）：

```bash
# 1) 确保裁决对应的标签存在（首次需创建；已存在则跳过）
gitlink-cli label +list --format json                      # 查现有标签拿 id
gitlink-cli label +create --name "gatekeeper:needs-changes" --color "#D73A4A" \
  --description "PR gate: changes requested"                # 不存在才创建

# 2) 取 PR 背后的 issue id（pr +view 返回里有 issue 对象）
ISSUE_ID=$(gitlink-cli pr +view -i "$PR" --format json | jq -r '.data.issue.id')

# 3) 通过 Issue 更新挂标签（必须带回当前 subject/description，否则会被清空——见 gitlink-shared）
gitlink-cli api POST /:owner/:repo/issues/$ISSUE_ID --body '{
  "issue_tag_ids": [<tag_id>],
  "done_ratio": 0,
  "subject": "<原始标题>",
  "description": "<原始描述>"
}'
```

### Step ⑨ 安全规则（硬性，不可绕过）

- **默认 dry-run**：不传 `--apply` 一律只打印，不产生任何写副作用。
- **绝不默认自动合并**：`behavior.auto_merge` 默认 `false`；即便策略里设为 `true`，也必须**同时**满足 `verdict == PASS` **且**命令显式带 `--apply` 才允许合并，且合并前再次向用户复述确认。
- 合并命令（仅在上述全部条件满足时）：
  ```bash
  # merge_method 来自策略 behavior.merge_method（merge|rebase|squash）
  gitlink-cli pr +merge -i "$PR" --method squash
  ```
  > 注意 CLI 标志是 `--method`（底层 API 字段才是 `do`）。
- 不回显 Token；遵循 `gitlink-shared/SKILL.md` 的认证与 API 注意事项（401 引导重登、403 查权限）。
- REQUEST_CHANGES 永不触发合并；COMMENT/PASS 默认也不合并，除非满足自动合并三条件。

---

## 完整示例

### 示例 1：dry-run（默认，安全，不写任何东西）

```bash
# 在目标仓库目录下，对 PR #42 跑门禁，仅预览评分卡
PR=42
gitlink-cli pr +view  -i "$PR" --format json
gitlink-cli pr +check-can-merge --head feature/pr --base master --dry-run
gitlink-cli pr +files -i "$PR" --format json
gitlink-cli pr +diff  -i "$PR" --format json
gitlink-cli workflow +review-context --number "$PR" --format json
gitlink-cli ci +builds --format json
# → AI 产出发现 → 按 §4 评分 → §5 硬门禁 → §6 裁决 → §7 渲染评分卡
# → dry-run：仅把评分卡打印给用户，结尾提示「如需回写到 PR，请加 --apply」
```

预期产出（节选）：
```
**Verdict: ❌ REQUEST_CHANGES**  ·  Score: 58/100  ·  policy: gatekeeper.yaml@v1
⛔ Hard gate failures (1)
- require_tests_for_src_changes: 改了 3 个源码文件但没有新增测试
（dry-run：未回写、未打标签、未合并）
```

### 示例 2：--apply（用户确认后回写评论 + 打标签）

```bash
PR=42
# …(同上采集 + 评分，得到裁决=REQUEST_CHANGES，渲染出 scorecard.md)…

# 写操作前向用户复述：将向 PR #42 回写评分卡评论 + 打标签 gatekeeper:needs-changes
# 用户确认后：

# 1) 回写评分卡（review 形式，自带 dry-run 可先预演）
gitlink-cli pr +review -i "$PR" --status common --dry-run --content "$(cat scorecard.md)"  # 预演
gitlink-cli pr +review -i "$PR" --status common          --content "$(cat scorecard.md)"  # 实际回写

# 2) 打标签 gatekeeper:needs-changes（按 Step ⑧ 取/建 tag_id 与 issue_id 后）
gitlink-cli api POST /:owner/:repo/issues/$ISSUE_ID --body '{
  "issue_tag_ids": [101], "done_ratio": 0,
  "subject": "feat: add rate limiter", "description": "<原始描述>"
}'

# 注意：裁决=REQUEST_CHANGES → 绝不合并。
# 即便裁决=PASS，也只有在 policy.auto_merge=true 且本次显式 --apply 时才允许：
#   gitlink-cli pr +merge -i "$PR" --method squash
```

---

## 策略预设

`examples/` 提供三套可直接 `--policy` 引用的预设：

| 预设 | 特点 | 适用 |
|------|------|------|
| `gatekeeper.yaml` | 默认平衡策略（pass 85 / rc 60） | 大多数仓库 |
| `gatekeeper.strict.yaml` | 高阈值、`require_linked_issue: true`、更小 `max_changed_files` | 核心库 / 发布分支 |
| `gatekeeper.lenient.yaml` | 低阈值、关掉部分硬门禁 | 早期项目 / 文档仓库 |

```bash
gitlink-cli pr +view -i 42 --format json   # 采集后用严格策略评分（评分逻辑同上，仅阈值/门禁不同）
# 评分时加载：--policy examples/gatekeeper.strict.yaml
```

---

## 最佳实践

1. **先 dry-run 再 `--apply`**：永远先看评分卡内容，确认无误再回写。
2. **策略进版本库**：把 `gatekeeper.yaml` 提交到仓库，让裁决标准对所有贡献者透明、可审计。
3. **可复现优先**：评分是确定性的——若两次裁决不同，先查是不是 PR 内容或策略变了，而非「AI 心情」。
4. **控制发现数量**：最严重的 3-5 条比 20 条琐碎问题更有价值；nit 折叠展示。
5. **大 PR 分段处理**：`pr +diff` 输出可能很大，Agent 应分段读取 diff 再汇总发现。
6. **标签复用**：同一仓库的三个 `gatekeeper:*` 标签建一次即可，后续只更新挂载关系。

## 注意事项

- PR review/评论提交后会通知关注该 PR 的参与者，评分卡内容保持专业、可操作。
- `--state` 过滤 PR 列表不精确，需用 `pull_request_status` 字段客户端判断（0=open,1=merged,2=closed）。
- GitLink 主分支是 `master`（非 `main`）；合并方式由策略 `merge_method` 决定。
- 对 draft PR 应提示用户先标记为 Ready for Review 再门禁。
- 端到端「PR 看门人闭环」（路由建议 reviewer → 裁决 → 回写 + 为 REQUEST_CHANGES 自动建 tracking issue）的完整工作流形态见配套独立仓库 recorder/gitlink-gatekeeper；策略字段与算法见 `REFERENCE.md`。
