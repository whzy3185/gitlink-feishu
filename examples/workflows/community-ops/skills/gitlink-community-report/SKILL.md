---
name: gitlink-community-report
version: 1.0.0
description: "社区周报自动生成：在 GitHub / GitLink / GitLab / Gitee 等开源项目中，自动汇总本周 Issue 开闭情况、标签分布、高优先级待办、Release 动态，生成结构化周报并发布为 Issue。触发场景：生成社区周报、社区运营报告、每周总结、weekly report、社区健康度检查。"
license: MulanPSL-2.0
metadata:
  requires:
    bins_any: ["gitlink-cli", "gh", "glab", "curl"]
    bins_note: "gitee 后端无官方 CLI，直接使用 curl 调用 https://gitee.com/api/v5/ REST API；其余三后端用对应原生 CLI"
  cliHelp: "gitlink-cli issue --help"
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
---

# gitlink-community-report（社区周报自动生成）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/skills/gitlink-shared/SKILL.md)（仅 GitLink 后端）或对应平台的 CLI 文档。所有 GitHub 操作必须使用 `gh`；所有 GitLab 操作必须使用 `glab`；所有 GitLink 操作必须使用 `gitlink-cli`。禁止混用或替代。**
**CRITICAL — 本 Skill 包含写入操作（创建周报 Issue）。默认必须先做 dry-run（仅预览），经用户明确确认后才执行发布。**

> **依赖工具：** 二选一即可——`gitlink-cli` / `gh` / `glab`，外加 `jq`。
> **依赖 Skill：** 使用 GitLink 后端时需加载 `gitlink-shared`；使用 GitHub / GitLab 时无需额外 Skill。
> **本 Skill 为 C1 模式（纯文档）**：所有命令由 Agent 按本文件步骤直接调用对应平台 CLI 执行。

---

## 功能概述

本 Skill 是**自包含**的：周报模板与分类逻辑**直接内嵌在 SKILL.md 中**，Agent 从本文件读取格式规范，不需要从外部仓库拉取配置文件。

1. **后端检测**：Agent 自动识别当前仓库所在的平台（GitHub / GitLink / GitLab / Gitee）
2. **数据采集**：Agent 用对应 CLI 收集开放/关闭 Issue、Release、标签分布、高优先级待办
3. **时间过滤**：按用户指定时间范围（默认本周）筛选新建/关闭 Issue
4. **报告生成**：Agent 按内嵌模板生成结构化 Markdown 周报
5. **dry-run 预览**：Agent 输出周报内容，等待用户确认
6. **发布**：用户确认后，Agent 用对应 CLI 创建周报 Issue
7. **可追踪**：周报 Issue 标题带日期标识，后续可自动识别跳过（不纳入下次分拣）

> **⚠️ 设计约束**：周报模板内嵌于 SKILL.md。如需自定义章节/格式，编辑本 SKILL.md 的「周报模板」小节即可。

## 触发场景

用户提到以下关键词时自动触发：
- "生成社区周报"、"社区运营报告"、"每周总结"
- "weekly report"、"community report"、"社区健康度"
- "帮我看看本周社区动态"

---

## 平台适配

### 后端自动检测

Agent 启动周报任务时，**必须**先按以下顺序检测后端：

```bash
detect_backend() {
  local remote_url
  remote_url="$(git remote get-url origin 2>/dev/null || echo '')"

  case "$remote_url" in
    *gitlink.org.cn*)        echo "gitlink" ;;
    *github.com*)            echo "github"  ;;
    *gitlab.com*|*gitlab.*)  echo "gitlab"  ;;
    *gitee.com*)             echo "gitee"   ;;
  esac

  command -v gitlink-cli >/dev/null && echo "gitlink"
  command -v gh          >/dev/null && echo "github"
  command -v glab        >/dev/null && echo "gitlab"
  command -v curl        >/dev/null && echo "gitee"
}
```

### 命令映射表

