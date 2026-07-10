---
name: gitlink-community-ops
version: 1.0.0
description: "GitLink 社区运营自动化统一入口：编排三个子 Skill（gitlink-issue-triage-rules + gitlink-community-report + gitlink-release-notes）实现端到端工作流——新 Issue 自动分类 → 分配责任人 → 生成社区周报 → 发布 Release Notes。触发场景：社区运营自动化、一键社区巡检、完整社区工作流、community ops、社区运营一键跑、帮我做社区运维。"
license: MulanPSL-2.0
metadata:
  requires:
    bins_any: ["gitlink-cli", "gh", "glab", "curl"]
    bins_note: "gitee 后端无官方 CLI，直接使用 curl 调用 https://gitee.com/api/v5/ REST API；其余三后端用对应原生 CLI"
  cliHelp: "gitlink-cli --help"
  platforms:
    agents:
      - openclaw
      - claude-code
      - cursor
      - generic-agent
    backends:
      - id: gitlink
        cli: gitlink-cli
        url_template: "https://www.gitlink.org.cn/{owner}/{repo}"
        api_base: "https://www.gitlink.org.cn/api/v1"
        auth_env: GITLINK_TOKEN
        default: true
      - id: github
        cli: gh
        url_template: "https://github.com/{owner}/{repo}"
        api_base: "https://api.github.com"
        auth_env: GH_TOKEN
      - id: gitlab
        cli: glab
        url_template: "https://gitlab.com/{owner}/{repo}"
        api_base: "https://gitlab.com/api/v4"
        auth_env: GITLAB_TOKEN
      - id: gitee
        cli: curl
        url_template: "https://gitee.com/{owner}/{repo}"
        api_base: "https://gitee.com/api/v5"
        auth_env: GITEE_TOKEN
  subSkills:
    - gitlink-issue-triage-rules
    - gitlink-community-report
    - gitlink-release-notes
---

# gitlink-community-ops（社区运营自动化统一入口）

**CRITICAL — 本 Skill 是编排入口，不直接执行任何 CLI 命令。它通过引用三个子 Skill 完成所有实际操作。Agent 必须按本文件的步骤依次加载并执行每个子 Skill，不能跳过、不能并行。**
**CRITICAL — 每个子 Skill 执行前必须独立做 dry-run 预览并等待用户确认。不能在编排层面一次性确认所有子 Skill 的写回。**
**CRITICAL — 子 Skill 之间共享上下文参数（OWNER / REPO / BACKEND），Agent 必须在每次子 Skill 调用时传递这些参数。**

> **依赖 Skill：** 三个子 Skill（必须全部可用）
> 1. `gitlink-issue-triage-rules` — Issue 自动分类 + 分配责任人
> 2. `gitlink-community-report` — 社区周报自动生成 + 发布
> 3. `gitlink-release-notes` — Release Notes 自动生成 + 发布
> **本 Skill 为 C1 模式（纯文档）**：Agent 按本文件步骤依次加载子 Skill 并执行。

---

## 功能概述

本 Skill 是社区运营自动化工作流的**统一编排入口**。它不自己执行任何 CLI 命令，而是按步骤指引 Agent 依次加载三个子 Skill：

```
编排流程:
  Step 1 → 加载 gitlink-issue-triage-rules → 执行 Issue 分拣
  Step 2 → 加载 gitlink-community-report    → 执行周报生成
  Step 3 → 加载 gitlink-release-notes       → 执行版本发布
  Step 4 → 输出工作流总结
```

**设计原则**：
- ✅ **顺序执行**：Step 1 必须先完成（分拣后才有标签数据供周报和 Release Notes 分类使用）
- ✅ **逐个确认**：每个子 Skill 的写回操作需独立确认（不在编排层一次性全确认）
- ✅ **上下文传递**：OWNER / REPO / BACKEND 在子 Skill 间自动继承
- ✅ **容错继续**：某子 Skill 失败不阻塞后续步骤（但输出警告）
- ✅ **可选跳过**：用户可选择只执行部分步骤（如只做分拣不做周报）

## 触发场景

