# GitLink 社区运营自动化工作流

> 端到端社区运营自动化：新 Issue 自动分类 → 分配责任人 → 定期生成社区周报 → 自动发布 Release Notes
>
> **四个 Skill 统一编排** — 每个阶段由独立的 Skill（SKILL.md）驱动，Agent 按 Skill 步骤直接调用 CLI 执行。`gitlink-community-ops` Skill 作为统一入口编排三个子 Skill。

**真实运行项目：** [Angel123456/gitlink-cli](https://www.gitlink.org.cn/Angel123456/gitlink-cli)（GitLink 平台）
**运行服务器：** `ssh -p 39925 root@connect.westb.seetacloud.com`

---

## 一、架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│           gitlink-community-ops (统一入口 Skill)                     │
│                                                                     │
│  Step 0: 环境检测 + 参数收集 (OWNER/REPO/BACKEND)                    │
│          ↓                                                          │
│  Step 1: ┌──────────────────────────────────────┐                  │
│          │ gitlink-issue-triage-rules (子 Skill) │                  │
│          │ → Issue 自动分类 + 分配责任人          │                  │
│          │ → 规则匹配 + LLM 兜底 + 写回           │                  │
│          └──────────────────────────────────────┘                  │
│          ↓                                                          │
│  Step 2: ┌──────────────────────────────────────┐                  │
│          │ gitlink-community-report (子 Skill)   │                  │
│          │ → 社区周报自动生成 + 发布               │                  │
│          └──────────────────────────────────────┘                  │
│          ↓                                                          │
│  Step 3: ┌──────────────────────────────────────┐                  │
│          │ gitlink-release-notes (子 Skill)      │                  │
│          │ → Release Notes 自动生成 + 发布        │                  │
│          └──────────────────────────────────────┘                  │
│          ↓                                                          │
│  Step 4: 输出工作流总结                                              │
│                                                                     │
│  底层: gitlink-cli / gh / glab / curl → 各平台 REST API             │
└─────────────────────────────────────────────────────────────────────┘
```

**四个 Skill 的职责分工：**

| Skill | 职责 | 模式 | 触发词 |
|-------|------|------|--------|
| `gitlink-community-ops` | 统一编排入口 | C1 (纯文档) | "社区运营自动化"、"一键社区巡检" |
| `gitlink-issue-triage-rules` | Issue 自动分类 + 分配责任人 | C1 (纯文档) | "分拣 Issue"、"按规则打标签" |
| `gitlink-community-report` | 社区周报生成 + 发布 | C1 (纯文档) | "生成社区周报"、"weekly report" |
| `gitlink-release-notes` | Release Notes 生成 + 发布 | C1 (纯文档) | "发布 Release Notes"、"版本发布" |

**关键依赖关系：**
- Step 1 (分拣) 必须先完成 → Issue 才有标签 → Step 2/3 的数据才准确
- Step 2 (周报) 和 Step 3 (Release Notes) 独立于彼此，可单独执行
- 每个子 Skill 独立做 dry-run 预览 + 用户确认，不在编排层一次性全确认

> 独立架构图文件：`docs/architecture.html`（可在浏览器中打开查看）

---

## 二、Skill 1 — gitlink-issue-triage-rules（Issue 自动分类 + 分配责任人）

### 执行机制

Agent 加载 `gitlink-issue-triage-rules/SKILL.md`，按 Step 0-7 直接调用 CLI 执行：

1. Step 0：后端检测 + 认证检查
2. Step 1：从 SKILL.md 内嵌 YAML 代码块加载规则（mode / rules[] / defaults）
3. Step 2：列出未分拣 Issue（`gitlink-cli issue +list --state open`）
4. Step 3：建立标签 + 成员 + 优先级 ID 映射
5. Step 4：按 SKILL.md 规则评估每条 Issue（keyword 匹配 / LLM 兜底）
6. Step 5：dry-run 预览（Markdown 表格，**必须先看**）
7. Step 6：用户确认后写回（先 `+view` 回读防清空 → `api PATCH` → `+comment` 审计）
8. Step 7：验证写回结果（`+view`）

### 内嵌规则 (mode: hybrid)

| 规则 ID | 类型 | 匹配关键词 | 标签 (GitLink 中文) | 优先级 |
|---------|------|-----------|-------------------|--------|
| bug-default | bug | 错误, 失败, 崩溃, panic, crash | 缺陷 | high |
| question-default | question | 请问, 如何, 怎么, how to | 疑问 | normal |
| docs-typo | docs | typo, 文档, README | 文档 | low |

**hybrid 模式**：规则优先匹配，无命中时 LLM 兜底。

### 双语 Label 翻译

GitLink/Gitee 后端优先中文标签（缺陷/功能/疑问/文档），GitHub/GitLab 后端优先英文（bug/enhancement/question/docs）。一份规则兼容 4 个平台。

### CRITICAL 约束

- 写回前必须 `+view` 回读标题和正文（防 PATCH 清空这两个字段）
- 默认 dry-run，用户确认后才执行写回
- 每次写回附带审计评论（含 matched_rules）

---

## 三、Skill 2 — gitlink-community-report（社区周报生成 + 发布）

### 执行机制

Agent 加载 `gitlink-community-report/SKILL.md`，按 Step 0-6 直接调用 CLI 执行：

| Step | 动作 | CLI 命令 |
|------|------|----------|
| 0 | 后端检测 + 认证 | `gitlink-cli auth status` |
| 1 | 采集数据 | `issue +list` (open/closed) + `release +list` + `member +list` |
| 2 | 时间过滤 + 统计 | Agent 计算本周新建/关闭/标签分布/高优先级 |
| 3 | 生成周报 | 按内嵌 Markdown 模板填充 |
| 4 | dry-run 预览 | 输出完整周报内容 |
| 5 | 发布周报 | `gitlink-cli issue +create --title "📊 社区周报 ..." --body <周报>` |
| 6 | 验证 | 确认周报 Issue 已创建 |

### 周报结构

周报包含：概览（开放/关闭/新建/关闭数）、本周新建 Issue 清单、本周关闭 Issue 清单、标签分布、高优先级待办、本周 Release。

### 周报标识

标题前缀 `📊 社区周报`，便于其他 Skill 自动识别跳过（不纳入 Issue 分拣）。

---

## 四、Skill 3 — gitlink-release-notes（Release Notes 生成 + 发布）

### 执行机制

Agent 加载 `gitlink-release-notes/SKILL.md`，按 Step 0-6 直接调用 CLI 执行：

| Step | 动作 | CLI 命令 |
|------|------|----------|
| 0 | 后端检测 + 认证 | `gitlink-cli auth status` |
| 1 | 采集数据 | `issue +list` (closed/open) + `release +list` |
| 2 | 版本号去重 + 分类 | Bug 修复 / 新功能 / 其他改进 |
| 3 | 生成 Release Notes | 按内嵌 Markdown 模板填充 |
| 4 | dry-run 预览 | 输出完整 Release Notes |
| 5 | 创建 Release | `gitlink-cli release +create --name --tag --target master --body <notes>` |
| 6 | 验证 | 确认 Release 已创建 |

### Issue 分类逻辑

| 分类组 | GitLink/Gitee 标签 | GitHub/GitLab 标签 |
|--------|-------------------|-------------------|
| Bug 修复 | 缺陷 | bug |
| 新功能 | 功能 | enhancement, feature |
| 其他改进 | 不属于以上两组 | 不属于以上两组 |

### 版本命名

默认 `vYYYY.MM.DD`（如 `v2026.07.10`）。当天已有同名 tag 时追加序号 `-2`、`-3`。用户也可指定版本号（如 `v1.2.0`）。

---

## 五、Skill 4 — gitlink-community-ops（统一入口）

### 编排逻辑

```
Step 0 → 环境检测 + 参数收集 (OWNER/REPO/BACKEND/PERIOD/VERSION)
Step 1 → 加载并执行 gitlink-issue-triage-rules → 分拣 Issue
Step 2 → 加载并执行 gitlink-community-report → 生成周报
Step 3 → 加载并执行 gitlink-release-notes → 发布 Release
Step 4 → 输出工作流总结
```

**编排规则：**
- **顺序执行**：Step 1 必须先完成（分拣后才有标签数据供周报和 Release Notes 使用）
- **逐个确认**：每个子 Skill 的写回操作需独立确认
- **上下文传递**：OWNER / REPO / BACKEND 在子 Skill 间自动继承
- **容错继续**：某子 Skill 失败不阻塞后续步骤
- **可选跳过**：用户可只执行部分步骤

---

## 六、串联的 CLI 命令 / Skill 调用清单（≥3 ✓）

本工作流串联了 **4 个 Skill + 12 类 gitlink-cli 命令**：

| # | 类型 | 命令/调用 | 阶段 | 作用 |
|---|------|----------|------|------|
| **1** | **Skill** | **gitlink-issue-triage-rules** | **1** | **Issue 自动分类+分配** |
| **2** | **Skill** | **gitlink-community-report** | **2** | **社区周报生成+发布** |
| **3** | **Skill** | **gitlink-release-notes** | **3** | **Release Notes 生成+发布** |
| **4** | **Skill** | **gitlink-community-ops** | **编排** | **统一入口，编排1-3** |
| 5 | CLI | `auth status` | 1/2/3 | 认证检测 |
| 6 | CLI | `issue +list` | 1/2/3 | 拉取 Issue 列表 |
| 7 | CLI | `label +list` | 1 | 标签 ID 映射 |
| 8 | CLI | `member +list` | 1 | 成员映射 |
| 9 | CLI | `issue +view` | 1 | 写回前回读 (CRITICAL) |
| 10 | CLI | `api PATCH` | 1 | 写回标签/优先级/责任人 |
| 11 | CLI | `issue +comment` | 1 | 审计评论 |
| 12 | CLI | `issue +create` | 2 | 发布周报 |
| 13 | CLI | `release +list` | 2/3 | 统计/去重 |
| 14 | CLI | `release +create` | 3 | 发布 Release |

---

## 七、可复现的执行方式

### 环境要求
- 服务器已安装 `gitlink-cli` 并 `gitlink-cli auth status` 显示已登录
- 四个 Skill 的 SKILL.md 文件已安装到 Agent 的 skills 目录
- 目标仓库已有标签和成员

### Skill 安装位置

```
~/.workbuddy/skills/                          # WorkBuddy Agent或其他AI Agent
  ├── gitlink-issue-triage-rules/SKILL.md     
  ├── gitlink-community-report/SKILL.md       
  ├── gitlink-release-notes/SKILL.md          
  └── gitlink-community-ops/SKILL.md          (统一入口)

# 或项目级:
<project>/.workbuddy/skills/
  ├── gitlink-issue-triage-rules/SKILL.md
  ├── gitlink-community-report/SKILL.md
  ├── gitlink-release-notes/SKILL.md
  └── gitlink-community-ops/SKILL.md
```

### 执行方式

**方式 A：WorkBuddy / Agent 对话触发**

```
# 全流程 (一键)
"帮我跑 Angel123456/gitlink-cli 的完整社区运营工作流"

# 单阶段
"帮我分拣 Angel123456/gitlink-cli 的未分拣 Issue"       → 只触发 gitlink-issue-triage-rules
"帮我生成 Angel123456/gitlink-cli 的社区周报"            → 只触发 gitlink-community-report
"帮我发布 Angel123456/gitlink-cli 的 Release Notes"     → 只触发 gitlink-release-notes

# 部分流程
"帮我分拣 Issue，然后生成周报"                            → Step 1 + Step 2
```

### 定期自动化

**WorkBuddy Automation**

使用 WorkBuddy 的定时自动化功能，每周一自动触发 `gitlink-community-ops` Skill。

---

## 八、真实运行效果

### Step 1 — Issue 自动分类 (gitlink-issue-triage-rules, 2026-07-10)

Agent 按 SKILL.md 步骤执行，hybrid 模式（规则匹配 + LLM 兜底）：

| Issue | 标题 | 命中规则 | 标签 | 优先级 | 责任人 |
|-------|------|---------|------|--------|--------|
| #2 | 建议支持 Markdown 格式的评论 | LLM 兜底 | 功能 | 正常 | Angel123456 |
| #5 | 建议支持 JSON 格式日志输出 | LLM 兜底 | 功能 | 正常 | Angel123456 |
| #7 | 建议支持 Markdown 格式的评论 | LLM 兜底 ⚠️(与#2重复) | 重复 | 正常 | Angel123456 |

> 写回方式：`gitlink-cli api PATCH`，写回前先 `+view` 回读防清空，写回后 `+comment` 审计。

### Step 2 — 社区周报 (gitlink-community-report, 2026-07-10)

周报标题：`📊 社区周报 2026-07-10`，发布为 Issue #9。
包含：概览（开放8/关闭0）、本周新建Issue清单、标签分布、高优先级待办、本周Release。

### Step 3 — Release Notes (gitlink-release-notes, 2026-07-10)

版本号 `v2026.07.10`，记录了 Step 1-2 的社区运营改进。
Release 已创建并验证（总 Release: 2）。

---

## 九、文件清单

```
community-ops/
├── README.md                                    # 本说明文档
├── SKILL.md                                     # gitlink-issue-triage-rules Skill 指令文件
├── skills/                                      # Skill 文件目录
│   ├── gitlink-community-report/SKILL.md        # 社区周报 Skill
│   ├── gitlink-release-notes/SKILL.md           # Release Notes Skill
│   └── gitlink-community-ops/SKILL.md           # 统一入口 Skill
├── logs/
│   ├── step1-issue-triage.log                   # Step 1 Issue 分拣日志
│   ├── step2-community-report.log               # Step 2 周报日志
│   └── step3-release-notes.log                  # Step 3 Release Notes 日志
└── docs/
    └── architecture.html                        # 架构图
```

---

## 十、安全策略

- **默认 dry-run**：所有子 Skill 默认先预览，用户确认后才执行写回
- **逐个确认**：每个子 Skill 的写回操作需独立确认（不在编排层一次性全确认）
- **view 回读**：Step 1 写回前必须 `+view` 回读标题和正文（防清空）
- **规则可审计**：每次写回附带审计评论，记录命中规则和匹配方式
- **版本去重**：Step 3 自动检测已有 Release，避免 tag 冲突
- **容错继续**：某步骤失败不阻塞后续
