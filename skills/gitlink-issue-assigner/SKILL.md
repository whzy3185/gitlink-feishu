---
name: gitlink-issue-assigner
version: 1.0.0
description: "Issue 智能分配（@通知版）：分析仓库历史贡献推荐负责人，在 issue 评论里 @ 推荐人+理由，绕过 GitLink 个人仓库 assigners 限制实现软分配。当 issue 分拣后需要派单、或传统 assign 失败需要替代方案时触发。任务二创新增强 Skill。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-issue-assigner（Issue 智能分配 · @通知版 · 创新增强 Skill）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。**
**CRITICAL — @ 通知（写评论）前务必确认用户意图。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务二**创新增强** Skill。**解决 GitLink 个人仓库 assigners 返回空、无法 assign 负责人的真实痛点**。思路：既然 `assigned_to_id` 走不通，就**在 issue 评论里 @ 推荐的负责人**——分析历史贡献（谁修过类似模块），推荐 Top 候选，评论 @ + 理由。被 @ 的人收到通知 = 软分配。绕过平台限制，切实让"对的人"看到 issue。

---

## 解决的痛点（真实存在）

```
gitlink-issue-triage 分拣完 issue → 打了标签
    ↓
想 assign 负责人 → issue +assigners 返回空（个人仓库）
    ↓
PATCH assigned_to_id → 即使填 owner 也无效（平台校验）
    ↓ ❌ 断点：分拣完没人管
```

**本 Skill 的创新解法**：用 @ 评论通知代替 assign，绕过限制。

---

## 工作流

### Step 1：分析 issue 涉及的模块
```bash
gitlink-cli issue +view --owner <o> --repo <r> --number <n> --format json
# 从 subject/description/tags 推断 issue 涉及的模块（如 wiki/label/notification）
```

### Step 2：推荐负责人（分析历史贡献）
```bash
# 谁修过类似模块（从 commit 历史匹配）
git log --format="%an|%s" -- <相关模块路径>
# 如 issue 涉及 wiki → git log -- shortcuts/wiki/ → 找谁贡献过 wiki

# 谁活跃（近期贡献多）
git log --since="3 months ago" --format="%an" | sort | uniq -c | sort -rn
```
AI 综合推荐 Top 1-3 候选（修过相关模块 + 近期活跃）。

### Step 3：@ 通知（软分配）⚠️写入
```bash
# 在 issue 评论里 @ 推荐人 + 推荐理由
MSYS_NO_PATHCONV=1 gitlink-cli api POST /v1/<owner>/<repo>/issues/<n>/journals \
  --body-file comment.json --format json
# comment.json: {"notes": "@ylly 这个 issue 涉及 wiki 模块，你之前贡献过 shortcuts/wiki，方便看一下吗？"}
```

### Step 4：输出分配报告
```markdown
## 📮 Issue 智能分配（@通知）— #<n>

### Issue 分析
涉及模块：<wiki/label/...>

### 推荐负责人
1. @ylly（贡献过 shortcuts/wiki，近期活跃）⭐ Top
2. @ZxR（贡献过相关）

### 已通知
✅ 已在 #<n> 评论 @ylly + 推荐理由
（注：GitLink 个人仓库 assigners 受限，改用 @ 通知软分配）
```

---

## 关键避坑

| 坑 | 解决 |
|----|------|
| assigners 个人仓库返回空 | 本 Skill 核心：用 @ 评论替代 assign |
| PATCH assigned_to_id 无效 | 不依赖 assign，用 @ 通知 |
| @ 用户名需是仓库成员 | 推荐仓库历史贡献者（必然是成员）|
| journals endpoint 必须 /v1/ 前缀 | `api POST /v1/<o>/<r>/issues/<n>/journals` |
| Windows JSON 中文乱码 | 用 `--body-file <UTF-8文件>` + `MSYS_NO_PATHCONV=1` |

---

## 实测落地参考

**场景**：gitlink-cli 某 issue 涉及 wiki 模块
- Step1 分析：issue 标题/描述涉及 wiki
- Step2 推荐：`git log -- shortcuts/wiki/` → ylly 是 wiki 模块主要贡献者
- Step3 @通知：评论 `@ylly 这个 issue 涉及 wiki，你贡献过 shortcuts/wiki，方便看下吗？`
- **效果**：ylly 收到通知，issue 不再"没人管"

详见 verification.md。
