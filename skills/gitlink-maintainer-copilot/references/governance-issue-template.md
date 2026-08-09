# Governance Issue Template

创建治理 Issue 前，必须先把完整草稿展示给用户并获得明确确认。

## 标题模板

```text
chore: 建立 <repo> 项目维护治理计划
```

## 正文模板

```markdown
## 背景

本 Issue 由 GitLink Maintainer Copilot 根据仓库公开/授权数据生成，用于跟踪项目维护治理动作。

## Evidence Pack

- [E1: repo_profile] <仓库基础信息摘要>
- [E2: open_issues] <open Issue 摘要>
- [E4: open_prs] <open PR 摘要>
- [E7: releases] <Release 摘要>
- [E9: commits] <提交活跃摘要>
- [E10: contributors] <贡献者摘要>

数据缺失：

- <如无缺失，写“无关键缺失”。>

## 诊断结论

风险等级：<High / Medium / Low>

匹配治理剧本：

- <Playbook 名称>：<触发证据>

关键风险：

1. <风险 1，引用证据>
2. <风险 2，引用证据>
3. <风险 3，引用证据>

## 30/60/90 天计划

30 天：

- [ ] <任务，含验收标准>
- [ ] <任务，含验收标准>

60 天：

- [ ] <任务，含验收标准>
- [ ] <任务，含验收标准>

90 天：

- [ ] <任务，含验收标准>
- [ ] <任务，含验收标准>

## 本周建议动作

- [ ] <最小可执行动作 1>
- [ ] <最小可执行动作 2>
- [ ] <最小可执行动作 3>

## 验收标准

- <可验证标准 1>
- <可验证标准 2>
- <可验证标准 3>

---

生成方式：GitLink Maintainer Copilot Skill
```

## 写入命令

用户确认后执行：

```bash
gitlink-cli issue +create --owner <owner> --repo <repo> \
  --title "chore: 建立 <repo> 项目维护治理计划" \
  --body "<上方正文>"
```

## 禁止事项

- 不要批量创建多个治理 Issue。
- 不要在未确认前执行写入。
- 不要把 Token、私有邮箱、调试日志放入 Issue 正文。
- 不要把缺失数据写成确定结论。
