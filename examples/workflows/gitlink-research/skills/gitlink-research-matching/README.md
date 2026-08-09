# gitlink-research-matching · 科研协作智能匹配（使用说明）

> 任务四第 4 个 Skill · 覆盖「科研协作智能匹配」· 作者 ylly

## 是什么
从贡献者的 commit/PR 活动推断**技能画像**，按技能互补/研究方向匹配科研合作者，输出协作匹配建议。

## 匹配模型
- 技能画像：commit 涉及的目录→技能（shortcuts→命令开发/skills→Skill设计/internal→核心架构/.github→CI）
- 互补匹配：A+B 技能覆盖全栈 → 推荐组队
- 研究方向：关键词→技能→人

## 怎么用
```
请阅读 .../gitlink-research-matching/SKILL.md，分析 Gitlink/gitlink-cli 的贡献者技能并匹配协作对。
```

## 验证案例
Gitlink/gitlink-cli：wbtiger(Go核心+CI) / ylly(API+Skill) / ZxR(命令+Skill) / zhangqing(命令+Skill) —— **4 人技能互补**，适合组队做 AI 工具开发。详见 verification.md。

## 文件清单
SKILL.md（匹配模型）/ README.md / verification.md
