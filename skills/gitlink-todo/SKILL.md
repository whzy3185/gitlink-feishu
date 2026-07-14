---
name: gitlink-todo
version: 1.0.0
description: "我的待办：跨 Issue/PR 汇总「分配给我的 / @我的 / 我的 PR 被 review 的」，生成个人待办清单。当用户提到「我的待办」「我有什么要做的」「@我」「待办清单」「todo」「我的任务」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli --help"
---

# gitlink-todo（我的待办）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 本 Skill 默认只读汇总；自动回复/关闭等写操作必须先确认用户意图。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能定位

解决「没有『我的』视角」的体验痛点：GitLink 的 Issue/PR 分散在各处，用户难以一眼看到「**哪些事在等我**」。本 Skill 聚合个人维度的待办，按紧急度排序输出清单。

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| ① 识别身份 | 获取当前登录用户 | `api GET /users/me` |
| ② 多维查询 | 分配我的 / @我的 / 我的PR | 并行采集三类待办 |
| ③ 排序 | 按紧急度（@我 > PR待review > 指派issue） | 标注优先级与停留时长 |
| ④ 输出 | 个人待办清单 | 生成 Markdown |

## 数据源（均为只读）

| 待办类型 | 命令 | 说明 |
|---------|------|------|
| 分配给我的 Issue | `gitlink-cli search +issues --assignee <me> --category opened` | 我的负责项 |
| @我的消息 | `gitlink-cli api GET "users/<me>/messages.json"` | 被提及/被通知 |
| 我的 PR 状态 | `gitlink-cli pr +list --format json` | 我提的 PR 是否被 review/合并 |
| 待我 Review 的 PR | `gitlink-cli pr +list --state open --format json` | 分配我 review 的 |

> `search +issues --assignee`（简写 `-a`）已确认可用（`shortcuts/search/search.go`）。

## 使用示例

```bash
# 1. 获取当前用户身份
gitlink-cli api GET "users/me" --format json

# 2. 找分配给我的开放 Issue
gitlink-cli search +issues --assignee <me> --category opened

# 3. 找 @我 的消息
gitlink-cli api GET "users/<me>/messages.json"

# 4. 查看我的 PR 状态（按作者过滤）
gitlink-cli pr +list --format json
```

## 工作流

### 工作流 1：生成我的待办（核心）

**场景**：用户问「我今天有什么要做的 / 给我看看我的待办」。

#### Step 1：识别身份

```bash
gitlink-cli api GET "users/me" --format json
# 从返回取 login / user_name 作为 <me>
```

#### Step 2：多维并行查询

```bash
# 分配给我的 Issue
gitlink-cli search +issues --assignee <me> --category opened

# @我的消息
gitlink-cli api GET "users/<me>/messages.json"

# 我的 PR 状态
gitlink-cli pr +list --format json
```

#### Step 3：AI 排序聚合

按紧急度分级：
- **🔴 紧急**：@我且停留 >24h 的消息、被阻塞的 PR
- **🟡 本周内**：分配我的 Issue、待我 review 的 PR
- **🔵 可延后**：低优先级 Issue、我的 PR 已 review 待合并

#### Step 4：输出待办清单

按下方「输出模板」生成。

### 工作流 2：批量处理待办（写操作）

**场景**：用户看完待办后，想批量更新状态。

```bash
# 批量关闭已解决的 Issue（⚠️ 先 dry-run）
gitlink-cli issue +batch-close --numbers 12,15 --dry-run

# 给待办 Issue 加优先级标签
gitlink-cli issue +batch-update --ids 20,21 --tag-ids 3 --dry-run

# 回复 @我的 消息
gitlink-cli issue +comment --number <n> --body "<回复内容>"
```

## 决策规则

| 条件 | 处理 |
|------|------|
| 用户身份未知 | 先 `api GET /users/me`，失败则提示登录 |
| @我 的消息停留 >24h | 标为 🔴 紧急置顶 |
| 我的 PR 长期未 review | 提醒「PR #<n> 已 <x> 天未 review，可 @维护者」 |
| 待办过多（>20） | 按优先级截断 top 20，其余计数 |
| 批量写操作 | 必须 `--dry-run` 预览后确认 |

## 输出模板

```markdown
# ✅ 我的待办 — <me>（<YYYY-MM-DD>）

## 🔴 紧急（<n>）
1. 🔔 Issue #<n> @你：<消息>（停留 2 天）
2. 🔀 PR #<n> 你被指派 review，已等待 3 天

## 🟡 本周内（<n>）
- 分配给你的 Issue：
  - #<n> <标题>（优先级 high）
  - #<n> <标题>
- 待 Review 的 PR：#<n>、#<n>

## 🔵 你的 PR（<n>）
- #<n> <标题> — ✅ 已 review，待合并
- #<n> <标题> — ⏳ 等待 review（2 天）

---
*由 gitlink-todo Skill 生成，共 <n> 项待办*
```

## 注意事项

- 本 Skill 核心是**只读汇总**，不修改资源
- `search +issues --assignee` 需传用户标识（login 或 id），先用 `/users/me` 获取
- 批量处理（关闭/打标签/回复）属写操作，必须先确认或 `--dry-run`
- 待办优先级由 AI 根据停留时长、@提及、阻塞情况综合判定

## References

- 全局参数与安全规则：[gitlink-shared/SKILL.md](../gitlink-shared/SKILL.md)
