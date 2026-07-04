# gitlink-research-insight · 科研项目洞悉（使用说明）

> 任务四（应用 GitLink 辅助科研）核心交付 · 科研辅助 Skill
> 作者：ylly

---

## 一、这是什么

**gitlink-research-insight** 是一个科研辅助 Skill：用 `gitlink-cli` 采集 GitLink 仓库的协作数据，AI 按五维模型分析，**评估一个开源项目作为"科研项目 / 科研工具"的价值**，输出科研洞悉报告。

## 二、解决什么真实问题

科研工作者、课题组面对海量开源项目，常困惑：
- 这个仓库**还在维护吗**？（能否复现）
- **有多少人在用**？（值不值得引用）
- **文档全不全**？（复现门槛高不高）
- **社区活跃吗**？（可持续吗）
- **适合作为我的研究对象吗**？

本 Skill **自动回答这 5 个问题**，辅助科研选题、复现选型、协作评估。

## 三、五维科研洞悉模型

| 维度 | 看什么 | 回答的科研问题 |
|------|--------|--------------|
| 🔥 活跃度 | issue/pr 频率、最近更新 | 项目持续维护吗？（可复现性）|
| 📈 影响力 | fork/star/贡献者数、PR 合并率 | 社区认可吗？（引用价值）|
| 🏗 成熟度 | release 版本、README 完整性、LICENSE | 稳定可靠吗？（可靠性）|
| 🤝 协作健康 | 贡献者分布、issue 响应 | 社区活跃吗？（可持续性）|
| 🎓 科研价值 | 综合四维 + 技术栈适配 | 适合做研究对象/复现基础吗？|

## 四、怎么用

### 方式 1：Claude Code 一句话触发（推荐）
```
请阅读 examples/workflows/gitlink-research/skills/gitlink-research-insight/SKILL.md，
对 Gitlink/gitlink-cli 做科研项目洞悉评估。
```
AI 会自动按五维采集数据 + 分析 + 输出洞悉报告。

### 方式 2：手动按 SKILL.md 的命令采集
```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --format json
gitlink-cli release +list --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo>
```
采集后按五维模型人工/AI 分析。

## 五、验证案例

**评估对象**：`Gitlink/gitlink-cli`（GitLink 官方 AI Agent CLI 工具，任务四背景明确其"连接科研工作者"）

**结果**：综合 **⭐ 4.4 / 5（89 分）— 优秀**
- 🔥 活跃度 ⭐4.5（322 PR + 持续发版）
- 📈 影响力 ⭐4.0（41 forks / 29 贡献者）
- 🏗 成熟度 ⭐4.8（12 Release + 3.4万字 README）
- 🤝 协作健康 ⭐4.4（29 贡献者活跃协作）
- 🎓 科研价值 ⭐4.5（AI Agent 工具适配科研）

详见 [`verification.md`](./verification.md)。

## 六、适用场景

| 场景 | 用法 |
|------|------|
| **科研选题** | 评估候选开源项目，挑活跃+成熟+有价值的作为研究方向 |
| **复现选型** | 选文档全、稳定、活跃的项目复现（降低复现失败风险）|
| **协作评估** | 了解项目社区健康度，判断是否值得加入贡献 |
| **工具引用** | 评估工具的影响力和可靠性，决定是否引用到科研流程 |

## 七、输出示例

```markdown
## 🔬 科研项目洞悉报告 — <owner>/<repo>
综合科研评分：⭐4.4/5（89/100）
[五维评分表 + 关键发现 + 科研使用建议]
```

## 八、文件清单

```
gitlink-research-insight/
├── SKILL.md          ← Skill 本体（五维模型 + 工作流 + 避坑）
├── README.md         ← 本文件（中文使用说明）
└── verification.md   ← 真实验证报告（Gitlink/gitlink-cli ⭐4.4/5）
```

## 九、注意事项

- 本 Skill 为**只读采集 + 分析**，不写入任何仓库（安全）
- 中文仓库名可能有 API 编码坑，优先选英文 repo 名
- `contributors` API 可能返回 HTML，降级用 `git log` 聚合作者
- 科研洞悉是**辅助决策**，不替代人工判断
