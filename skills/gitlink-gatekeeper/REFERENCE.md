# gitlink-gatekeeper REFERENCE

> 查字段 / 查算法的参考手册。工作流与裁决步骤见 [`SKILL.md`](./SKILL.md)，权威定义见 [`./REFERENCE.md`](./REFERENCE.md)（SSOT）。
> 本文与 SSOT 不一致时，**以 SSOT 为准**。所有数值、字段名、公式均逐条对齐 SSOT 第 2–7 节。

**CRITICAL — 认证、全局参数、GitLink 真实 API 坑见 [`gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — GitLink 平台只用 `gitlink-cli`，禁止 `gh`。**

---

## 1. `gatekeeper.yaml` 全字段参考

策略默认从仓库根目录 `gatekeeper.yaml` 读取，可用 `--policy <path>` 指定；找不到时回退到内置默认策略（即下表「默认值」列）。

### 1.1 顶层

| 字段 | 类型 | 默认值 | 含义 | 约束 |
|------|------|--------|------|------|
| `version` | int | `1` | 策略 schema 版本 | 当前固定为 `1` |
| `weights` | map | 见 §1.2 | 五维加权评分权重 | 五项之和 **必须 = 100** |
| `hard_gates` | map | 见 §1.3 | 硬门禁开关 | — |
| `severity_penalty` | map | 见 §1.4 | 严重度 → 扣分 | 仅用于 `review_findings` 维度 |
| `thresholds` | map | 见 §1.5 | 总分 → 裁决阈值 | `pass ≥ request_changes`，均 ∈ 0..100 |
| `labels` | map | 见 §1.6 | 三态裁决回写的标签名 | — |
| `source_globs` | list[str] | 见 §1.7 | 源码文件判定 glob | 用于 `test_coverage` / 测试硬门禁 |
| `test_globs` | list[str] | 见 §1.7 | 测试文件判定 glob | 同上 |
| `behavior` | map | 见 §1.8 | 行为开关 | — |

### 1.2 `weights`（五项之和必须 = 100）

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `review_findings` | int | `40` | AI 代码审查发现（按严重度扣分） |
| `test_coverage` | int | `20` | 改动的源码是否伴随测试 |
| `pr_hygiene` | int | `15` | PR 描述 / 关联 Issue / 体量 |
| `commit_quality` | int | `15` | commit 是否符合 Conventional Commits |
| `ci_status` | int | `10` | CI 是否通过 |

> 校验：`Σweights == 100`。不等于 100 应报错并拒绝运行（避免总分不再是 0–100）。

### 1.3 `hard_gates`（任一命中 → 直接 `REQUEST_CHANGES`，无视总分）

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `forbid_blocker_findings` | bool | `true` | 出现 blocker 级发现即拦截 |
| `require_ci_pass` | bool | `true` | CI 未通过即拦截 |
| `require_tests_for_src_changes` | bool | `true` | 改了源码却没加测试即拦截 |
| `require_linked_issue` | bool | `false` | PR 是否必须关联 Issue |
| `max_changed_files` | int | `80` | 超过该改动文件数即拦截；**`0` 表示不限** |

### 1.4 `severity_penalty`（用于 `review_findings`）

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `blocker` | int | `100` | 单条 blocker 即可把本维度清零（≥ 权重 40） |
| `major` | int | `25` | |
| `minor` | int | `5` | |
| `nit` | int | `1` | |

> `finding.severity` 取值域：`{blocker, major, minor, nit}`。未知严重度不应计入（视为 0 或报错），由 SKILL 侧约束 AI 只产出这四类。

### 1.5 `thresholds`

| 字段 | 类型 | 默认值 | 含义 | 约束 |
|------|------|--------|------|------|
| `pass` | int | `85` | 总分 ≥ `pass` 且无硬门禁失败 → `PASS` | 0..100 |
| `request_changes` | int | `60` | 总分 < `request_changes` → `REQUEST_CHANGES` | 0..100，且 `≤ pass` |

> 介于 `[request_changes, pass)` 之间 → `COMMENT`。

### 1.6 `labels`（依赖 `gitlink-cli label`）

| 字段 | 类型 | 默认值 | 对应裁决 |
|------|------|--------|----------|
| `pass` | str | `"gatekeeper:pass"` | `PASS` |
| `request_changes` | str | `"gatekeeper:needs-changes"` | `REQUEST_CHANGES` |
| `comment` | str | `"gatekeeper:review"` | `COMMENT` |

### 1.7 `source_globs` / `test_globs`

| 字段 | 类型 | 默认值 |
|------|------|--------|
| `source_globs` | list[str] | `["**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.rs", "**/*.java"]` |
| `test_globs` | list[str] | `["**/*_test.go", "**/test_*.py", "**/*.test.*", "**/*.spec.*", "tests/**"]` |

> 一个文件先按 `test_globs` 命中算测试；否则按 `source_globs` 命中算源码。`test_globs` 优先，避免 `foo_test.go` 同时被 `**/*.go` 计为源码。

### 1.8 `behavior`

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `dry_run_default` | bool | `true` | 默认只预览，不写任何东西（不传 `--apply` 即生效） |
| `post_comment` | bool | `true` | 把评分卡作为评论回写 PR |
| `apply_label` | bool | `true` | 按裁决打标签 |
| `auto_merge` | bool | `false` | **仅当 `true` 且裁决=PASS 且显式 `--apply` 时才合并** |
| `merge_method` | enum | `squash` | `merge` \| `rebase` \| `squash` |

### 1.9 `advisory_flags`（可选 · 咨询标记 · 默认全关）

与硬门禁互补的「软」一层。硬门禁是真牙齿（命中即 `REQUEST_CHANGES`）；咨询标记**只在评分卡中给出建议性提示，不参与评分、不改变裁决**，把治理建议留给人工判断。默认关闭、**向后兼容**（策略无此段时等同关闭，旧行为不变）；**确定性**——仅消费已采集的 PR 数据，不依赖当前时间等非确定性输入，与评分主逻辑同样可复算。

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `enabled` | bool | `false` | 总开关，默认关闭 |
| `large_pr_files` | int | `0` | 变更文件数 ≥ 此值则提示拆分（`0`=不启用；建议 `< hard_gates.max_changed_files`，使其落在「偏大但未触硬门禁」区间） |

> 典型用途：在 PR 积压的活跃仓库里，对「超体量但未触硬门禁」的 PR 提示拆分而**不阻断**——既给出治理信号，又不把判断权从人手里夺走。命中的标记渲染在评分卡的 `### 💡 Advisory` 小节，并写入 `summary.json` 的 `advisory_flags` 字段。

---

## 2. 评分算法规格（确定性，可复现）

输入：一个 PR 的上下文（变更文件、diff、commits、CI 状态、PR 元信息）+ AI 审查产出的**发现列表**（每条带 `severity ∈ {blocker, major, minor, nit}`、`file`、`line`、`message`）。

每个维度产出 `0..weights[dim]` 的得分，五维相加得 `total ∈ 0..100`。同策略 + 同输入 → 同 total → 同裁决。

记号：`W.x` = `weights.x`，`SP[s]` = `severity_penalty[s]`。

### 2.1 `review_findings`（默认权重 40）

```
penalty = Σ over all findings  SP[finding.severity]
score   = W.review_findings * max(0, 1 - penalty / W.review_findings)
```

- 累计扣分达到该维度权重即扣到 0（不为负）。
- **边界（无发现）**：`penalty = 0 → score = W.review_findings`（满分）。
- **边界（含 blocker）**：单条 blocker 默认 `SP=100 ≥ 40`，本维度即清零；通常同时触发 `forbid_blocker_findings` 硬门禁。

### 2.2 `test_coverage`（默认权重 20）

```
changed_src   = 变更文件中匹配 source_globs 的数量
changed_tests = 变更文件中匹配 test_globs   的数量

if   changed_src == 0:     score = W.test_coverage          # 无源码改动，不扣，满分
elif changed_tests == 0:   score = 0                        # 改了源码但零测试
else:                      ratio = min(1, changed_tests / changed_src)
                           score = round(W.test_coverage * (0.5 + 0.5*ratio))
```

- **边界（无源码改动）**：纯文档 / 配置 PR → 满分（不应因没测试被罚）。
- 有测试即至少拿一半分；测试文件数 ≥ 源码文件数 → `ratio=1` → 满分。
- `round` 为四舍五入到整数。

### 2.3 `pr_hygiene`（默认权重 15，三项各占 1/3）

逐项累加命中比例 `hit ∈ {0, 1/6, 1/3, ...}`：

| 子项 | 命中条件 | 贡献 |
|------|----------|------|
| 描述 | PR 描述非空且长度 ≥ 30 字符 | +1/3 |
| 关联 Issue | PR body 含 `#<n>` 或 API 标记关联了 Issue | +1/3 |
| 体量 | `changed_files ≤ max_changed_files / 2` → +1/3；`max_changed_files/2 < changed_files ≤ max_changed_files` → +1/6；超上限 → +0 | +1/3 或 +1/6 或 0 |

```
score = round(W.pr_hygiene * hit_ratio)        # hit_ratio = 三项贡献之和
```

- **边界（`max_changed_files == 0` 表示不限）**：体量子项视为满足，记 +1/3。
- 体量子项与 `max_changed_files` 硬门禁独立计算；硬门禁只看是否 `> max_changed_files`。

### 2.4 `commit_quality`（默认权重 15）

```
conforming = 符合 Conventional Commits（type(scope): subject）的 commit 数
total      = commit 总数

if total == 0:  score = W.commit_quality                    # 满分
else:           score = round(W.commit_quality * conforming / total)
```

- **边界（`total == 0`）**：取不到 commit（API 空 / 异常）时给满分，不因数据缺失惩罚。
- Conventional Commits 判定：`type(optional-scope): subject`，常见 `type` ∈ {feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert}（`scope` 可省略）。

### 2.5 `ci_status`（默认权重 10）

```
CI 通过      → score = W.ci_status
CI 失败      → score = 0
CI 未知 / 无 → score = round(W.ci_status * 0.5)
```

- **边界（CI 未知/无）**：拿一半分，避免对没有 CI 的仓库一刀切清零。
- 注意与硬门禁的区别：`require_ci_pass` 只在 CI **明确失败**时拦截；CI 未知不触发硬门禁，但本维度仅得半分。

---

## 3. 硬门禁判定表（SSOT §4）

按 `hard_gates` 逐项判定，命中任一 → `hard_gate_failed = true`：

| 门禁字段 | 触发条件 | 失败说明（写入评分卡） |
|----------|----------|------------------------|
| `forbid_blocker_findings` | `true` 且存在 ≥1 条 blocker 发现 | 存在 blocker 级问题 |
| `require_ci_pass` | `true` 且 CI **未通过**（失败；未知不触发） | CI 未通过 |
| `require_tests_for_src_changes` | `true` 且 `changed_src > 0` 且 `changed_tests == 0` | 改了源码但无测试 |
| `require_linked_issue` | `true` 且 PR 未关联 Issue | 未关联 Issue |
| `max_changed_files` | `> 0` 且 `changed_files > max_changed_files` | 改动文件数超上限 |

> `max_changed_files == 0` → 该门禁直接跳过（不限）。多项命中时全部记入评分卡的 Hard gate failures 区块。

---

## 4. 裁决伪代码（SSOT §5）

```python
def decide(total, hard_gate_failed, thresholds):
    if hard_gate_failed:
        return "REQUEST_CHANGES"          # 硬门禁优先，无视总分
    if total >= thresholds.pass:
        return "PASS"
    if total < thresholds.request_changes:
        return "REQUEST_CHANGES"
    return "COMMENT"                        # total ∈ [request_changes, pass)
```

裁决映射（用于评分卡标题与回写）：

| verdict | emoji | 回写标签（默认） | 是否允许合并 |
|---------|:-----:|------------------|--------------|
| `PASS` | ✅ | `gatekeeper:pass` | 仅当 `behavior.auto_merge==true` 且显式 `--apply` |
| `REQUEST_CHANGES` | ❌ | `gatekeeper:needs-changes` | 否 |
| `COMMENT` | 💬 | `gatekeeper:review` | 否 |

> 评分卡 Markdown 模板见 SSOT §6 / SKILL.md，本文不重复。

---

## 5. CLI 命令映射速查（SSOT §7，已对 `shortcuts/` 逐条核验）

| 步骤 | 数据 | 命令 | 备注 |
|------|------|------|------|
| PR 元信息 | 标题/描述/作者/关联 | `gitlink-cli pr +view -i <id> --format json` | `+view`，`-i`/`--id` |
| 变更文件 | 文件路径列表 | `gitlink-cli pr +files -i <id> --format json` | `+files` |
| Diff | 变更内容供 AI 审查 | `gitlink-cli pr +diff -i <id> --format json` | `+diff`；当前实现与 `+files` 命中同一 `/pulls/:id/files` 端点 |
| commits | commit 列表 | `gitlink-cli pr +commits -i <id> --format json` | `+commits` |
| CI 状态 | 构建结果 | `gitlink-cli ci +builds --format json` | `+builds`（`-p`/`-l` 分页） |
| 回写评论 | 评分卡 | `gitlink-cli pr +comment -i <id> -b "<scorecard>"` | `+comment` 底层走 issue journals（评论流）；评审记录形式用 `pr +review -i <id> -s common -c "..."`（走 reviews 端点，payload 字段是 `content`/`status`，status 取 `common`/`approved`/`rejected`）。评分卡作为建议性回写，二者均用 `common` |
| 创建标签 | 裁决标签 | `gitlink-cli label +create -n "<name>" -c "#RRGGBB"` | `+create`（本作品子题一新增；`label +list/+update/+delete` 同组） |
| 挂标签 | 把标签挂到 PR 对应 Issue | `ISSUE_ID=$(gitlink-cli pr +view -i <pr> --format json \| jq -r '.data.issue.id')` 后 `gitlink-cli api POST /:owner/:repo/issues/$ISSUE_ID --body '{"issue_tag_ids":[<id>],"done_ratio":0,"subject":"...","description":"..."}'` | `:id` 是 PR 关联的 **issue.id（非 PR 号）**；更新须带回 `subject`/`description`（见 gitlink-shared）。亦可用 `gitlink-cli issue +update`（自动保留 subject/description） |
| 合并（受限） | 仅 PASS+`auto_merge`+`--apply` | `gitlink-cli pr +merge -i <id> --method <merge_method>` | **CLI 标志为 `--method`/`-m`**（值 `merge`/`rebase`/`squash`）；底层 API payload 字段才叫 `do`（不要当 CLI flag 用） |

### 真实 API 注意（GitLink 平台特性，gatekeeper 直接受影响）

| 坑 | 对 gatekeeper 的影响 |
|----|----------------------|
| PR review `status` 为 `common`/`approved`/`rejected`（非 GitHub 的 event=COMMENT/APPROVE） | gatekeeper 刻意把自动裁决统一以建议性 `common` 回写（或用 `pr +comment`），强语义 `approved`/`rejected` 留给人工；三态裁决靠评分卡标题 + `gatekeeper:*` 标签表达状态 |
| PR list `--state` 过滤不准 | 列 PR 时用返回体 `pull_request_status` 字段客户端过滤（0=open, 1=merged, 2=closed） |
| Issue 更新需带 `subject`/`description`/`done_ratio` | 经 `issue_tag_ids` 挂标签时，Raw API 须先 GET 再回传这些字段，否则可能清空描述 |
| `create_file` 的 `content` 须 base64 | 子题三创建 tracking issue 若需落文件时遵守 |
| 分支删除 API 是平台 bug | gatekeeper 不依赖删分支；勿用 `DELETE .../branches/:name` |
| GitLink 主分支为 `master` | 合并 base、`pr +create --base` 默认 `master` |

> 写操作前必须向用户复述将要做什么；不传 `--apply` 时一律 dry-run（`behavior.dry_run_default`）。Token 不回显。
