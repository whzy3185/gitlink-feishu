---
name: gitlink-issue-triage-rules
version: 1.0.0
description: "跨平台 Issue 规则驱动分拣：在 GitHub / GitLink / GitLab 等开源项目中，按 .triage/rules.yml 配置自动为 Issue 分类、打标签、分配责任人。三种运行模式：rule（纯规则）/ hybrid（规则+LLM 兜底）/ ai（纯 AI 即兴判断）。触发场景：批量分拣 Issue、规范化标签分配、降低维护成本、CI/夜间自动化。"
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
        raw_template: "https://www.gitlink.org.cn/{owner}/{repo}/raw/master/{path}"
        api_base: "https://www.gitlink.org.cn/api/v1"
        auth_env: GITLINK_TOKEN
        default: true
      - id: github
        cli: gh
        url_template: "https://github.com/{owner}/{repo}"
        raw_template: "https://raw.githubusercontent.com/{owner}/{repo}/master/{path}"
        api_base: "https://api.github.com"
        auth_env: GH_TOKEN
      - id: gitlab
        cli: glab
        url_template: "https://gitlab.com/{owner}/{repo}"
        raw_template: "https://gitlab.com/{owner}/{repo}/-/raw/master/{path}"
        api_base: "https://gitlab.com/api/v4"
        auth_env: GITLAB_TOKEN
      - id: gitee
        cli: curl                              # Gitee 无官方 CLI，用 curl 直调 API
        url_template: "https://gitee.com/{owner}/{repo}"
        raw_template: "https://gitee.com/{owner}/{repo}/raw/master/{path}"
        api_base: "https://gitee.com/api/v5"
        auth_env: GITEE_TOKEN
        auth_header: "access_token"            # Gitee API 用 ?access_token=xxx 传鉴权
---

# gitlink-issue-triage-rules（跨平台 Issue 规则驱动分拣）

## **CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/skills/gitlink-shared/SKILL.md)（仅 GitLink 后端）或对应平台的 CLI 文档。所有 GitHub 操作必须使用 `gh`；所有 GitLab 操作必须使用 `glab`；所有 GitLink 操作必须使用 `gitlink-cli`。禁止混用或替代。**
## **CRITICAL — 本 Skill 包含写入操作（打标签、分配责任人、设置优先级、添加评论）。默认必须先做 dry-run（仅预览），经用户明确确认后才执行写回。**
## **CRITICAL — 更新 Issue 时必须先 `view` 获取当前标题和正文，并在写回请求中一并提交，否则这两个字段会被清空（不同平台字段名不同，详见下文「平台适配」）。**

> **依赖工具：** 二选一即可——`gitlink-cli` / `gh` / `glab`，外加 `jq`。
> **依赖 Skill：** 使用 GitLink 后端时需加载 `gitlink-shared`；使用 GitHub / GitLab 时无需额外 Skill。
> **本 Skill 为 C1 模式（纯文档）**：所有命令由 Agent 按本文件步骤直接调用对应平台 CLI 执行。

---

## 功能概述

本 Skill 是**自包含**的：分拣规则 JSON **直接内嵌在 SKILL.md 的「机器可读规则块」章节中**，Agent 或自动化脚本从本文件读取规则，不需要从外部仓库拉取、不需要创建本地文件。下方 YAML 仍作为 Schema 示例保留，便于人类阅读。

1. **后端检测**：Agent 自动识别当前仓库所在的平台（GitHub / GitLink / GitLab / Gitee）
2. **读取规则**：Agent 直接从本 SKILL.md 的内嵌 JSON 块读取规则（见下文「机器可读规则块」小节）
3. **匹配规则**：对每条 Issue 按规则评分；`mode: rule` 直接采纳；`mode: hybrid` 规则无命中时由 LLM 兜底；`mode: ai` 规则仅作为 prompt 提示
4. **dry-run 预览**：Agent 输出分拣建议，等待用户确认
5. **写回**：用户确认后，Agent 用对应平台 CLI 执行写回（PATCH / edit / update / curl）
6. **可审计**：每次写入附带 `matched_rules` 字段（写入 Issue comment 中），方便人工复核

