---
name: gitlink-digest
version: 1.0.0
description: "每日简报：聚合仓库 Issue/PR/CI/通知动态，生成一份可读的项目简报。当用户提到「每日简报」「今天发生了什么」「项目动态」「日报」「digest」「简报」「汇总」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli --help"
---

# gitlink-digest（每日简报）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 本 Skill 只读聚合，不产生任何写操作，安全可随时运行。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能定位

解决「信息太分散」的体验痛点：用户不用挨个刷 Issue/PR/CI/通知，AI 一次性采集多源数据，聚合分类，输出**一份按优先级排序的 Markdown 简报**。

| 阶段 | 操作 | AI Agent 角色 |
|------|------|--------------|
| ① 采集 | 并行拉取 Issue/PR/CI/通知/活跃度 | 执行 CLI 命令采集多源数据 |
| ② 分类 | 按主题归类（新增/动态/CI/需关注） | 去重、排序、标注优先级 |
| ③ 聚合 | 合并成一份简报 | 生成结构化 Markdown |
| ④ 输出 | 可选发布到 Wiki/Issue | 生成或发布简报 |

### 与 `gitlink-notification-digest` 的分工（避免重复）

项目已有 `gitlink-notification-digest`，专注**通知/消息中心**（messages API、按 source 分类、标记已读）。本 Skill 定位不同——做**项目全景日报**：

| 维度 | notification-digest（团队已有） | digest（本 Skill） |
|------|-------------------------------|-------------------|
| 范围 | 通知/消息（messages） | 项目全景：Issue + PR + CI + 活跃度 + 通知 |
| 重点 | 通知分类、标记已读、清理未读 | 跨源聚合、按优先级出日报 |
| 输出 | 通知摘要（P0–P3） | 项目简报（需关注/新增/进行中/指标） |

> 本 Skill **不做**通知标记已读（那是 notification-digest 的职责），只把通知作为简报的一个输入源。

## 数据源（均为只读）

| 数据 | 命令 | 说明 |
|------|------|------|
| 近期 Issue | `gitlink-cli issue +list --state open --format json` | 新增/开放 Issue |
| 近期 PR | `gitlink-cli pr +list --state open --format json` | PR 动态 |
| CI 构建 | `gitlink-cli ci +builds --owner <owner> --repo <repo> --format json` | 构建成功/失败 |
| 通知消息 | `gitlink-cli api GET "users/{owner}/messages.json"` | @我/系统通知 |
| 活跃度 | `gitlink-cli api GET "users/{owner}/statistics/activity.json"` | 贡献活跃度 |

> 注：通知/活跃度为 Raw API，路径以 API 文档为准；首次使用建议带 `--debug` 确认 CLI 的路径前缀行为。

## 使用示例

```bash
# 1. 采集近期 Issue（默认从 git remote 解析 owner/repo）
gitlink-cli issue +list --state open --format json

# 2. 采集近期 PR
gitlink-cli pr +list --state open --format json

# 3. 采集 CI 构建状态
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json

# 4. 采集通知消息（Raw API）
gitlink-cli api GET /api/users/<your-username>/messages.json

# 5. 采集活跃度统计（Raw API）
gitlink-cli api GET /api/users/<your-username>/statistics/activity.json
```

## 工作流

### 工作流 1：生成每日简报（核心）

**场景**：用户问「今天我的项目发生了什么 / 给我一份简报」。

#### Step 1：并行采集多源数据

```bash
gitlink-cli issue +list --state open --format json
gitlink-cli pr +list --state open --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli api GET /api/users/<me>/messages.json
gitlink-cli api GET /api/users/<me>/statistics/activity.json
```

#### Step 2：AI 分类聚合

对采集到的数据按以下维度归类：
- **🔴 需立即关注**：失败的 CI、@我的紧急消息、阻塞型 PR
- **🟢 新增动态**：新开的 Issue、新提交的 PR
- **🔵 进行中**：有更新的 Issue/PR、待 review 的 PR
- **📊 健康指标**：活跃度数字、Issue/PR 增减趋势

#### Step 3：输出简报

按下方「输出模板」生成 Markdown。

### 工作流 2：发布简报

**场景**：把简报发布为 Wiki 或 Issue 评论（写操作，需确认）。

```bash
# 发布为 Wiki 页面（⚠️ 写操作，需确认）
gitlink-cli wiki +create --name "Daily-<date>" --content "<简报内容>"

# 或发布为 Issue 评论
gitlink-cli issue +comment --number <n> --body "<简报内容>"
```

## 决策规则

| 条件 | 处理 |
|------|------|
| 数据量大（Issue/PR > 50） | 只取最近 24h 或 top 20，其余汇总计数 |
| CI 有失败 | 置顶到「需立即关注」，附 build 号 |
| 有 @我 的消息 | 置顶，标注来源 Issue/PR |
| 采集某数据源失败（403/404） | 跳过该源，简报中标注「⚠️ XX 数据未获取」 |
| 简报需对外发布 | 写操作，必须先确认用户意图 |

## 输出模板

```markdown
# 📰 项目简报 — <owner>/<repo>（<YYYY-MM-DD>）

## 🔴 需立即关注
1. ❌ CI 构建 #<n> 失败（分支 master）— <错误摘要>
2. 🔔 @你 在 Issue #<n>：<消息摘要>

## 🟢 今日新增
- **新 Issue**：<n> 个，其中 bug <x> / enhancement <y>
- **新 PR**：<n> 个

## 🔵 进行中
- 待 Review 的 PR：#<n>、#<n>
- 有更新的 Issue：#<n>、#<n>

## 📊 健康指标
- 开放 Issue：<n>（较昨日 +<x>）
- 开放 PR：<n>
- 近期活跃度：<activity 分数>

---
*由 gitlink-digest Skill 于 <时间> 生成*
```

## 注意事项

- 本 Skill **纯只读聚合**，不修改任何资源，可放心运行
- 多源数据采集建议用 `--format json` 便于 AI 解析
- 通知/活跃度是 Raw API，路径前缀以实际 CLI 行为准（首次 `--debug` 验证）
- 时间范围默认「近期」，可由用户指定（如「本周」「最近 3 天」）
- 简报对外发布（Wiki/Issue 评论）属写操作，必须先确认

## References

- 全局参数与安全规则：[gitlink-shared/SKILL.md](../gitlink-shared/SKILL.md)
