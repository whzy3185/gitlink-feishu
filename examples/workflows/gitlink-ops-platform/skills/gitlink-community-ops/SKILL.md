---
name: gitlink-community-ops
version: 1.0.0
description: "社区运营自动化编排 Skill：串联 gitlink-issue-triage + gitlink-insight + gitlink-release-auto，完成「Issue 自动分类 → 项目周报 → 自动发版」的社区运营闭环。当用户需要批量治理 Issue、生成项目周报、基于提交历史发版，或对仓库做社区运营收尾时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-community-ops（社区运营自动化 · 端到端编排）

**CRITICAL — 开始前先阅读任务二的 `gitlink-shared/SKILL.md`，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入操作前（打标签、改 Issue、发版），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **定位**：本 Skill 是一个"总指挥"编排 Skill，本身不直接定义底层命令，而是依次调用任务二的 3 个子 Skill（issue-triage → insight → release-auto）完成端到端社区运营闭环。

---

## 工作流概览（串联 3 个子 Skill）

| 步骤 | 调用子 Skill | 做什么 | 写入 |
|------|------------|--------|:----:|
| ① Issue 自动分类 | gitlink-issue-triage 工作流 1 | 扫描未分类 Issue → 语义分类 → 复用现有标签打标 | 是 |
| ② 责任人分配 | gitlink-issue-triage 工作流 2 | 推荐责任人 → 分配 → notification 验证 | 是（个人仓库受限） |
| ③ 项目周报 | gitlink-insight 工作流 2 | 采集 Issue/PR/Release 数据 → 输出周报 | 否 |
| ④ 自动发版 | gitlink-release-auto | 推荐版本号 → 生成 Release Notes → 发版 | 是 |

**串联满足 PDF「≥3 命令/Skill 串联」要求**：4 个 Step 串 3 个子 Skill。

---

## 详细工作流

### Step 1：Issue 自动分类（调用 gitlink-issue-triage 工作流 1）

严格按 `skills/gitlink-issue-triage/SKILL.md` 工作流 1（**第 54-117 行**）执行：

| 子动作 | 蓝图行号 | 命令 |
|--------|---------|------|
| 拉开放 Issue | 第 58-62 行 Step 1 | `gitlink-cli issue +list --owner <o> --repo <r> --state open --format json` |
| 筛未分类 | 第 64-68 行 Step 2 | 过滤 `tags=[]` 或 `issue_tags=[]` |
| 读详情 | 第 70-78 行 Step 3-4 | `gitlink-cli issue +view --number <n>` 拿完整描述 |
| 语义分类 | 第 70-78 行 + 分类规则表 第 35-50 行 | 按语义（非仅关键词）归类 |
| **复用现有标签**（关键决策）| 第 80-91 行 Step 5 | **优先复用仓库已有标签（含中文同义词），先匹配再考虑新建英文标签** |
| 打标签 | 第 93-99 行 Step 6 | `gitlink-cli issue +update --owner <o> --repo <r> --number <n> --label <id>` |
| 输出报告 | 第 101-117 行 Step 7 | 表格汇总每个 Issue 的分类结果 |

**分类决策原则**（蓝图第 46-50 行）：
1. 安全类最高优先级
2. bug 优先于 enhancement
3. 模糊的 feature/question 归 question
4. 完全无法理解 → `triage` 兜底

**验证**：再次 `issue +list` 确认未分类 Issue = 0。

### Step 2：责任人分配（调用 gitlink-issue-triage 工作流 2）

按 `skills/gitlink-issue-triage/SKILL.md` 第 125-146 行：

```bash
gitlink-cli issue +assigners --owner <o> --repo <r> --format json
```

**⚠️ 平台限制**（蓝图第 133 行红框已预判）：GitLink 个人仓库的 assigners 机制对 Manager 角色不开放，返回空数组（`total_count: 0`）。即便账号已是 Manager 也不在候选列表内。蓝图第 146 行明确："列表为空则跳过分配并在报告中标注"。Raw API PATCH `/v1/.../issues/<n>` 带 `assigned_to_id` 返回 ok 但 assigners 仍为 `[]`，**勿重复尝试**。

