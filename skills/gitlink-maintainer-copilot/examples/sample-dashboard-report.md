# GitLink Maintainer Copilot Report

> 样例报告使用模拟数据，适合比赛 README、截图或录屏展示。真实执行时必须使用 `gitlink-cli` 采集 Evidence Pack。

## 维护状态总览

项目：`Gitlink/gitlink-cli`

总体判断：维护状态为 **Medium Risk**。项目已有清晰 CLI 能力和 Agent Skills 基础，但 PR 响应、Release 节奏和新人引导材料需要进一步稳定。

可信度：High。已采集仓库详情、Issue、PR、Release、Commit、贡献者和 README 数据；CI 数据缺失。

## Evidence Pack

| 编号 | 摘要 |
|---|---|
| E1 repo_profile | 仓库有 README、默认分支为 `master`，License 存在，watchers=42，forked=18 |
| E2 open_issues | open Issue 16 个，其中 5 个超过 14 天未更新 |
| E3 closed_issues | 最近关闭 Issue 22 个，说明仍有维护活动 |
| E4 open_prs | open PR 7 个，其中 3 个超过 7 天未更新 |
| E5 merged_prs | 最近合并 PR 11 个，主要集中在 CLI bugfix 和文档 |
| E7 releases | 最近 Release 距今 45 天 |
| E8 ci_builds | 数据缺失，无法判断 CI 稳定性 |
| E9 commits | 最近 100 个提交中包含 18 个 fix、9 个 docs、6 个 feat |
| E10 contributors | 贡献者 6 人，前 2 名贡献占比约 76% |
| E12 readme | README 有安装说明，但贡献入口和新人任务入口不明显 |

## 关键风险

1. **PR 堵塞风险中等**：[E4] open PR 7 个，3 个超过 7 天未更新，可能降低外部贡献者反馈体验。
2. **新人转化不足**：[E12] README 缺少明确新人入口；[E2] open Issue 中可领取任务标识不足。
3. **Release 节奏不稳定**：[E7] 最近 Release 距今 45 天；[E9] 已累计多项 fix/feat/docs 变更。
4. **贡献者集中度偏高**：[E10] 前 2 名贡献占比约 76%，存在维护者负载集中风险。
5. **CI 可信度缺失**：[E8] 未采集到 CI 数据，无法确认基础自动化质量。

## 匹配 Playbooks

### P2 PR 堵塞清理

触发证据：[E4] open PR 7 个，其中 3 个超过 7 天未更新。

本周动作：

- 对所有 open PR 留下最新维护者反馈。
- 将 PR 分为“可合并 / 需要修改 / 已过期”三组。
- 优先处理文档和低风险 bugfix PR。

### P3 新人友好改造

触发证据：[E12] README 新人入口不足；[E2] open Issue 缺少可领取标识。

本周动作：

- 在 README 增加“首次贡献”小节。
- 选择 3 个低风险 Issue 补充复现步骤和验收标准。
- 给新人任务添加清晰标签或标题前缀。

### P4 Release 稳定化

触发证据：[E7] Release 间隔较长；[E9] 最近已有多项用户可见变更。

本周动作：

- 生成下一版 Release Notes 草稿。
- 将最近合并变更归类为 Added / Fixed / Docs。
- 确认 CI 或手工验证结果后再发布。

## 30/60/90 天治理计划

30 天：

- [ ] 所有 open PR 都有维护者反馈，超过 14 天未更新的 PR 降到 0。
- [ ] README 增加首次贡献入口和本地运行最短路径。
- [ ] 建立下一个 Release 的变更清单草稿。

60 天：

- [ ] 固化 PR 评审 SLA，例如 7 天内首次响应。
- [ ] 整理 3-5 个适合新人领取的 Issue。
- [ ] 补齐 Issue/PR 模板，降低沟通成本。

90 天：

- [ ] 形成每月维护节奏，固定回顾 Issue、PR、Release 状态。
- [ ] 将基础检查纳入 CI 或公开验证说明。
- [ ] 复盘新人 Issue 的领取和合并情况。

## Governance Issue 草稿

标题：

```text
chore: 建立 gitlink-cli 项目维护治理计划
```

正文摘要：

```markdown
## 背景

本 Issue 由 GitLink Maintainer Copilot 根据仓库数据生成，用于跟踪项目维护治理动作。

## 诊断结论

风险等级：Medium

匹配治理剧本：

- P2 PR 堵塞清理：[E4] open PR 7 个，3 个超过 7 天未更新。
- P3 新人友好改造：[E12] README 新人入口不足。
- P4 Release 稳定化：[E7] Release 距今 45 天。

## 本周建议动作

- [ ] 回复所有超过 7 天未更新的 open PR。
- [ ] 补 README 首次贡献入口。
- [ ] 生成下一版 Release Notes 草稿。
```

## 数据缺失与可信度

| 缺失项 | 影响 | 建议 |
|---|---|---|
| E8 ci_builds | 无法判断 CI 稳定性 | 维护者可补充 CI 权限或手工验证记录 |

确认写入前，Agent 必须展示完整 Issue 正文并等待用户明确确认。