> **⚠️ 设计约束**：本 Skill **不走仓库 `.triage/rules.yml` 文件**，规则全部内嵌于 SKILL.md。这样 Skill 可在任意仓库复用、零外部依赖。

## 机器可读规则块（供自动化工作流读取）

下面的 JSON 块是本 Skill 的稳定规则接口。社区运营自动化工作流会直接读取该块完成 Issue 分类、标签候选、优先级和负责人建议。

<!-- TRIAGE_RULES_JSON_START -->
```json
{
  "version": 1,
  "mode": "rule",
  "skill": {
    "name": "gitlink-issue-triage-rules",
    "version": "1.0.0"
  },
  "defaults": {
    "dry_run": true,
    "priority": "P3",
    "skip_when": {
      "has_label_any": ["wontfix", "duplicate", "重复", "已关闭"]
    }
  },
  "priority_ids": {
    "P0": 4,
    "P1": 3,
    "P2": 2,
    "P3": 1,
    "critical": 4,
    "high": 3,
    "normal": 2,
    "low": 1
  },
  "labels_by_type": {
    "bug": ["缺陷", "bug"],
    "feature": ["功能", "enhancement"],
    "question": ["疑问", "question"],
    "docs": ["文档", "documentation"],
    "security": ["缺陷", "security"],
    "performance": ["缺陷", "performance"],
    "ci": ["测试", "ci"],
    "refactor": ["任务", "refactor"],
    "duplicate": ["重复", "duplicate"]
  },
  "members": [
    {
      "id": 153579,
      "login": "Angel123456",
      "name": "Angel123456",
      "role": "Manager",
      "role_name": "管理员"
    },
    {
      "id": 134328,
      "login": "Hsy15889521073",
      "name": "Hsy15889521073",
      "role": "Developer",
      "role_name": "开发者"
    },
    {
      "id": 134450,
      "login": "Priziq",
      "name": "Priziq",
      "role": "Reporter",
      "role_name": "报告者"
    }
  ],
  "assignment_strategy": "type-map-config",
  "assigners_by_type": {
    "security": [153579],
    "bug": [153579],
    "performance": [153579],
    "feature": [134328],
    "refactor": [134328],
    "ci": [134328],
    "question": [134450],
    "docs": [134450],
    "duplicate": [134450]
  },
  "rules": [
    {
      "id": "security-sensitive",
      "type": "security",
      "label": ["缺陷", "security"],
      "priority": "critical",
      "match": {
        "any_keyword": ["漏洞", "安全", "权限", "认证", "token", "secret", "泄露", "CVE", "SQL 注入", "XSS", "越权"]
      }
    },
    {
      "id": "duplicate-detect",
      "type": "duplicate",
      "label": ["重复", "duplicate"],
      "priority": "low",
      "match": {
        "any_keyword": ["重复", "duplicate", "同问题", "如 #", "same as", "duplicated"]
      }
    },
    {
      "id": "bug-crash",
      "type": "bug",
      "label": ["缺陷", "bug"],
      "priority": "high",
      "match": {
        "any_keyword": ["崩溃", "闪退", "panic", "crash", "无法启动", "无法登录", "安装失败"]
      }
    },
    {
      "id": "performance-default",
      "type": "performance",
      "label": ["缺陷", "performance"],
      "priority": "normal",
      "match": {
        "any_keyword": ["性能", "很慢", "卡顿", "延迟", "内存", "CPU", "cpu", "latency", "slow", "超时"]
      }
    },
    {
      "id": "ci-build",
      "type": "ci",
      "label": ["测试", "ci"],
      "priority": "normal",
      "match": {
        "any_keyword": ["CI", "ci", "构建失败", "build failed", "pipeline", "测试失败", "单测失败", "lint"]
      }
    },
    {
      "id": "bug-default",
      "type": "bug",
      "label": ["缺陷", "bug"],
      "priority": "high",
      "match": {
        "any_keyword": ["bug", "错误", "失败", "异常", "报错", "问题", "不可用", "无响应", "不一致", "消失", "变红色", "缺少", "缺失", "不替换", "未生效", "不能", "无法", "timeout", "error", "failed"]
      }
    },
    {
      "id": "feature-request",
      "type": "feature",
      "label": ["功能", "enhancement"],
      "priority": "normal",
      "match": {
        "any_keyword": ["建议", "希望", "支持", "是否支持", "新增", "增加", "需要", "让", "自动", "需求", "feature", "enhancement", "request"]
      }
    },
    {
      "id": "question-default",
      "type": "question",
      "label": ["疑问", "question"],
      "priority": "low",
      "match": {
        "any_keyword": ["请问", "如何", "怎么", "是否", "讨论", "反馈", "how to", "求助", "咨询", "使用方式"]
      }
    },
    {
      "id": "docs-default",
      "type": "docs",
      "label": ["文档", "documentation"],
      "priority": "low",
      "match": {
        "any_keyword": ["文档", "README", "readme", "typo", "错别字", "说明", "教程", "示例"]
      }
    },
    {
      "id": "refactor-default",
      "type": "refactor",
      "label": ["任务", "refactor"],
      "priority": "low",
      "match": {
        "any_keyword": ["重构", "清理", "优化结构", "refactor", "cleanup", "技术债"]
      }
    }
  ]
}
```
<!-- TRIAGE_RULES_JSON_END -->

