---
name: gitlink-release-notes
version: 1.0.0
description: "Release Notes 自动生成与发布：在 GitHub / GitLink / GitLab / Gitee 等开源项目中，自动汇总已关闭 Issue，按 Bug 修复 / 新功能 / 其他改进分类，生成结构化 Release Notes 并发布为正式版本。触发场景：发布 Release Notes、生成版本说明、版本发布、changelog、release notes、自动发布版本。"
license: MulanPSL-2.0
metadata:
  requires:
    bins_any: ["gitlink-cli", "gh", "glab", "curl"]
    bins_note: "gitee 后端无官方 CLI，直接使用 curl 调用 https://gitee.com/api/v5/ REST API；其余三后端用对应原生 CLI"
  cliHelp: "gitlink-cli release --help"
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

# gitlink-release-notes（Release Notes 自动生成与发布）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/skills/gitlink-shared/SKILL.md)（仅 GitLink 后端）或对应平台的 CLI 文档。所有 GitHub 操作必须使用 `gh`；所有 GitLab 操作必须使用 `glab`；所有 GitLink 操作必须使用 `gitlink-cli`。禁止混用或替代。**
**CRITICAL — 本 Skill 包含写入操作（创建 Release）。默认必须先做 dry-run（仅预览），经用户明确确认后才执行发布。**
**CRITICAL — 创建 Release 前必须检查已有 Release（tag 去重），避免重复发布。**

> **依赖工具：** 二选一即可——`gitlink-cli` / `gh` / `glab`，外加 `jq`。
> **依赖 Skill：** 使用 GitLink 后端时需加载 `gitlink-shared`；使用 GitHub / GitLab 时无需额外 Skill。
> **本 Skill 为 C1 模式（纯文档）**：所有命令由 Agent 按本文件步骤直接调用对应平台 CLI 执行。

---

## 功能概述

本 Skill 是**自包含**的：Release Notes 分类逻辑与模板**直接内嵌在 SKILL.md 中**，Agent 从本文件读取规范，不需要从外部仓库拉取配置。

1. **后端检测**：Agent 自动识别当前仓库所在的平台
2. **数据采集**：Agent 用对应 CLI 收集已关闭 Issue、开放 Issue、已有 Release
3. **分类汇总**：按 Bug 修复 / 新功能 / 其他改进三组分类
4. **版本命名**：自动生成版本号（日期格式），去重已有 tag
5. **内容生成**：Agent 按内嵌模板生成结构化 Markdown Release Notes
6. **dry-run 预览**：Agent 输出 Release Notes，等待用户确认
7. **发布**：用户确认后，Agent 用对应 CLI 创建 Release

> **⚠️ 设计约束**：版本号格式为 `vYYYY.MM.DD`（日期格式）。如当天已有同名 tag，追加序号 `-N`。

## 触发场景

用户提到以下关键词时自动触发：
- "发布 Release Notes"、"生成版本说明"、"版本发布"
- "release notes"、"changelog"、"自动发布版本"
- "帮我创建一个 release"

---

## 平台适配

### 后端自动检测

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
| **列关闭 Issue** | 取已关闭 Issue | `gitlink-cli issue +list --owner X --repo Y --state closed --format json` | `gh issue list --repo X/Y --state closed --json number,title,labels` | `glab issue list --repo X/Y --state closed --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues?state=closed&access_token=$GITEE_TOKEN"` |
| **列开放 Issue** | 取开放 Issue（待处理） | `gitlink-cli issue +list --owner X --repo Y --state open --format json` | `gh issue list --repo X/Y --state open --json number,title,labels` | `glab issue list --repo X/Y --state opened --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/issues?state=open&access_token=$GITEE_TOKEN"` |
| **列 Release** | 已有版本去重 | `gitlink-cli release +list --owner X --repo Y --format json` | `gh release list --repo X/Y --json tagName` | `glab release list --repo X/Y --output json` | `curl -s "https://gitee.com/api/v5/repos/X/Y/releases?access_token=$GITEE_TOKEN"` |
| **创建 Release** | 发布版本 | `gitlink-cli release +create --owner X --repo Y --name V --tag V --target master --body "..." --format json` | `gh release create V --repo X/Y --title V --notes "..." --target master` | `glab release create V --repo X/Y --name V --notes "..." --ref master` | `curl -X POST "https://gitee.com/api/v5/repos/X/Y/releases?access_token=$GITEE_TOKEN&tag_name=V&name=V&body=..."` |