| 步骤 | 目的 | GitLink (`gitlink-cli`) | GitHub (`gh`) | GitLab (`glab`) | Gitee (`curl`) |
|------|------|------------------------|----------------|------------------|----------------|
| **认证检查** | 确认已登录 | `gitlink-cli auth status` | `gh auth status` | `glab auth status` | `test -n "$GITEE_TOKEN"` |
| **列开放 Issue** | 取开放 Issue | `gitlink-cli issue +list --owner X --repo Y --state open --format json` | `gh issue list --repo X/Y --state open --json number,title,labels,body,createdAt` | `glab issue list --repo X/Y --state opened --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues?state=open&access_token=$GITEE_TOKEN"` |
| **列关闭 Issue** | 取关闭 Issue | `gitlink-cli issue +list --owner X --repo Y --state closed --format json` | `gh issue list --repo X/Y --state closed --json number,title,labels,closedAt` | `glab issue list --repo X/Y --state closed --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues?state=closed&access_token=$GITEE_TOKEN"` |
| **列 Release** | 取已有版本 | `gitlink-cli release +list --owner X --repo Y --format json` | `gh release list --repo X/Y --json tagName,name,publishedAt` | `glab release list --repo X/Y --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/releases?access_token=$GITEE_TOKEN"` |
| **列成员** | 取贡献者 | `gitlink-cli member +list --owner X --repo Y --format json` | `gh api repos/X/Y/collaborators` | `glab api projects/:fullpath/members/all` | `curl -s "https://gitee.com/api/v5/repos/X/Y/collaborators?access_token=$GITEE_TOKEN"` |
| **创建 Issue** | 发布周报 | `gitlink-cli issue +create --owner X --repo Y --title "..." --body "..." --format json` | `gh issue create --repo X/Y --title "..." --body "..."` | `glab issue create --repo X/Y --title "..." --description "..."` | `curl -X POST "https://gitee.com/api/v5/repos/X/Y/issues?access_token=$GITEE_TOKEN&title=...&body=..."` |

### 字段名差异

| 概念 | GitLink | GitHub | GitLab | Gitee |
|------|---------|--------|--------|-------|
| Issue 标题 | `subject` | `title` | `title` | `title` |
| Issue 正文 | `description` | `body` | `description` | `body` |
| 创建时间 | `created_at` | `createdAt` | `created_at` | `created_at` |
| 更新时间 | `updated_at` | `updatedAt` | `updated_at` | `updated_at` |
| 标签 | `issue_tags` (数组 of {name,id}) | `labels` (数组 of {name}) | `labels` (数组 of {name}) | `labels` (逗号分隔) |
| 优先级 | `priority_id` (1-4) | 靠 label 表达 | 靠 label 表达 | 靠 label 表达 |

---

## 工作流（Agent 执行步骤）

### Step 0：确认环境 + 检测后端

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

# 0.3 确认参数
# Agent 需从用户输入或上下文获取:
#   OWNER, REPO, PERIOD (默认 "本周" = 7天)
echo "仓库: $OWNER/$REPO | 周期: 最近 $PERIOD 天"
```

### Step 1：采集数据（按后端选命令）

```bash
case "$BACKEND" in
  gitlink)
    OPEN=$(gitlink-cli issue +list --owner $OWNER --repo $REPO --state open --limit 200 --format json)
    CLOSED=$(gitlink-cli issue +list --owner $OWNER --repo $REPO --state closed --limit 200 --format json)
    RELEASES=$(gitlink-cli release +list --owner $OWNER --repo $REPO --format json)
    MEMBERS=$(gitlink-cli member +list --owner $OWNER --repo $REPO --format json)
    ;;
  github)
    OPEN=$(gh issue list --repo $OWNER/$REPO --state open --limit 200 --json number,title,labels,createdAt)
    CLOSED=$(gh issue list --repo $OWNER/$REPO --state closed --limit 200 --json number,title,labels,closedAt)
    RELEASES=$(gh release list --repo $OWNER/$REPO --json tagName,name,publishedAt --limit 50)
    ;;
  gitlab)
    OPEN=$(glab issue list --repo $OWNER/$REPO --state opened --output json --all)
    CLOSED=$(glab issue list --repo $OWNER/$REPO --state closed --output json --all)
    RELEASES=$(glab release list --repo $OWNER/$REPO --output json)
    ;;
  gitee)
    OPEN=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?state=open&access_token=$GITEE_TOKEN&per_page=200")
    CLOSED=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?state=closed&access_token=$GITEE_TOKEN&per_page=200")
    RELEASES=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/releases?access_token=$GITEE_TOKEN")
    ;;
esac
```

### Step 2：时间过滤 + 统计

Agent 根据采集到的数据，计算以下指标：

```
统计指标:
  1. 开放 Issue 总数       ← OPEN 数组长度
  2. 已关闭 Issue 总数     ← CLOSED 数组长度
  3. 本周新建 Issue       ← created_at >= (今天 - PERIOD天)
  4. 本周关闭 Issue       ← updated_at/closed_at >= (今天 - PERIOD天)
  5. 标签分布             ← OPEN Issue 的标签 name → 计数
  6. 高优先级待办         ← OPEN Issue 中 priority_id >= 3 (GitLink) / label 含 "priority: high/critical" (GitHub/GitLab)
  7. 本周 Release         ← RELEASES 中 published_at >= (今天 - PERIOD天)
  8. 贡献者/成员          ← MEMBERS 数量 + 角色
