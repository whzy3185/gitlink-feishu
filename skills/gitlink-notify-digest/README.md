# gitlink-notify-digest · 通知智能摘要（使用说明）

> 任务二**创新增强** Skill · 作者 ylly

## 是什么
采集用户通知，AI 按重要性分级（被 @/分配/review 为高优）+ 分类汇总，输出**每日通知摘要**，让用户快速抓重点。复用任务一 notification 命令。

## 解决的痛点
通知刷屏、重要信息（@我/分配/review）被淹没。

## 分级模型
🔴高（@/分配/review/合并驳回）→ 🟡中（新issue/pr/成员）→ 🟢低（系统/流水线）

## 怎么用
```
请阅读 skills/gitlink-notify-digest/SKILL.md，为我（ylly）生成今日通知摘要。
```

## 验证案例
ylly 5 条通知 → 分级：🔴1（被分配 IssueAssigned）/ 🟡1（MemberJoined）/ 🟢流水线 → 摘要让"被分配"高优凸显。详见 verification.md。

## 文件清单
SKILL.md（分级模型）/ README.md / verification.md