## 触发场景

用户提到以下关键词时自动触发：
- "用规则分拣 Issue"、"按规则打标签"、"批量分拣"
- "triage-rules.yml"、".triage/rules.yml"、"配置化分拣"
- "自动分配责任人"、"定期巡检 Issue"

---

## 平台适配（核心）

### 后端自动检测

Agent 启动分拣任务时，**必须**先按以下顺序检测后端：

```bash
# 检测当前仓库所在平台
detect_backend() {
  local remote_url
  remote_url="$(git remote get-url origin 2>/dev/null || echo '')"

  # 1. 优先看 remote URL
  case "$remote_url" in
    *gitlink.org.cn*)        echo "gitlink" ;;
    *github.com*)            echo "github"  ;;
    *gitlab.com*|*gitlab.*)  echo "gitlab"  ;;
    *gitee.com*)             echo "gitee"   ;;
  esac

  # 2. 回退：检查可用 CLI
  if ! command -v gitlink-cli >/dev/null && ! command -v gh >/dev/null && \
     ! command -v glab >/dev/null && ! command -v curl >/dev/null; then
    echo "ERROR: 未找到支持的 CLI（gitlink-cli / gh / glab / curl）" >&2
    return 1
  fi

  # 3. 最后回退：依次尝试
  command -v gitlink-cli >/dev/null && echo "gitlink"
  command -v gh          >/dev/null && echo "github"
  command -v glab        >/dev/null && echo "gitlab"
  command -v curl        >/dev/null && echo "gitee"
}
```

检测结果决定后续所有命令使用哪一组 CLI。

### 命令映射表

不同后端命令差异较大。下表是**完整的命令对照**——Agent 按当前后端选对应一列执行：