### 字段名差异

| 概念 | GitLink | GitHub | GitLab | Gitee |
|------|---------|--------|--------|-------|
| Issue 标题 | `subject` | `title` | `title` | `title` |
| Issue 标签 | `issue_tags` (数组 of {name,id}) | `labels` (数组 of {name}) | `labels` (数组 of {name}) | `labels` (逗号分隔) |
| Release tag | `tag_name` | `tagName` | `tag_name` | `tag_name` |
| Release name | `name` | `name` | `name` | `name` |

---

## Issue 分类逻辑（内嵌）

Agent 对已关闭 Issue 按标签进行三组分类：

| 分类组 | 匹配标签（中文平台：GitLink / Gitee） | 匹配标签（英文平台：GitHub / GitLab） | 优先级 |
|--------|--------------------------------------|--------------------------------------|--------|
| **Bug 修复** | `缺陷` | `bug` | 高 |
| **新功能** | `功能` | `enhancement`, `feature` | 中 |
| **其他改进** | 不属于以上两组的所有 Issue | 不属于以上两组的所有 Issue | 低 |

**分类规则**：
1. 遍历每条 Issue 的标签列表
2. 首个命中分类组的标签决定该 Issue 所属分类
3. 一条 Issue 只归属一个分类组（不重复）
4. 无标签的 Issue 归入"其他改进"

---

## 版本命名规则