```

**时间字段映射**（不同平台字段名不同，Agent 按后端做翻译）：

| 平台 | 创建时间字段 | 关闭时间字段 | 发布时间字段 |
|------|-------------|-------------|-------------|
| GitLink | `created_at` | `updated_at` (状态变更时更新) | `created_at` |
| GitHub | `createdAt` | `closedAt` | `publishedAt` |
| GitLab | `created_at` | `closed_at` | `created_at` |
| Gitee | `created_at` | `updated_at` | `created_at` |

### Step 3：生成周报内容（按内嵌模板）

Agent 按以下模板生成 Markdown 周报。**所有数值占位符由 Agent 在 Step 2 计算后填入**：

```markdown
# 社区周报 | {OWNER}/{REPO}

**统计周期:** {PERIOD_START} ~ {PERIOD_END}
**生成时间:** {NOW} (由 gitlink-community-report 自动生成)

## 概览

- 开放 Issue: **{OPEN_COUNT}**
- 已关闭 Issue: **{CLOSED_COUNT}**
- 本周新建: **{NEW_COUNT}**
- 本周关闭: **{CLOSED_WEEK_COUNT}**
- 累计 Release: **{RELEASE_COUNT}**
- 本周新 Release: **{NEW_RELEASE_COUNT}**
- 贡献者/成员: **{MEMBER_COUNT}**

## 本周新建 Issue

{NEW_ISSUES_LIST}
<!-- 格式: - #{number} [{tags}] {subject/title} -->

## 本周关闭 Issue

{CLOSED_WEEK_LIST}
<!-- 格式: - #{number} {subject/title} -->

## 当前开放 Issue 标签分布

{TAG_DISTRIBUTION}
<!-- 格式: - {tag_name}: {count} -->

## 高优先级待办

{HIGH_PRIORITY_LIST}
<!-- 格式: - #{number} [优先级: {priority}] {subject/title} -->

## 本周 Release

{WEEK_RELEASES}
<!-- 格式: - {tag_name} ({published_date}) -->

---

_本周报由 gitlink-community-report Skill 自动生成_
```

### Step 4：dry-run 预览（必须）

Agent 输出完整周报 Markdown，附带元信息：

```markdown
## 📊 社区周报预览（dry-run）

> 后端：{BACKEND}
> 仓库：{OWNER}/{REPO}
> 统计周期：最近 {PERIOD} 天
> 开放 Issue：{count} 条 | 已关闭：{count} 条

{完整周报 Markdown 内容}

⚠️ 涉及写入操作（创建 Issue）。回复"确认"或"apply"执行。
```

### Step 5：发布周报（用户确认后，按后端分发）

```bash
TITLE="📊 社区周报 $(date +%Y-%m-%d)"

case "$BACKEND" in
  gitlink)
    gitlink-cli issue +create --owner $OWNER --repo $REPO \
      --title "$TITLE" --body "$REPORT_MD" --format json
    ;;
  github)
    gh issue create --repo $OWNER/$REPO \
      --title "$TITLE" --body "$REPORT_MD"
    ;;
  gitlab)
    glab issue create --repo $OWNER/$REPO \
      --title "$TITLE" --description "$REPORT_MD"
    ;;
  gitee)
    curl -X POST "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?access_token=$GITEE_TOKEN" \
      -d "title=$TITLE" -d "body=$REPORT_MD"
    ;;
esac
```

**周报 Issue 标识**：标题以 `📊 社区周报` 开头，便于后续 Skill 自动识别并跳过（不纳入 Issue 分拣）。

### Step 6：验证

```bash
# 确认周报 Issue 已创建
case "$BACKEND" in
  gitlink) gitlink-cli issue +list --owner $OWNER --repo $REPO --state open --format json \
    | jq '[.data[] | select(.subject | startswith("📊"))]' ;;
  github)  gh issue list --repo $OWNER/$REPO --state open --json number,title \
    | jq '[.[] | select(.title | startswith("📊"))]' ;;
  gitlab)  glab issue list --repo $OWNER/$REPO --state opened --output json \
    | jq '[.[] | select(.title | startswith("📊"))]' ;;
  gitee)   curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?state=open&access_token=$GITEE_TOKEN" \
    | jq '[.[] | select(.title | startswith("📊"))]' ;;
esac
```

---

## 时间范围配置

默认统计最近 7 天。用户可指定：

| 输入 | PERIOD | 说明 |
|------|--------|------|
| "本周周报" / "weekly report" | 7 | 最近 7 天 |
| "半月报" / "biweekly" | 14 | 最近 14 天 |
| "月度报告" / "monthly" | 30 | 最近 30 天 |
| "自定义 N 天" | N | 最近 N 天 |

---

## 周报模板（内嵌）

以下是完整的周报 Markdown 模板。Agent 按此结构生成周报，所有 `{PLACEHOLDER}` 由 Step 2 计算结果替换：

```markdown
# 社区周报 | {OWNER}/{REPO}