| 步骤 | 目的 | GitLink (`gitlink-cli`) | GitHub (`gh`) | GitLab (`glab`) | Gitee (`curl` + REST) |
|------|------|------------------------|----------------|------------------|------------------------|
| **认证检查** | 确认已登录 | `gitlink-cli auth status` | `gh auth status` | `glab auth status` | `test -n "$GITEE_TOKEN" && echo OK` |
| **列 Issue** | 取开放 Issue | `gitlink-cli issue +list --owner X --repo Y --state open --format json` | `gh issue list --repo X/Y --state open --json number,title,labels,assignees,body,state` | `glab issue list --repo X/Y --state opened --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues?state=open&access_token=$GITEE_TOKEN" \| jq` |
| **查 Issue** | 单 Issue 详情 | `gitlink-cli issue +view --owner X --repo Y --number N --format json` | `gh issue view N --repo X/Y --json number,title,body,labels,assignees,state` | `glab issue view N --repo X/Y --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues/N?access_token=$GITEE_TOKEN" \| jq` |
| **列标签** | 建立 label_name → id 映射 | `gitlink-cli label +list --owner X --repo Y --format json` | `gh label list --repo X/Y --json name,id,color` | `glab label list --repo X/Y --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/labels?access_token=$GITEE_TOKEN" \| jq` |
| **列成员** | 建立 login → user_id 映射 | `gitlink-cli member +list --owner X --repo Y --format json` | `gh api repos/X/Y/collaborators --jq '.[] \| {login,id}'` | `glab api projects/:fullpath/members/all --jq '.[] \| {username,id}'` | `curl -s "https://gitee.com/api/v5/repos/X/Y/collaborators?access_token=$GITEE_TOKEN" \| jq` |
| **写回 Issue** | 打标签 + 分配责任人 + 优先级 | `gitlink-cli api PATCH /v1/X/Y/issues/N --body '{...}'` | `gh issue edit N --repo X/Y --add-label "bug,priority:high" --add-assignee user1` | `glab issue update N --repo X/Y --label "bug" --assignee user1` | `curl -X PATCH "https://gitee.com/api/v5/repos/X/Y/issues/N?access_token=$GITEE_TOKEN&labels=bug,priority:high&assignees=user1"` |
| **添加评论** | 写审计评论 | `gitlink-cli issue +comment --number N --body "..."` | `gh issue comment N --repo X/Y --body "..."` | `glab issue note N --repo X/Y --message "..."` | `curl -X POST "https://gitee.com/api/v5/repos/X/Y/issues/N/comments?access_token=$GITEE_TOKEN&body=..."` |

### 写回字段名差异

不同平台字段名不同，Agent **必须按后端做翻译**：

| 概念 | GitLink 字段 | GitHub 字段 | GitLab 字段 | Gitee 字段 |
|------|--------------|-------------|--------------|------------|
| 标题 | `subject` | `title` | `title` | `title` |
| 正文 | `description` | `body` | `description` | `body` |
| 标签 | `issue_tag_ids`（数组 of ID） | `labels`（数组 of name） | `labels`（数组 of name） | `labels`（数组 of name，逗号分隔） |
| 责任人 | `assigner_ids`（数组 of user_id） | `assignees`（数组 of login） | `assignees`（数组 of username） | `assignees`（数组 of login，逗号分隔） |
| 优先级 | `priority_id`（1-4） | 无原生字段，靠 label 表达 | 无原生字段，靠 label 表达 | 无原生字段，靠 label 表达 |
| 状态 | `status_id`（1/2/3/5） | `state`（open/closed） | `state`（opened/closed） | `state`（open/closed/progressing/closed） |
| 鉴权 | Bearer / access_token | Bearer token | PRIVATE-TOKEN header | `?access_token=xxx` query |

### 各后端的写回示例

#### GitLink 写回
```bash
BODY=$(jq -n \
  --arg subject "..." --arg desc "..." \
  --argjson tag_ids '[323829]' \
  --argjson assigner_ids '[149027]' \
  --argjson priority_id 3 \
  '{subject:$subject,description:$desc,issue_tag_ids:$tag_ids,assigner_ids:$assigner_ids,priority_id:$priority_id}')
gitlink-cli api PATCH /v1/Owner/Repo/issues/N --body "$BODY"
```

#### GitHub 写回
```bash
gh issue edit N --repo Owner/Repo \
  --add-label "bug,priority: high" \
  --add-assignee chroe
# ⚠️ gh issue edit 不修改 title/body；要改用 gh issue edit --title/--body
```

#### GitLab 写回
```bash
glab issue update N --repo Owner/Repo \
  --label "bug" \
  --assignee chroe
# ⚠️ glab 不支持 label 数组，一次一个；要批量需循环
```

---

## 工作流（Agent 执行步骤）

### Step 0：确认环境 + 检测后端

