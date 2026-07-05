---
name: gitlink-notify-digest
version: 1.0.0
description: "通知智能摘要：采集用户通知，AI 按重要性分级（@我/分配/review 为高优）+ 分类汇总（issue/pr/member/release），输出每日通知摘要。当用户通知过多、重要信息被淹没时触发。任务二创新增强 Skill。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli notification --help"
---

# gitlink-notify-digest（通知智能摘要 · 创新增强 Skill）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读采集 + 分析，不写入。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务二**创新增强** Skill。活跃用户通知刷屏、重要信息（被 @、被分配、review 请求）被淹没——本 Skill 采集通知，AI **按重要性分级 + 分类汇总**，输出"每日通知摘要"，让用户快速抓重点。复用任务一 notification 命令。

---

## 解决的痛点
- 通知太多看不过来 → 重要通知（@我/分配/review）被淹没
- 无优先级 → 每条都点开看，效率低

## 分级模型

| 优先级 | 通知类型 | 处理 |
|:------:|---------|------|
| 🔴 高 | 被 @、被分配 Issue/PR、Review 请求、PR 合并/驳回 | 必看，摘要置顶 |
| 🟡 中 | 新 Issue、PR 提交、成员加入 | 关注，分类汇总 |
| 🟢 低 | 系统/流水线/一般动态 | 摘要计数即可 |

## 工作流

### Step 1：采集通知
```bash
# ⚠️ --owner 填自己的 login
gitlink-cli notification +list --owner <self_login> --format json
# 提取每条：source / content / status(已读?) / time_ago
```

### Step 2：AI 分级 + 分类
AI 按通知 source/content 分级（高/中/低）+ 分类（issue/pr/member/release/系统）。

### Step 3：输出每日摘要
```markdown
## 📬 每日通知摘要 — <login>

📅 共 N 条（X 未读）

### 🔴 重要（必看）
- 被 @ 在 #<n>：<内容>
- PR #<n> 待你 review
- Issue #<n> 分配给你

### 🟡 关注
- 新 Issue：N 条（#a #b #c）
- 新 PR：N 条
- 新成员加入：N 人

### 🟢 动态
- 流水线/系统通知：N 条

### 建议
优先处理：<最紧急的>
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| notification --owner 查别人 403 | 必须填自己的 login |
| 通知 content 含 HTML 标签 | AI 提取纯文本关键词 |
| 分级规则需结合 source + content | source=ProjectIssue+content含@→高优 |
| 大量通知分页 | --page/--limit 分页采集 |

---

## 实测落地参考
**ylly 的通知**（之前采集：5 条）：
- 🔴 重要：IssueAssigned（被分配）、ProjectIssue（@相关）
- 🟡 关注：ProjectMemberJoined（新成员）
- 🟢 动态：ProjectOpenDevOps（流水线）

**摘要输出**：1 条高优（被分配 issue）+ 1 条关注（新成员）+ 流水线动态，让 ylly 一眼知道"最重要的 issue 被分配了"。

详见 verification.md。