```
1. 默认格式: vYYYY.MM.DD (如 v2026.07.10)
2. 去重: 查已有 Release 的 tag_name 列表
   - 当天已有同名 tag →追加序号: v2026.07.10-2, v2026.07.10-3 ...
3. 用户指定版本号时: 直接使用用户指定的版本号 (如 v1.2.0)
```

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
#   OWNER, REPO, VERSION (可选, 默认日期格式)
echo "仓库: $OWNER/$REPO | 版本: ${VERSION:-auto(vYYYY.MM.DD)}"
```

### Step 1：采集数据（按后端选命令）

```bash
case "$BACKEND" in
  gitlink)
    CLOSED=$(gitlink-cli issue +list --owner $OWNER --repo $REPO --state closed --limit 200 --format json)
    OPEN=$(gitlink-cli issue +list --owner $OWNER --repo $REPO --state open --limit 200 --format json)
    RELEASES=$(gitlink-cli release +list --owner $OWNER --repo $REPO --format json)
    ;;
  github)
    CLOSED=$(gh issue list --repo $OWNER/$REPO --state closed --limit 200 --json number,title,labels)
    OPEN=$(gh issue list --repo $OWNER/$REPO --state open --limit 200 --json number,title,labels)
    RELEASES=$(gh release list --repo $OWNER/$REPO --json tagName --limit 50)
    ;;
  gitlab)
    CLOSED=$(glab issue list --repo $OWNER/$REPO --state closed --output json --all)
    OPEN=$(glab issue list --repo $OWNER/$REPO --state opened --output json --all)
    RELEASES=$(glab release list --repo $OWNER/$REPO --output json)
    ;;
  gitee)
    CLOSED=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?state=closed&access_token=$GITEE_TOKEN&per_page=200")
    OPEN=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/issues?state=open&access_token=$GITEE_TOKEN&per_page=200")
    RELEASES=$(curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/releases?access_token=$GITEE_TOKEN")
    ;;
esac
```

### Step 2：版本号去重 + 分类

```bash
# 2.1 版本号去重
# 从 RELEASES 提取已有 tag_name 列表
# 检查 VERSION 是否冲突，冲突时追加序号

# 2.2 Issue 分类 (按内嵌分类逻辑)
# Bug 修复: 标签含 "缺陷" (GitLink/Gitee) 或 "bug" (GitHub/GitLab)
# 新功能:  标签含 "功能" (GitLink/Gitee) 或 "enhancement"/"feature" (GitHub/GitLab)
# 其他改进: 不属于以上两组的 Issue

# 2.3 统计
BUG_COUNT    ← Bug 修复组 Issue 数量
FEATURE_COUNT ← 新功能组 Issue 数量
OTHER_COUNT  ← 其他改进组 Issue 数量
OPEN_COUNT   ← 开放 Issue 总数
```

### Step 3：生成 Release Notes（按内嵌模板）

Agent 按以下模板生成 Markdown Release Notes。**所有数值占位符由 Step 2 计算后填入**：

```markdown
# {VERSION} Release Notes

**发布日期:** {NOW_DATE}
**仓库:** {OWNER}/{REPO}

## 本次更新摘要

- 修复缺陷: **{BUG_COUNT}** 项
- 新增功能: **{FEATURE_COUNT}** 项
- 其他改进: **{OTHER_COUNT}** 项
- 仍开放 Issue: **{OPEN_COUNT}** 项

## Bug 修复

{BUG_LIST}
<!-- 格式: - #{number} {subject/title} -->

## 新功能

{FEATURE_LIST}
<!-- 格式: - #{number} {subject/title} -->

## 其他改进

{OTHER_LIST}
<!-- 格式: - #{number} {subject/title} -->

## 仍待处理

{OPEN_LIST}
<!-- 格式: - #{number} {subject/title} (最多10条) -->
<!-- 超过10条: "...及其他 {remaining} 条" -->

---

_本 Release Notes 由 gitlink-release-notes Skill 自动生成_
```

### Step 4：dry-run 预览（必须）

Agent 输出完整 Release Notes Markdown，附带元信息：

```markdown
## 📦 Release Notes 预览（dry-run）

> 后端：{BACKEND}
> 仓库：{OWNER}/{REPO}
> 版本号：{VERSION}
> Bug 修复：{count} 项 | 新功能：{count} 项 | 其他改进：{count} 项

{完整 Release Notes Markdown 内容}

⚠️ 涉及写入操作（创建 Release）。回复"确认"或"apply"执行。
```

### Step 5：创建 Release（用户确认后，按后端分发）

```bash
case "$BACKEND" in
  gitlink)
    gitlink-cli release +create --owner $OWNER --repo $REPO \
      --name $VERSION --tag $VERSION --target master \
      --body "$NOTES_MD" --format json
    ;;
  github)
    gh release create $VERSION --repo $OWNER/$REPO \
      --title $VERSION --notes "$NOTES_MD" --target master
    ;;
  gitlab)
    glab release create $VERSION --repo $OWNER/$REPO \
      --name $VERSION --notes "$NOTES_MD" --ref master
    ;;
  gitee)
    curl -X POST "https://gitee.com/api/v5/repos/$OWNER/$REPO/releases?access_token=$GITEE_TOKEN" \
      -d "tag_name=$VERSION" -d "name=$VERSION" -d "body=$NOTES_MD" \
      -d "target_commitish=master"
    ;;
esac
```

### Step 6：验证

```bash
# 确认 Release 已创建
case "$BACKEND" in
  gitlink) gitlink-cli release +list --owner $OWNER --repo $REPO --format json \
    | jq '.data.releases[] | select(.tag_name=="$VERSION")' ;;
  github)  gh release view $VERSION --repo $OWNER/$REPO ;;
  gitlab)  glab release view $VERSION --repo $OWNER/$REPO ;;
  gitee)   curl -s "https://gitee.com/api/v5/repos/$OWNER/$REPO/releases?access_token=$GITEE_TOKEN" \
    | jq '.[] | select(.tag_name=="$VERSION")' ;;
esac
```

---

## Release Notes 模板（内嵌）

以下是完整的 Release Notes Markdown 模板：

```markdown
# {VERSION} Release Notes

**发布日期:** {NOW_DATE}
**仓库:** {OWNER}/{REPO}

## 本次更新摘要

- 修复缺陷: **{BUG_COUNT}** 项
- 新增功能: **{FEATURE_COUNT}** 项
- 其他改进: **{OTHER_COUNT}** 项
- 仍开放 Issue: **{OPEN_COUNT}** 项