```bash
# 0.1 检测后端
BACKEND=$(detect_backend)   # 输出 gitlink | github | gitlab
echo "✓ 后端: $BACKEND"

# 0.2 认证检查
case "$BACKEND" in
  gitlink) gitlink-cli auth status ;;
  github)  gh auth status ;;
  gitlab)  glab auth status ;;
esac
```

### Step 1：加载规则（从 SKILL.md 内嵌 JSON 块）

**本步骤不需要任何外部文件操作**。Agent 按以下逻辑加载规则：

```
1. Agent 从本 SKILL.md 的 `TRIAGE_RULES_JSON_START/END` 标记之间读取内嵌 JSON 代码块
2. 解析为内存数据结构：
   - mode（rule / hybrid / ai）
   - defaults
   - assigners
   - rules[]
3. 后续 Step 2-7 直接使用这份内存中的规则
```

**为什么不用仓库文件**：
- ✅ 零依赖：Skill 可丢到任意仓库即用，无需仓库内预先存在规则文件
- ✅ 版本一致：规则随 Skill 发布，避免 Skill 与规则不同步
- ✅ 可移植：同一个 Skill 在 100 个仓库用 100 次，规则完全一致
- ⚠️ 代价：修改规则需改 SKILL.md 本体（而非仓库文件）

**如需自定义规则**：编辑本 SKILL.md 的「机器可读规则块」小节，替换其中 JSON 代码块即可。无需改工作流 Step 1。

### Step 2：识别目标 Issue（按后端选命令）

```bash
case "$BACKEND" in
  gitlink)
    gitlink-cli issue +list --owner $OWNER --repo $REPO --state open --format json \
      | jq '[.data[] | select(.status_id==1 or .status_id==2) | select((.issue_tags//[]|length==0) or (.assigners//[]|length==0))]' ;;
  github)
    gh issue list --repo $OWNER/$REPO --state open --limit 200 \
      --json number,title,labels,assignees,state,body \
      | jq '[.[] | select((.labels|length==0) or (.assignees|length==0))]' ;;
  gitlab)
    glab issue list --repo $OWNER/$REPO --state opened --output json --all \
      | jq '[.[] | select((.labels|length==0) or (.assignees|length==0))]' ;;
esac
```

### Step 3：拉取标签 + 成员（建立 ID 映射）

按后端选对应命令（见上方"命令映射表"），构建：
```
{label_name → label_id}    # GitHub 标 ID 可选，gh 接受 name
{login → user_id}          # GitHub gh issue edit 接受 login；GitLab 同
```

### Step 4：评估每条 Issue（按 mode 处理）

三种 mode 算法与平台无关，按文件描述的 `mode: rule / hybrid / ai` 处理即可。

### Step 5：dry-run 预览（必须）

输出 Markdown 表格，**附带后端信息**：

```markdown
## 📋 Issue 分拣草稿（dry-run）

> 后端：<backend>（GitHub/GitLink/GitLab）
> 仓库：<owner>/<repo>
> 模式：<mode>
> 规则版本：<version>
> 涉及 Issue：<count> 条

| # | 标题 | 建议分类 | 建议标签 | 责任人 | 优先级 | 命中规则 |
|---|------|----------|----------|--------|--------|----------|
| 18 | ...  | bug | 缺陷, bug | chroe | 高 | bug-default |

⚠️ 涉及写入操作。回复"确认"或"apply"执行。
```

### Step 6：执行写回（用户确认后，按后端分发）