**统计周期:** {PERIOD_START} ~ {PERIOD_END}
**生成时间:** {NOW} (由 gitlink-community-report 自动生成)

## 概览

- 开放 Issue: **{OPEN_COUNT}**
- 已关闭 Issue: **{CLOSED_COUNT}**
- 本周新建: **{NEW_COUNT}**
- 本周关闭: **{CLOSED_WEEK_COUNT}**
- 累计 Release: **{RELEASE_COUNT}**
- 本周新 Release: **{NEW_RELEASE_COUNT}**
- 贡献者/成员: **{MEMBER_COUNT}**

## 本周新建 Issue

{NEW_ISSUES_LIST}

## 本周关闭 Issue

{CLOSED_WEEK_LIST}

## 当前开放 Issue 标签分布

{TAG_DISTRIBUTION}

## 高优先级待办

{HIGH_PRIORITY_LIST}

## 本周 Release

{WEEK_RELEASES}

---

_本周报由 gitlink-community-report Skill 自动生成_
```

---

## 安全与写回策略

| 规则 | 说明 |
|------|------|
| **必走 dry-run** | Step 4 输出预览，未确认前**禁止** 发布 |
| **周报标识** | 标题前缀 `📊 社区周报`，便于其他 Skill 识别跳过 |
| **限流** | 数据采集 >100 条时，每次 CLI 调用后 sleep 200ms |
| **错误透明** | 任何 4xx/5xx 必须打印响应体，不要吞错 |
| **空数据容错** | 某分类无数据时输出"无"，不跳过章节 |

---

## 使用示例

### 示例 1：GitLink 项目

**用户输入**：

```
帮我生成 Angel123456/gitlink-cli 的本周社区周报
```

**Agent 执行**：

```bash
# Step 0
gitlink-cli auth status

# Step 1
gitlink-cli issue +list --owner Angel123456 --repo gitlink-cli --state open --limit 200 --format json
gitlink-cli issue +list --owner Angel123456 --repo gitlink-cli --state closed --limit 200 --format json
gitlink-cli release +list --owner Angel123456 --repo gitlink-cli --format json
gitlink-cli member +list --owner Angel123456 --repo gitlink-cli --format json

# Step 2-3: 计算统计 → 按模板生成周报

# Step 4: 输出 dry-run 预览

# Step 5 (用户确认后):
gitlink-cli issue +create --owner Angel123456 --repo gitlink-cli \
  --title "📊 社区周报 2026-07-10" \
  --body "$REPORT_MD" --format json

# Step 6: 验证
gitlink-cli issue +list --owner Angel123456 --repo gitlink-cli --state open --format json \
  | jq '[.data[] | select(.subject | startswith("📊"))]'
```

### 示例 2：GitHub 项目

**用户输入**：

```
生成 xuanlanwuta/gps_SM 的半月报
```

**Agent 执行**：

```bash
# Step 0
gh auth status

# Step 1
gh issue list --repo xuanlanwuta/gps_SM --state open --limit 200 --json number,title,labels,createdAt
gh issue list --repo xuanlanwuta/gps_SM --state closed --limit 200 --json number,title,labels,closedAt
gh release list --repo xuanlanwuta/gps_SM --json tagName,name,publishedAt

# Step 2-3: PERIOD=14 → 计算统计 → 生成周报

# Step 5 (用户确认后):
gh issue create --repo xuanlanwuta/gps_SM \
  --title "📊 社区半月报 2026-07-10" \
  --body "$REPORT_MD"
```

---

## Agent 平台兼容性

| 平台 | 加载方式 | 触发方式 |
|------|----------|----------|
| **OpenClaw** | `workspace/skills/gitlink-community-report/SKILL.md` | 自然语言 |
| **Claude Code** | `~/.claude/skills/gitlink-community-report/SKILL.md` | 自然语言 |
| **Cursor** | `~/.cursor/skills/gitlink-community-report/SKILL.md` | 自然语言 + `/` 命令 |
| **WorkBuddy** | `~/.workbuddy/skills/gitlink-community-report/SKILL.md` | 自然语言 |
| **通用 Agent** | 作为参考文档 | 按 SKILL.md 流程自取 |

---

## 与 gitlink-community-ops 的关系

本 Skill 可**独立使用**，也可作为 `gitlink-community-ops` 统一入口 Skill 的子 Skill 被编排调用：

- 独立触发：用户说"生成社区周报"
- 编排触发：`gitlink-community-ops` Step 2 指令"加载并执行 gitlink-community-report"

当作为子 Skill 被调用时，Agent 需在上下文中继承 `OWNER`、`REPO`、`BACKEND`、`PERIOD` 参数。
