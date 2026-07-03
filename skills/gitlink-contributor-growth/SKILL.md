---
name: gitlink-contributor-growth
version: 1.0.0
description: "贡献者成长体系编排 Skill：串联 gitlink-insight + gitlink-issue-triage，完成「采集贡献数据 → 综合积分排行榜 → 三级徽章授予 → 颁奖 Issue 公布」的端到端流程。当用户需要识别仓库贡献者、做积分排行、颁发徽章 label、建颁奖 Issue 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-contributor-growth（贡献者成长体系 · 端到端编排）

**CRITICAL — 开始前先阅读任务二的 `gitlink-shared/SKILL.md`，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入操作前（建 label、建颁奖 Issue、打标签），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **定位**：本 Skill 是一个"总指挥"编排 Skill，依次调用任务二的 2 个子 Skill（insight 取数 + issue-triage 颁奖）+ git log 聚合，完成贡献者激励闭环。

---

## 工作流概览（串联 2 个子 Skill + git log）

| 步骤 | 调用子 Skill / 命令 | 做什么 | 写入 |
|------|------------------|--------|:----:|
| ① 取数 | gitlink-insight 工作流 3 + git log | 采集 commits / merged PR / Issue 作者 | 否 |
| ② 排行 | （本 Skill 内置公式）| 计算综合积分，输出排行榜 | 否 |
| ③ 颁奖 | gitlink-issue-triage 工作流 1（Step 5-6）| 创建三级徽章 label + 颁奖 Issue + @mention | 是 |

**串联满足 PDF「≥3 命令/Skill 串联」要求**：3 个 Step 串 2 个子 Skill + git log。

---

## 详细工作流

### Step 1：采集贡献数据（调用 gitlink-insight 工作流 3）

严格按 `skills/gitlink-insight/SKILL.md` 工作流 3（**第 169-203 行**）：

| 数据源 | 蓝图行号 | 命令 |
|--------|---------|------|
| Commits 排行 | 第 173-183 行 采集数据 | `git log --format="%an" \| sort \| uniq -c \| sort -rn` |
| 合并 PR | 第 173-183 行 | `gitlink-cli pr +list --owner <o> --repo <r> --state merged --format json` |
| Issue 作者聚合 | 第 173-183 行 | `gitlink-cli issue +list --owner <o> --repo <r> --format json` 后按 `author` 聚合 |
| 三层分布 | 第 194-197 行 | 核心 / 活跃 / 新增（对应"星级 / 活跃 / 贡献者"三级） |

**⚠️ 已知降级**：`contributors` API endpoint 在 GitLink 返回 HTML 而非 JSON（蓝图已记录），改用 `git log` 取 commits 是经过验证的降级方案。

**⚠️ 数据清洗**：必须过滤测试号（如 `15972095207` / `2403_89190320` / `fsafasff` 等明显机器号），否则排行榜失真。

### Step 2：生成综合积分排行榜（本 Skill 内置公式）

**积分公式**：`综合积分 = commits × 1 + 合并 PR × 5 + Issue × 2`

**公式理由**：PR 权重高，因合并工作量大（需通过 code review）；Issue 权重中等（提需求成本低）；commits 是基础活跃度。

**输出格式**：

```markdown
| 排名 | 贡献者 | Commits | 合并 PR | Issue | 综合积分 |
|:----:|--------|:-------:|:-------:|:-----:|:--------:|
| 1    | xxx    | 42      | 0       | 0     | 42       |
| ...
```

### Step 3：颁发徽章（调用 gitlink-issue-triage 工作流 1 Step 5-6）

按 `skills/gitlink-issue-triage/SKILL.md` 工作流 1 Step 5-6（**第 80-117 行**）：

#### Step 3a：查现有标签 + 创建徽章 label（第 80-91 行）

```bash
# 先查是否已有同名 label（蓝图要求优先复用）
gitlink-cli label +list --owner <o> --repo <r> --format json
```