```bash
apply_to_issue() {
  local num="$1" rule_id="$2" labels="$3" assignee="$4" priority="$5"

  # 6.1 回读（必须！防 title/body 被清空）
  case "$BACKEND" in
    gitlink)
      ISSUE=$(gitlink-cli issue +view --owner $OWNER --repo $REPO --number $num --format json)
      TITLE=$(echo "$ISSUE" | jq -r .data.subject)
      BODY=$(echo "$ISSUE" | jq -r .data.description)
      ;;
    github)
      ISSUE=$(gh issue view $num --repo $OWNER/$REPO --json title,body,labels)
      TITLE=$(echo "$ISSUE" | jq -r .title)
      BODY=$(echo "$ISSUE" | jq -r .body)
      ;;
    gitlab)
      ISSUE=$(glab issue view $num --repo $OWNER/$REPO --output json)
      TITLE=$(echo "$ISSUE" | jq -r .title)
      BODY=$(echo "$ISSUE" | jq -r .description)
      ;;
  esac

  # 6.2 按后端写回
  case "$BACKEND" in
    gitlink)
      PATCH_BODY=$(jq -n \
        --arg s "$TITLE" --arg d "$BODY" \
        --argjson tags "$labels_id_array" \
        --argjson assigners "$assignee_id_array" \
        '{subject:$s, description:$d}
          + (if ($tags|length>0) then {issue_tag_ids:$tags} else {} end)
          + (if ($assigners|length>0) then {assigner_ids:$assigners} else {} end)')
      gitlink-cli api PATCH /v1/$OWNER/$REPO/issues/$num --body "$PATCH_BODY"
      ;;

    github)
      ARGS=()
      [[ -n "$labels" ]]    && ARGS+=(--add-label "$labels")
      [[ -n "$assignee" ]]  && ARGS+=(--add-assignee "$assignee")
      gh issue edit $num --repo $OWNER/$REPO "${ARGS[@]}"
      # ⚠️ GitHub 无原生 priority，靠 label 表达
      [[ -n "$priority" ]] && gh issue edit $num --repo $OWNER/$REPO --add-label "priority: $priority"
      ;;

    gitlab)
      [[ -n "$labels" ]]   && glab issue update $num --repo $OWNER/$REPO --label "$labels"
      [[ -n "$assignee" ]] && glab issue update $num --repo $OWNER/$REPO --assignee "$assignee"
      # ⚠️ GitLab 同 GitHub，靠 label 表达 priority
      [[ -n "$priority" ]] && glab issue update $num --repo $OWNER/$REPO --label "priority: $priority"
      ;;

    gitee)
      # ⚠️ Gitee 无官方 CLI，用 curl 直调 REST API
      # labels/assignees 用逗号分隔（不支持原生数组）；priority 靠 label 表达
      LABELS_CSV="$labels"
      [[ -n "$priority" && -z "$LABELS_CSV" ]] && LABELS_CSV="priority: $priority"
      [[ -n "$priority" && -n "$LABELS_CSV" ]] && LABELS_CSV="${LABELS_CSV},priority: $priority"
      ARGS=()
      [[ -n "$LABELS_CSV" ]] && ARGS+=(-d "labels=$LABELS_CSV")
      [[ -n "$assignee" ]]   && ARGS+=(-d "assignees=$assignee")
      if [[ ${#ARGS[@]} -gt 0 ]]; then
        curl -fsS -X PATCH \
          "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues/$num?access_token=$GITEE_TOKEN" \
          "${ARGS[@]}"
      fi
      ;;
  esac
}
```

### Step 7：验证

```bash
case "$BACKEND" in
  gitlink) gitlink-cli issue +view --owner $OWNER --repo $REPO --number $num --format json ;;
  github)  gh issue view $num --repo $OWNER/$REPO --json labels,assignees,state ;;
  gitlab)  glab issue view $num --repo $OWNER/$REPO --output json ;;
  gitee)   curl -fsSL "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues/$num?access_token=$GITEE_TOKEN" | jq . ;;
esac
```

---

## 规则文件 Schema

最小示例（用户放进 `<repo>/.triage/rules.yml`）：

```yaml
version: 1
mode: hybrid   # rule | hybrid | ai

defaults:
  dry_run: true
  skip_when:
    has_label_any: ["wontfix", "duplicate"]

rules:
  - id: bug-default
    type: bug
    label: ["缺陷", "bug"]
    priority: high           # 跨平台：human-readable，Agent 翻译成对应平台字段
    match:
      any_keyword: ["错误", "失败", "崩溃", "panic", "crash"]

  - id: question-default
    type: question
    label: ["疑问", "question"]
    priority: normal
    match:
      any_keyword: ["请问", "如何", "怎么", "how to"]

  - id: docs-typo
    type: docs
    label: ["文档", "good first issue"]
    priority: low
    match:
      any_keyword: ["typo", "文档", "README"]
```

