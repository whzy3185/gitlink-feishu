# gitlink-issue-assigner · Issue 智能分配 @通知（使用说明）

> 任务二**创新增强** Skill · 解决 GitLink 个人仓库 assign 痛点 · 作者 ylly

## 是什么
分析仓库历史贡献（谁修过类似模块），推荐 Issue 负责人，**在评论里 @ 推荐人 + 理由**——绕过 GitLink 个人仓库 assigners 返回空、无法 assign 的限制，实现"@通知式软分配"。

## 解决的痛点
`issue +assigners` 个人仓库返回空 → `PATCH assigned_to_id` 无效 → issue 分拣完没人管。本 Skill 用 **@ 评论通知**代替 assign，让对的人收到通知。

## 怎么用
```
请阅读 skills/gitlink-issue-assigner/SKILL.md，
为 ylly/gitlink-cli 的 #<n> 推荐负责人并 @ 通知。
```

## 验证案例
gitlink/gitlink-cli 某 wiki 相关 issue → `git log -- shortcuts/wiki/` 推荐出 ylly（wiki 主贡献者）→ 评论 @ylly + 理由 → 软分配成功。详见 verification.md。

## 创新点
绕过平台 assign 限制，用 @ 通知实现"软派单"——这是 GitLink 个人仓库场景下**唯一可行**的自动分配方案。

## 文件清单
SKILL.md（@通知工作流）/ README.md / verification.md