### Step 3：项目周报（调用 gitlink-insight 工作流 2）

严格按 `skills/gitlink-insight/SKILL.md` 工作流 2（**第 116-166 行**）：

| 数据 | 蓝图行号 | 命令 |
|------|---------|------|
| 开放 Issue | 120-122 | `gitlink-cli issue +list --state open` |
| 已关闭 Issue | 120-122 | `gitlink-cli issue +list --state closed` |
| 合并 PR | 123-125 | `gitlink-cli pr +list --state merged` |
| Release | 126-128 | `gitlink-cli release +list` |
| 风险标注 | 164-165 | 列风险与阻塞 |

**输出格式**：Markdown 周报，含计数表 / 标签分布 / Issue 作者排行 / 风险观察。

### Step 4：自动发版（调用 gitlink-release-auto）

严格按 `skills/gitlink-release-auto/SKILL.md`：

| 子动作 | 蓝图行号 | 做什么 |
|--------|---------|------|
| 版本号推荐 | 第 32-90 行 一、 | 按 Conventional Commits 分类统计 feat/fix/docs，推 Semver 升级（feat→MINOR, fix→PATCH） |
| Release Notes 生成 | 第 94-145 行 二、 | 按类型分组 + **贡献者必须去重**（第 145 行红框警告） |
| 预发布 | 第 179-189 行 三、 | 任务未完工时 `--prerelease true` 发 beta 版本 |

**⚠️ 关键避坑**：`gitlink-cli release +create` **无 `--body-file` 参数**，`--body` 传多行中文在 Windows Git Bash 必乱码。改用 Raw API：

```bash
# payload.json（UTF-8 编码）
# {
#   "tag_name": "vX.Y.Z",
#   "name": "版本名",
#   "body": "<Release Notes 全文>",
#   "target_commitish": "master",
#   "prerelease": true
# }

MSYS_NO_PATHCONV=1 gitlink-cli api POST /<owner>/<repo>/releases \
  --body-file payload.json --format json
```

**API 路径**：release 走 `/owner/repo/...`（不带 `/v1`）。

---

## 输出

| 步骤 | 产物 |
|------|------|
| ① 分类 | 分类报告表（每个 Issue → 标签）+ 未分类计数归零验证 |
| ② 分配 | 跳过说明（命中平台限制时）|
| ③ 周报 | Markdown 周报（可直接贴 Issue/Wiki）|
| ④ 发版 | version_id + tag + URL |

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| `release +create` 无 `--body-file`，多行中文乱码 | 改用 `api POST /owner/repo/releases --body-file payload.json` |
| GitLink API 路径双轨制 | release/label 用 `/owner/repo/...` 即可，**勿加 `/v1`** |
| assigners 在个人仓库对 Manager 不开放 | 平台限制，报告中标注跳过，勿重复尝试 |
| Windows Git Bash 拼 JSON 中文乱码 | 全部用 `--body-file <UTF-8文件>` + `MSYS_NO_PATHCONV=1` |
| `issue +list` 不返回 tags 字段 | 用 `issue +view --number N` 拿完整 tags |

---

## 实测落地参考

本工作流已在 `ylly/gitlink-cli` 实测落地（2026-06-29）：

| 步骤 | 实测结果 |
|------|---------|
| ① 分类 | 11 个未分类 Issue 全部按仓库已有中文标签（测试/缺陷/功能/性能/疑问）归类，未新建英文标签 |
| ② 分配 | 命中平台限制跳过（assigners 返回空） |
| ③ 周报 | 采集到 16 open / 6 closed / 2 merged PR，标签分布：测试 6 / 缺陷 3 / 文档 3 / good first 3 |
| ④ 发版 | 发布 Release **v0.2.0-beta.1**（version_id=2218，预发布）|

执行报告：仓库内 `examples/workflows/zhangqing-task3/01-community-ops-automation.md`。