### 字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `version` | ✅ | 固定 1 |
| `mode` | ✅ | `rule` / `hybrid` / `ai` |
| `defaults.dry_run` | ❌ | 默认 true（仅预览） |
| `defaults.skip_when.has_label_any` | ❌ | 已带这些标签则跳过 |
| `defaults.audit_log` | ❌ | 是否写 `.triage/logs/` |
| `rules[]` | ✅ | 规则列表（按出现顺序匹配，先列优先） |
| `rules[].id` | ✅ | 唯一 ID，用于审计 |
| `rules[].type` | ✅ | bug / enhancement / question / docs / security / performance / refactor / other |
| `rules[].label` | ❌ | 要打的标签名（自动匹配已有） |
| `rules[].priority` | ❌ | low / normal / high / critical（跨平台通用，Agent 翻译） |
| `rules[].assigner` | ❌ | 分配对象的登录名（必须是仓库成员） |
| `rules[].match.any_keyword` | * | OR 关键词 |
| `rules[].match.all_keyword` | * | AND 关键词 |
| `rules[].match.regex` | * | Python 正则（大小写不敏感） |
| `rules[].match.has_label` | * | 必须已带某标签 |
| `rules[].match.no_label` | * | 必须未带标签 |
| `rules[].match.min_description_length` | * | 描述最小字符数 |
| `rules[].exclude.any_keyword` | ❌ | 命中这些关键词则跳过本规则 |
| `rules[].add_comment` | ❌ | 给 Issue 加评论（支持 `${subject}` 占位符） |

### 跨平台字段翻译

Agent 内部按当前后端把 YAML 字段翻译成对应平台 API 字段：

```
priority: critical   →  GitLink: priority_id=4  /  GitHub: label "priority: critical"  /  GitLab: 同 GitHub  /  Gitee: 同 GitHub
priority: high       →  GitLink: priority_id=3  /  GitHub: label "priority: high"      /  GitLab: 同       /  Gitee: 同
priority: normal     →  GitLink: priority_id=2  /  GitHub: 不打 priority 标签           /  GitLab: 同       /  Gitee: 同
priority: low        →  GitLink: priority_id=1  /  GitHub: label "priority: low"       /  GitLab: 同       /  Gitee: 同
```

---

### 跨平台 Label 翻译（双语智能匹配）

每条规则 `label` 字段是双语列表（如 `["协助", "任务", "bug", "help wanted"]`），Agent 按当前后端**优先选择该平台的语言**：

| 后端 | 优先 Label 语言 | 备选 |
|------|------------------|------|
| `gitlink` | 🇨🇳 中文（协助 / 任务 / 支持 / 疑问 / 文档 / 重复 / 搁置） | 英文 |
| `github`  | 🇺🇸 英文（bug / help wanted / enhancement / question / documentation / duplicate / wontfix） | 中文 |
| `gitlab`  | 🇺🇸 英文（同 GitHub） | 中文 |
| `gitee`   | 🇨🇳 中文（同 GitLink） | 英文 |

**匹配逻辑**（Agent 在写回前执行）：

```
1. 调 label +list --format json 取仓库已有 label 映射 {name → id}
2. 对规则 label[] 列表按后端优先级顺序遍历：
   - BACKEND=gitlink/gitee：中文优先 [协助, 任务, bug, help wanted]
                              → 命中 "协助" 用它；否则尝试"任务"；否则尝试英文；以此类推
   - BACKEND=github/gitlab：英文优先 [bug, help wanted, 协助, 任务]
                              → 命中 "bug" 用它；否则尝试"help wanted"；否则尝试中文；以此类推
3. 首个在仓库已有的 label 被采用，写入 Issue
4. 都不存在 → 跳过打 label（不创建新 label，避免污染仓库）
```

**为什么这样设计**：