用户提到以下关键词时自动触发：
- "社区运营自动化"、"一键社区巡检"、"完整社区工作流"
- "community ops"、"社区运营一键跑"、"帮我做社区运维"
- "帮我跑完整工作流"（同时涉及分拣 + 周报 + 发布）
- "全流程社区运营"

> **注意**：如果用户只说"分拣 Issue" → 只触发 `gitlink-issue-triage-rules`；只说"生成周报" → 只触发 `gitlink-community-report`；只说"发布 Release Notes" → 只触发 `gitlink-release-notes`。本 Skill 仅在用户表达**完整/多阶段**需求时触发。

---

## 全局参数

| 参数 | 来源 | 默认值 | 说明 |
|------|------|--------|------|
| `OWNER` | 用户输入或 git remote | 必须 | 仓库 owner |
| `REPO` | 用户输入或 git remote | 必须 | 仓库名 |
| `BACKEND` | 自动检测 | gitlink | 平台后端（gitlink/github/gitlab/gitee） |
| `PERIOD` | 用户指定 | 7 | 周报统计周期（天） |
| `VERSION` | 用户指定或自动 | vYYYY.MM.DD | Release Notes 版本号 |
| `DRY_RUN` | 默认 | true | 所有子 Skill 默认先预览 |

---

## 工作流（Agent 执行步骤）

### Step 0：环境检测 + 参数收集

```bash
# 0.1 检测后端
BACKEND=$(detect_backend)
echo "✓ 后端: $BACKEND"

# 0.2 认证检查
case "$BACKEND" in
  gitlink) gitlink-cli auth status ;;
  github)  gh auth status ;;
  gitlab)  glab auth status ;;
  gitee)   test -n "$GITEE_TOKEN" && echo "Gitee OK" ;;
esac

# 0.3 确认仓库参数
# Agent 从用户输入或 git remote 推断 OWNER 和 REPO
echo "仓库: $OWNER/$REPO"

# 0.4 确认执行范围
# 询问用户: 全流程 (Step 1-3) 还是部分步骤?
# 默认: 全流程
```

### Step 1：执行 gitlink-issue-triage-rules（Issue 自动分类）

```
Agent 操作:
1. 加载 gitlink-issue-triage-rules Skill
2. 传递参数: OWNER, REPO, BACKEND
3. Agent 按该 Skill 的 Step 0-7 执行:
   - 认证检测 → 加载规则 → 列出未分拣 Issue → 建立映射 → 规则评估 → dry-run 预览 → 用户确认 → 写回 → 验证
4. 记录分拣结果摘要 (供后续步骤使用)
```

**分拣结果摘要格式**（Agent 输出并保留在上下文中）：

```markdown
## Step 1 完成：Issue 分拣结果

- 分拣 Issue: {count} 条
- 规则命中: {count} 条
- LLM 兜底: {count} 条
- 跳过: {count} 条
- 写回成功: {count} 条
```

### Step 2：执行 gitlink-community-report（社区周报生成）

```
Agent 操作:
1. 加载 gitlink-community-report Skill
2. 传递参数: OWNER, REPO, BACKEND, PERIOD
3. Agent 按该 Skill 的 Step 0-6 执行:
   - 认证检测 → 数据采集 → 时间过滤 → 生成周报 → dry-run 预览 → 用户确认 → 发布 → 验证
4. 记录周报结果摘要
```

**周报结果摘要格式**：

```markdown
## Step 2 完成：社区周报

- 周报标题: 📊 社区周报 {date}
- 开放 Issue: {count} | 已关闭: {count}
- 本周新建: {count} | 本周关闭: {count}
- 周报 Issue 编号: #{number}
```

### Step 3：执行 gitlink-release-notes（Release Notes 发布）

```
Agent 操作:
1. 加载 gitlink-release-notes Skill
2. 传递参数: OWNER, REPO, BACKEND, VERSION
3. Agent 按该 Skill 的 Step 0-6 执行:
   - 认证检测 → 数据采集 → 分类 → 生成 Release Notes → dry-run 预览 → 用户确认 → 创建 Release → 验证
4. 记录发布结果摘要
```

**发布结果摘要格式**：

```markdown
## Step 3 完成：Release Notes

- 版本号: {version}
- Bug 修复: {count} 项 | 新功能: {count} 项 | 其他改进: {count} 项
- Release 已创建: ✓
```