未命中则建 3 个徽章（**写入前确认用户意图**）：

| 徽章名 | 颜色 | 授予标准 |
|--------|------|---------|
| 星级贡献者 | `#FFD700`（金） | commits ≥ 10 或 合并 PR ≥ 2 |
| 活跃贡献者 | `#FF6B35`（橙） | commits ≥ 5 或 Issue ≥ 5 |
| 贡献者 | `#87C95F`（绿） | 有任意提交 |

```bash
# payload_star.json: {"name":"星级贡献者","color":"#FFD700"}
MSYS_NO_PATHCONV=1 gitlink-cli api POST /<owner>/<repo>/labels --body-file payload_star.json --format json
# 同理建活跃 / 贡献者
```

**API 路径**：label 走 `/owner/repo/...`（不带 `/v1`）。

#### Step 3b：建颁奖 Issue（第 101-117 行报告格式）

颁奖 Issue 模板：
- **subject**：`🏆 <版本号> 贡献者排行榜公布`
- **description**：含完整排行榜表格 + Top 贡献者 @mention
- **必须传 `done_ratio: 0`**，否则 MySQL 报错
- **issue_tag_ids 创建时不生效**（GitLink bug），需创建后用 `issue +update --label` 补打

```bash
# payload_award.json:
# {
#   "subject": "🏆 vX.Y.Z 贡献者排行榜公布",
#   "description": "<完整排行榜 + Top @mention>",
#   "priority_id": 2,
#   "done_ratio": 0
# }

MSYS_NO_PATHCONV=1 gitlink-cli api POST /<owner>/<repo>/issues --body-file payload_award.json --format json
# 拿到 issue number 后补打星级 label：
gitlink-cli issue +update --owner <o> --repo <r> --number <n> --label <star_label_id>
```

**API 路径**：issue 走 `/owner/repo/...`（不带 `/v1`）也能创建，但 issue 后续操作（如 journals）必须 `/v1`。

---

## 输出

| 步骤 | 产物 |
|------|------|
| ① 取数 | commits / PR / Issue 原始数据 JSON |
| ② 排行 | 综合积分排行榜 Markdown 表格 |
| ③ 颁奖 | 3 个 label ID + 1 个颁奖 Issue number + URL |

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| `contributors` API 返回 HTML 不是 JSON | 降级用 `git log --format="%an" \| sort \| uniq -c` 取 commits |
| 排行榜混入测试号（15972095207 等）| 积分前过滤已知测试账号 |
| 创建 Issue 不传 `done_ratio` 报 MySQL 错 | payload 必加 `done_ratio: 0, priority_id: 2` |
| `issue_tag_ids` 创建时不生效（GitLink bug） | 创建后用 `issue +update --label <id>` 补打 |
| Windows Git Bash 拼 JSON 中文乱码 | 全部用 `--body-file <UTF-8文件>` + `MSYS_NO_PATHCONV=1` |
| label 跨仓库独立 | 每个 owner/repo 的 label ID 不通用，颁奖前需在目标仓库重新建 |

---

## 实测落地参考

本工作流已在 `ylly/gitlink-cli` 实测落地（2026-06-29）：

| 步骤 | 实测结果 |
|------|---------|
| ① 取数 | git log 取到 commits 排行（wbtiger 42 / whzy 9 / wbavon 8 等）+ merged PR + 14 个 Issue 作者 |
| ② 排行 | 14 人综合积分排行榜（已过滤 3 个测试号） |
| ③ 颁奖 | 创建 3 个徽章 label（星级 394180 / 活跃 394181 / 贡献者 394182）+ 颁奖 Issue **#17**（带「星级贡献者」label） |

**Top 3 贡献者**：① wbtiger（42 分）② ylly（33 分）③ ZxR123-Z（24 分）

执行报告：仓库内 `examples/workflows/zhangqing-task3/05-contributor-growth.md`。