- ✅ 一份 YAML 兼容 4 平台，无需准备多份配置
- ✅ 中文平台（GitLink / Gitee）社区习惯中文标签
- ✅ 英文平台（GitHub / GitLab）社区习惯英文默认标签
- ✅ 不强行创建新标签，靠仓库已有标签生存

**举例**：规则 `bug-default` 的 label 是 `["协助", "bug"]`

- 在 GitHub 仓库 → Agent 优先匹配 `bug`（GitHub 默认标签） → 命中 → 打 `bug`
- 在 GitLink 仓库 → Agent 优先匹配 `协助`（GitLink 用户常用中文） → 命中 → 打 `协助`
- 仓库都没有 → 跳过（不创建新标签）

## 运行模式决策

```
mode: rule        ─→  100% 规则驱动，无 LLM
                      适用：CI、批量、对结果稳定性要求高
                      注意：无匹配时跳过该 Issue，不补判

mode: hybrid      ─→  规则优先，无命中时 Agent 调用 LLM 兜底
                      适用：日常运维，规则+灵活性兼顾 ★ 推荐默认

mode: ai          ─→  LLM 主导，规则仅作为 prompt 提示
                      适用：新仓库无足够历史标签时
                      注意：结果不可审计
```

---

## 安全与写回策略

| 规则 | 说明 |
|------|------|
| **必走 dry-run** | Step 5 输出预览，未确认前**禁止** 写回 |
| **必带字段** | GitLink 必带 subject+description；GitHub/GitLab 写回无需带（各自 CLI 自动保留） |
| **必走 `view` 回读** | 写入前回读当前状态 |
| **限流** | 批量 >100 条时，每次写回后 sleep 200ms |
| **失败回退** | 单条失败不阻塞；记录失败原因供用户决定 |
| **错误透明** | 任何 4xx/5xx 必须打印响应体，不要吞错 |

---

## 🙋 多人分配决策点

> 本轮分拣中有 3 条 type=测试 的 Issue，候选负责人为 [alice, bob, carol]。

请选择分配方式（回复序号）：
  1. 全部分配 → 该 type 的 Issue 同时分给三人（小团队协作）
  2. 轮流分配 → 按 Issue 创建时间轮流分配（alice → bob → carol → alice ...）
  3. 随机分配 → 每条 Issue 随机分给一人
  4. 跳过分配 → 仅打标签，不分配人
  5. 逐条指定 → 我逐条告诉你分给谁
  6. 记住选择 → 将你的选择写入 .triage/owners.yml，后续不再询问

你也可以直接说：
  - "全部" / "轮流" / "随机" / "跳过"
  - 或直接说"测试类都分给 bob"
```

#### 3. Agent 完整处理流程

```
Step 1: 加载 assigners / owners.yml
Step 2: 发现某 type 的值是列表 → 进入多人决策
Step 3: dry-run 预览阶段输出“多人决策点”提示
Step 4: 用户回复选择（1-6 或自然语言）
Step 5: Agent 按选择处理当前批次，并在 dry-run 表格中体现实际分配人
Step 6: 用户确认后执行写回
```

#### 4. 写入 `.triage/owners.yml` 后不再询问

用户选择「6.记住选择」后，Agent 会将选择写入 `.triage/owners.yml`：

```yaml
测试:
  pool: [alice, bob, carol]
  strategy: round_robin     # all | round_robin | random | skip | ask
```

后续运行直接按 strategy 执行，不再询问：
| strategy | 行为 |
|----------|------|
| `all` | 全部分配给池里所有人 |
| `round_robin` | 按 Issue 创建时间轮流 |
| `random` | 随机选一个 |
| `skip` | 不分配，仅打标签 |
| `ask` | 每次都询问（默认） |

#### 5. 混写示例（部分单人 + 部分多人）

```yaml
assigners:
  测试: [alice, bob, carol]   # 多人，每次询问（默认 strategy=ask）
  缺陷: bob                    # 单人，静默分给 bob
  功能: alice                  # 单人，静默分给 alice
  default: skip                # 未分类的不分配
```

类型分类标签 `assigners` 池中的单人形式会被静默处理，多人形式进入“决策点”流程。

---