### Step 4：输出工作流总结

Agent 输出完整的执行报告：

```markdown
## 🎯 社区运营自动化工作流总结

**仓库:** {OWNER}/{REPO}
**后端:** {BACKEND}
**执行时间:** {NOW}

### Step 1 — Issue 自动分类
- 分拣: {count} 条 | 规则命中: {count} | LLM 兜底: {count} | 跳过: {count}

### Step 2 — 社区周报
- 周报已发布: Issue #{number}
- 开放 Issue: {count} | 本周新建: {count} | 本周关闭: {count}

### Step 3 — Release Notes
- 版本: {version}
- Bug 修复: {count} | 新功能: {count} | 其他改进: {count}

### 工作流串联的 CLI 命令 (≥3)
1. gitlink-cli auth status        (Step 0 认证)
2. gitlink-cli issue +list        (Step 1 列 Issue)
3. gitlink-cli label +list        (Step 1 标签映射)
4. gitlink-cli member +list       (Step 1 成员映射)
5. gitlink-cli issue +view        (Step 1 回读)
6. gitlink-cli api PATCH          (Step 1 写回)
7. gitlink-cli issue +comment     (Step 1 审计)
8. gitlink-cli issue +create      (Step 2 周报发布)
9. gitlink-cli release +list      (Step 3 去重)
10. gitlink-cli release +create   (Step 3 发布)

✅ 工作流完成
```

---

## 容错与错误处理

| 场景 | 处理 |
|------|------|
| Step 1 分拣失败 | 输出警告，**继续执行 Step 2/3**（周报和 Release Notes 仍可基于未分拣数据生成，只是分类可能不够精确） |
| Step 2 周报发布失败 | 输出警告，**继续执行 Step 3**（Release Notes 独立于周报） |
| Step 3 发布失败 | 输出警告，**Step 1/2 已完成的部分不受影响** |
| 认证失败 | **阻塞所有步骤**，提示用户检查 token |
| 子 Skill 不可用 | **阻塞该步骤**，提示用户安装缺失的 Skill |

---

## 数据流图

```
┌───────────────────────────────────────────────────────────────────┐
│              gitlink-community-ops (统一入口)                      │
│                                                                   │
│  Step 0: 环境检测 + 参数收集                                       │
│          ↓ OWNER, REPO, BACKEND, PERIOD, VERSION                  │
│                                                                   │
│  Step 1: ┌──────────────────────────────────────┐                 │
│          │ gitlink-issue-triage-rules             │                │
│          │ → 分拣结果 (标签/优先级/责任人写入)      │                │
│          │ → 分拣摘要供后续步骤使用                  │                │
│          └──────────────────────────────────────┘                 │
│          ↓                                                         │
│  Step 2: ┌──────────────────────────────────────┐                 │
│          │ gitlink-community-report               │                │
│          │ → 周报 Issue 发布                       │                │
│          │ → 标签分布 + 高优先级数据来自分拣后的      │                │
│          │   Issue 列表 (所以 Step 1 须先完成)      │                │
│          └──────────────────────────────────────┘                 │
│          ↓                                                         │
│  Step 3: ┌──────────────────────────────────────┐                 │
│          │ gitlink-release-notes                  │                │
│          │ → Release 创建                          │                │
│          │ → Bug/Feature 分类来自分拣后的标签数据    │                │
│          └──────────────────────────────────────┘                 │
│          ↓                                                         │
│  Step 4: 输出工作流总结                                             │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

**关键依赖关系**：
- Step 2 和 Step 3 的数据质量取决于 Step 1 的分拣结果
- Step 1 必须先完成 → Issue 才有标签 → 周报标签分布才能准确 → Release Notes 才能按 Bug/Feature 正确分类
- 如果 Step 1 被跳过或失败 → Step 2/3 仍可运行，但分类基于原始标签（可能不精确）

---

## 部分执行

用户可只执行部分步骤：

| 用户意图 | 执行范围 |
|----------|----------|
| "只分拣 Issue" | Step 1 only → 直接触发 `gitlink-issue-triage-rules` |
| "只生成周报" | Step 2 only → 直接触发 `gitlink-community-report` |
| "只发布 Release" | Step 3 only → 直接触发 `gitlink-release-notes` |
| "分拣 + 周报" | Step 1 + Step 2 |
| "全流程" | Step 1 + Step 2 + Step 3 |

Agent 应根据用户意图自动选择执行范围，而不是每次都跑全流程。

---

## 安全与写回策略

| 规则 | 说明 |
|------|------|
| **逐个确认** | 每个子 Skill 的写回操作需**独立**确认（不在编排层一次性全确认） |
| **默认 dry-run** | 所有子 Skill 默认先做 dry-run 预览 |
| **容错继续** | 某步骤失败不阻塞后续步骤 |
| **认证阻塞** | 认证失败阻塞所有步骤 |
| **记录摘要** | 每步完成后输出摘要，供用户和工作流总结使用 |

---

## 使用示例

### 示例 1：全流程（GitLink）

**用户输入**：

```
帮我跑 Angel123456/gitlink-cli 的完整社区运营工作流
```

**Agent 执行**：

```
Step 0: 检测后端 → gitlink | 认证 OK | 仓库: Angel123456/gitlink-cli