## Bug 修复

{BUG_LIST}

## 新功能

{FEATURE_LIST}

## 其他改进

{OTHER_LIST}

## 仍待处理

{OPEN_LIST}

---

_本 Release Notes 由 gitlink-release-notes Skill 自动生成_
```

---

## 安全与写回策略

| 规则 | 说明 |
|------|------|
| **必走 dry-run** | Step 4 输出预览，未确认前**禁止** 发布 |
| **必须去重** | Step 2.1 检查已有 Release tag，避免重复 |
| **限流** | 数据采集 >100 条时，每次 CLI 调用后 sleep 200ms |
| **错误透明** | 任何 4xx/5xx 必须打印响应体，不要吞错 |
| **空数据容错** | 某分类无数据时输出"无"，不跳过章节 |
| **仍待处理截断** | 开放 Issue >10 条时截断，输出"…及其他 N 条" |

---

## 使用示例

### 示例 1：GitLink 项目

**用户输入**：

```
帮我发布 Angel123456/gitlink-cli 的 Release Notes
```

**Agent 执行**：

```bash
# Step 0
gitlink-cli auth status

# Step 1
gitlink-cli issue +list --owner Angel123456 --repo gitlink-cli --state closed --limit 200 --format json
gitlink-cli issue +list --owner Angel123456 --repo gitlink-cli --state open --limit 200 --format json
gitlink-cli release +list --owner Angel123456 --repo gitlink-cli --format json

# Step 2: 版本号 v2026.07.10 + 分类 (Bug 修复/新功能/其他)

# Step 3-4: 生成 Release Notes → dry-run 预览

# Step 5 (用户确认后):
gitlink-cli release +create --owner Angel123456 --repo gitlink-cli \
  --name v2026.07.10 --tag v2026.07.10 --target master \
  --body "$NOTES_MD" --format json

# Step 6: 验证
gitlink-cli release +list --owner Angel123456 --repo gitlink-cli --format json \
  | jq '.data.releases[] | select(.tag_name=="v2026.07.10")'
```

### 示例 2：GitHub 项目（指定版本号）

**用户输入**：

```
发布 xuanlanwuta/gps_SM v1.2.0 的 Release Notes
```

**Agent 执行**：

```bash
# Step 0
gh auth status

# Step 1
gh issue list --repo xuanlanwuta/gps_SM --state closed --limit 200 --json number,title,labels
gh issue list --repo xuanlanwuta/gps_SM --state open --limit 200 --json number,title,labels
gh release list --repo xuanlanwuta/gps_SM --json tagName

# Step 2: VERSION=v1.2.0 (用户指定) + 去重检查 + 分类

# Step 5 (用户确认后):
gh release create v1.2.0 --repo xuanlanwuta/gps_SM \
  --title v1.2.0 --notes "$NOTES_MD" --target master
```

---

## Agent 平台兼容性

| 平台 | 加载方式 | 触发方式 |
|------|----------|----------|
| **OpenClaw** | `workspace/skills/gitlink-release-notes/SKILL.md` | 自然语言 |
| **Claude Code** | `~/.claude/skills/gitlink-release-notes/SKILL.md` | 自然语言 |
| **Cursor** | `~/.cursor/skills/gitlink-release-notes/SKILL.md` | 自然语言 + `/` 命令 |
| **WorkBuddy** | `~/.workbuddy/skills/gitlink-release-notes/SKILL.md` | 自然语言 |
| **通用 Agent** | 作为参考文档 | 按 SKILL.md 流程自取 |

---

## 与 gitlink-community-ops 的关系

本 Skill 可**独立使用**，也可作为 `gitlink-community-ops` 统一入口 Skill 的子 Skill 被编排调用：

- 独立触发：用户说"发布 Release Notes"
- 编排触发：`gitlink-community-ops` Step 3 指令"加载并执行 gitlink-release-notes"

当作为子 Skill 被调用时，Agent 需在上下文中继承 `OWNER`、`REPO`、`BACKEND`、`VERSION` 参数。
