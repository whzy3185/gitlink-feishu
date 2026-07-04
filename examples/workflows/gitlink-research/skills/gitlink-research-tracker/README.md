# gitlink-research-tracker · 科研进度跟踪与预警（使用说明）

> 任务四第 5 个 Skill · 覆盖「进度跟踪与预警」· 作者 ylly

## 是什么
通过 milestone/issue/pr 采集科研项目进度，AI 分析完成度/积压/阻塞，**预警超期/停滞/阻塞**。

## 跟踪模型
里程碑（完成度/截止）+ Issue（积压/未分类）+ PR（阻塞）+ 整体（关闭速率）

## 怎么用
```
请阅读 .../gitlink-research-tracker/SKILL.md，跟踪 Gitlink/gitlink-cli 进度并预警。
```

## 验证案例
Gitlink/gitlink-cli：无正式 milestone（用 Release 节奏 v0.1.x→v0.2.0 稳定迭代）/ Issue 53% 关闭无积压 / PR 322 活跃 → **进度 🟢 良好，无预警**。详见 verification.md。

## 文件清单
SKILL.md（跟踪模型+预警规则）/ README.md / verification.md