Step 1: 加载 gitlink-issue-triage-rules
  → 7 条未分拣 Issue → 规则命中 4 条 → dry-run → 用户确认 → 写回 → 验证
  → 分拣摘要: 4 条写回成功

Step 2: 加载 gitlink-community-report (PERIOD=7)
  → 数据采集 → 本周新建 5 条 / 关闭 3 条 → dry-run → 用户确认 → 发布周报 Issue
  → 周报摘要: Issue #12 已发布

Step 3: 加载 gitlink-release-notes (VERSION=v2026.07.10)
  → Bug 修复 2 / 新功能 1 / 其他 2 → dry-run → 用户确认 → 创建 Release
  → 发布摘要: v2026.07.10 已发布

Step 4: 输出工作流总结 (10+ CLI 命令串联)
```

### 示例 2：分拣 + 周报（GitHub，不发布 Release）

**用户输入**：

```
帮我分拣 xuanlanwuta/gps_SM 的 Issue，然后生成一份周报
```

**Agent 执行**：

```
Step 0: 检测后端 → github | 认证 OK

Step 1: 加载 gitlink-issue-triage-rules
  → 4 条 Issue → 规则命中 4 条 → dry-run → 用户确认 → 写回

Step 2: 加载 gitlink-community-report (PERIOD=7)
  → 数据采集 → dry-run → 用户确认 → 发布周报

(跳过 Step 3，用户未要求发布 Release)

Step 4: 输出部分工作流总结 (Step 1 + Step 2)
```

---

## Agent 平台兼容性

| 平台 | 加载方式 | 触发方式 |
|------|----------|----------|
| **OpenClaw** | `workspace/skills/gitlink-community-ops/SKILL.md` | 自然语言 |
| **Claude Code** | `~/.claude/skills/gitlink-community-ops/SKILL.md` | 自然语言 |
| **Cursor** | `~/.cursor/skills/gitlink-community-ops/SKILL.md` | 自然语言 + `/` 命令 |
| **WorkBuddy** | `~/.workbuddy/skills/gitlink-community-ops/SKILL.md` | 自然语言 |
| **通用 Agent** | 作为参考文档 | 按 SKILL.md 流程自取 |

---

## 子 Skill 安装位置

三个子 Skill 必须与本 Skill 在同一级目录：

```
~/.workbuddy/skills/
  ├── gitlink-issue-triage-rules/SKILL.md   (已有)
  ├── gitlink-community-report/SKILL.md     (新增)
  ├── gitlink-release-notes/SKILL.md        (新增)
  └── gitlink-community-ops/SKILL.md        (新增, 本文件)
```

或项目级：

```
<project>/.workbuddy/skills/
  ├── gitlink-issue-triage-rules/SKILL.md
  ├── gitlink-community-report/SKILL.md
  ├── gitlink-release-notes/SKILL.md
  └── gitlink-community-ops/SKILL.md
```

Agent 加载子 Skill 时，按同级目录的相对路径 `../<skill-name>/SKILL.md` 查找。
