# GitLink 飞书完整实现入口

本分支集中保存：

- 飞书原有导出、卡片、Base、Doc 和 Task preview 代码；
- P0、P1、P1.1、P2.0、P2.1 Review 协作实现；
- Channel SDK、可靠 Gateway、SQLite Job 和回复代码；
- GitLink GET-only Review Queue / Context；
- Agent Skill、测试、fixture 和实施记录；
- 飞书开放平台研究；
- 企业微信对照方案；
- 复赛准备文档和实验样例；
- 当前入站消息排障记录与飞书 Agent 回复。

完整文件地图、阶段提交、安全排除、验证方式和推荐阅读顺序：

- [GitLink 飞书完整实现与资料索引](docs/FEISHU_COMPLETE_IMPLEMENTATION_INDEX.md)

当前机器人能力：

- [GitLink 飞书 Review 机器人能力总结](docs/FEISHU_REVIEW_BOT_CAPABILITY_SUMMARY.md)

当前排障：

- [飞书 Agent 排障交接](docs/FEISHU_AGENT_DEBUG_HANDOFF.md)
- [飞书 Agent 回复与本地行动修订](docs/FEISHU_AGENT_RESPONSE_20260731.md)
- [P2.1.1 收口与真实验收规划](docs/ROUND2_PR_REVIEW_COLLABORATION_P211_CLOSEOUT_PLAN.md)
- [P2–P5 实施记录](docs/ROUND2_PR_REVIEW_COLLABORATION_P2_P5_IMPLEMENTATION.md)
- [P2–P5 P0 收口记录](docs/ROUND2_P0_CLOSEOUT_20260731.md)

当前边界：

```text
飞书应用鉴权和出站消息成立
GitLink Review Core GET-only 成立
入站消息到业务 handler 尚待收口
不允许从飞书批准、拒绝、评论、Reviewer 或合并写入
```
