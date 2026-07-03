---
name: gitlink-multi-repo-sync
version: 1.0.0
description: "多仓库协同编排：循环遍历多个仓库，依次调用 gitlink-issue + gitlink-pr + gitlink-release 采集各仓库 Issue/PR/Release 状态，汇总成跨仓库协同看板，并给出协调建议。当用户需要统一跟踪多个仓库进展、做跨仓库 Issue/PR/Release 协调时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-multi-repo-sync（多仓库协同 · 编排 Skill）

**CRITICAL — 开始前先阅读任务二的 [`gitlink-shared/SKILL.md`](../../../../skills/gitlink-shared/SKILL.md)（认证、权限、API 注意事项）。**
**CRITICAL — 本工作流为只读采集，不写入任何仓库（纯统计汇总）。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 gh（GitHub CLI）操作 GitLink 资源。**

> **定位**：本 Skill 是"总指挥"编排 Skill，循环调用 issue / pr / release 三个子 Skill，跨多个仓库采集数据，输出统一的协同看板 + 协调建议。解决 PDF 场景"跨多个仓库的统一 Issue 追踪、PR 状态看板、Release 协调发布"。

---

## 编排架构

```
gitlink-multi-repo-sync
  ├── 对每个仓库 <repo> 循环：
  │   ├── Skill("gitlink-issue")     采集 Issue 统计（开放/关闭/未分类）
  │   ├── Skill("gitlink-pr")        采集 PR 状态（开放/已合并）
  │   └── Skill("gitlink-release")   采集 Release（版本数/最新版）
  └── 汇总：跨仓库协同看板 + 协调建议
```

## 子 Skill 依赖

| 子 Skill | 用途 | 写入 |
|---------|------|:----:|
| gitlink-issue | 各仓库 Issue 统计（开放/关闭/未分类）| 否 |
| gitlink-pr | 各仓库 PR 状态（开放/已合并）| 否 |
| gitlink-release | 各仓库 Release（数量/最新版本）| 否 |

**串联满足 PDF「≥3 命令/Skill 串联」要求**：3 个子 Skill × N 个仓库。

---

## 前置：收集参数

| 参数 | 说明 | 示例 |
|------|------|------|
| owner | 所有者（通常多仓库同属一个 owner）| `ylly` |
| repos | 仓库列表（逗号分隔，2-5 个）| `gitlink-cli,gitlink-help-center,demo-repo` |

> 若用户未指定仓库，先用 `repo +list` 列出该 owner 名下仓库，取前 3 个。

---

## 工作流

### Step 1：遍历仓库采集 Issue

**→ 对每个 `<repo>` 调用 `Skill("gitlink-issue", args="列出 <owner>/<repo> 的 Issue 统计：开放数、关闭数、未分类数（tags 为空）。只读。")`**

关键命令（每个仓库执行一次）：
```bash
gitlink-cli issue +list --owner <o> --repo <r> --state open --format json    # opened_count
gitlink-cli issue +list --owner <o> --repo <r> --state closed --format json  # closed_count
```
记录每个仓库的 open / closed / 未分类数。

### Step 2：采集 PR 状态

**→ 对每个 `<repo>` 调用 `Skill("gitlink-pr", args="列出 <owner>/<repo> 的 PR：开放数、已合并数。只读。")`**

```bash
gitlink-cli pr +list --owner <o> --repo <r> --format json
```
记录每个仓库的 PR 开放 / 合并数。

### Step 3：采集 Release

**→ 对每个 `<repo>` 调用 `Skill("gitlink-release", args="列出 <owner>/<repo> 的 Release，取最新版本号。只读。")`**

```bash
gitlink-cli release +list --owner <o> --repo <r> --format json
```
记录每个仓库的 Release 数 + 最新版本。

### Step 4：汇总跨仓库协同看板

AI 汇总所有仓库数据，输出协同看板 + 协调建议（哪个仓库积压、哪个可协调发版等）。

---

## 最终输出

```markdown
## 🔗 跨仓库协同看板 — <owner>

| 仓库 | 开放Issue | 关闭Issue | 未分类 | PR(开/合) | 最新Release | 状态 |
|------|:--------:|:--------:|:------:|:---------:|:----------:|:----:|
| gitlink-cli | 10 | 6 | 7 | 0/2 | v0.2.0-beta | 🟡 积压 |
| help-center | 5 | 20 | 1 | 1/5 | v1.0 | 🟢 活跃 |
| demo-repo | 0 | 0 | 0 | 0/0 | 无 | ⚪ 空仓 |

### 🎯 协同建议
1. **gitlink-cli** Issue 积压 + 半数未分类 → 建议联动 `gitlink-health-doctor` 治理
2. **help-center** 发版活跃 → 可与 gitlink-cli 协调统一发版节奏
3. **demo-repo** 空仓 → 建议初始化（联动 `gitlink-project-bootstrap`）或归档
```

> 这条输出体现了任务三"三件套协同"：multi-repo 发现问题 → 引导用 health-doctor / project-bootstrap 解决。

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| 循环多仓库时 API 频率限制 | 仓库间适当间隔；仓库数控制在 3-5 个 |
| 某仓库无权限 / 不存在 | try-catch 跳过，在看板标注"无权限/不存在"|
| `pr +list --state` 过滤不精确 | 客户端按 `pull_request_status` 字段二次判断（0=open,1=merged,2=closed）|
| `issue +list` 返回数组含已关闭 | 客户端按 `status.id` 二次过滤（1=开放）|
| fork 仓库 PR/Release 为 0 | 属正常（fork 无独立 PR/发版），看板如实展示 |

---

## 实测落地参考

**⚠️ 数据准备**：需 owner 名下 2-3 个仓库。若 ylly 账号下仓库不足，可新建 1-2 个测试仓库（或用 `gitlink-project-bootstrap` 自动创建）。

实测时遍历 `ylly/gitlink-cli` + 其他仓库，输出跨仓库看板，给出协同建议（联动 health-doctor / project-bootstrap）。
