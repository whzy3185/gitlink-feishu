# Agent 对话触发语（PDF 核心：Agent 对话是首要产物）

> 用途：评委/复现者在 Claude Code 里**逐字粘贴**下面任一条触发语，即可观察到
> AI Agent 自动加载对应 Skill（按 SKILL.md 蓝图）→ 调 gitlink-cli 命令 → 串联多步 → 落地写入。
> 把这个对话过程截图/录屏，就是 PDF 要求的「Agent 对话记录」产物。

---

## 为什么需要触发语

PDF「正确做法」明确：

- **Agent 对话记录是首要产物**，脚本是辅助
- **Skill 是设计蓝图**，不是配置文件
- 必须能演示"Skill 真的被加载、真的驱动 Agent 执行"

触发语的设计原则：**用自然语言匹配 SKILL.md 第 4 行 `description` 字段的触发语义**，让 Agent 自动识别并调用 Skill 工具。

---

## 前置准备（所有触发语共用）

在 Claude Code 里先说一句建立上下文：

```
我在做 GitLink 端到端自动化任务三。目标仓库是 ylly/gitlink-cli，
我的登录账号是 zhangqing23（已是该仓库 Manager）。
gitlink-cli.exe 在 C:/Users/Lenovo/Desktop/gitlink-cli/ 目录下，已 auth login。
下面我会给你具体的自动化需求，请用任务二设计好的 Skill 来做。
```

---

## 工作流 ① 触发语（社区运营自动化）

**单条完整触发语**（推荐用于演示，一句话串起分类→周报→发版）：

```
请帮我对 ylly/gitlink-cli 做一次社区运营收尾：
1. 自动分拣所有未分类的开放 Issue（用 gitlink-issue-triage 工作流1）；
2. 生成一份项目周报（用 gitlink-insight 工作流2）；
3. 根据自上次发版以来的提交历史，推荐版本号并生成 Release Notes，
   先给我预览，确认后再发布（用 gitlink-release-auto）。
每一步执行前告诉我你正在加载哪个 Skill、对应 SKILL.md 的哪一段。
```

**分步触发语**（如果只想演示单步）：

| Step | 触发语（直接粘贴） | 对应 Skill 蓝图 |
|:----:|-------------------|----------------|
| 1 | "帮我把 ylly/gitlink-cli 所有没标签的开放 Issue 自动分类打标签" | issue-triage 工作流 1 |
| 2 | "给我一份 ylly/gitlink-cli 本周的项目周报" | insight 工作流 2 |
| 3 | "根据 ylly/gitlink-cli 自上次发版以来的提交，推荐下一个版本号并生成 Release Notes" | release-auto 一、二 |

**预期 Agent 行为**（截图要点）：
- Agent 会先说"使用 gitlink-issue-triage Skill"
- 调用 `issue +list --state open` → 筛未分类 → `issue +view` 读详情 → 按分类规则表归类 → `issue +update --label`
- 每步都会展示它对应 SKILL.md 的哪一段（这是"Skill 作为设计蓝图"的可视证据）

---

## 工作流 ⑤ 触发语（贡献者成长体系）

**单条完整触发语**：

```
请帮我对 ylly/gitlink-cli 建立贡献者成长体系：
1. 取仓库全历史的 commits（git log）+ merged PR（pr +list --state merged）
   + Issue 作者（issue +list），生成一份综合积分排行榜（公式 commits×1 + PR×5 + Issue×2），
   过滤掉测试号 15972095207 / 2403_89190320 / fsafasff；
2. 按三层标准授予徽章（星级/活跃/贡献者），先创建对应 label（颜色 #FFD700/#FF6B35/#87C95F）；
3. 建一个颁奖 Issue 公布排行榜并 @mention Top 贡献者。
执行前先给我清单确认。
```

**预期 Agent 行为**：
- Agent 调用 insight 工作流 3 的数据采集方法（第 173-183 行）
- 调用 issue-triage 工作流 1 Step 5 的 label 创建（第 86-89 行）
- 输出排行榜表格 + 颁奖动作清单等用户确认

---

## 工作流 ⑦ 触发语（新人全流程保姆）

**单条完整触发语**：

```
请帮我对 ylly/gitlink-cli 做新人引导全流程：
1. 用 member +invite-link 演示邀请能力（仓库无真实新人账号，仅演示）；
2. 扫描开放 Issue，按新人友好识别标准找出适合新人的 Issue（onboarding 工作流1）；
3. 对找出的 good first Issue 写**完全个性化**的引导评论（onboarding 工作流2），
   每条评论要定位到具体文件、给本地准备命令、写 PR 提交规范——禁止用同一模板；
4. 用 notification +list --owner zhangqing23 看通知动态；
5. 交叉验证通知 + Issue + PR 数据，还原一位新人从加入到首次贡献的全过程。
写入操作前给我预览。
```

**分步触发语**：

| Step | 触发语 | 对应 Skill 蓝图 |
|:----:|-------|----------------|
| 2 | "帮我找出 ylly/gitlink-cli 适合新人的 Issue 并打 good first 标签" | onboarding 工作流 1 |
| 3 | "帮我给 ylly/gitlink-cli 适合新人的 Issue 写个性化引导评论" | onboarding 工作流 2 |

**预期 Agent 行为**：
- 识别阶段会引用 onboarding SKILL.md 第 31-46 行的 5 条友好信号 + 3 条排除标准
- 引导评论阶段会引用第 205 行"禁止对所有 Issue 用同一句"，每条评论定位不同文件

---

## 对话产物采集建议

录 Agent 对话时重点截这几段（对应 PDF 评分点）：

1. **Skill 加载声明**：Agent 说"我将使用 gitlink-xxx Skill"的那一句
2. **蓝图引用**：Agent 提到"按 SKILL.md 第 X-Y 行"的段落
3. **命令调用**：Agent 调 gitlink-cli 的命令行（带参数）
4. **真实写入回执**：服务器返回的 `ok:true` / `id` / `version_id` 等
5. **网页验证**：复制返回的 URL 到浏览器打开看到真实效果

---

## 与 reproduce.sh 的关系

| 产物 | 角色 | 适合谁看 |
|------|------|---------|
| **Agent 对话记录**（本文档触发的） | 首要产物，演示"Skill 驱动 Agent" | 评委看过程 |
| `reproduce.sh` | 辅助产物，纯命令复现 | 评委验结果 |
| 3 份执行报告 | 落地证据，含 Skill 蓝图对照 | 评委查细节 |
