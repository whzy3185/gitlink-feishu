---
name: gitlink-maintainer-copilot
version: 1.0.0
description: "Use when a maintainer wants a GitLink project diagnosis, governance plan, contributor-readiness check, or a confirmed governance Issue for a repository."
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli --help"
---

# gitlink-maintainer-copilot

**CRITICAL - 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理、JSON 输出和 GitLink 工具边界。**
**CRITICAL - 默认只读。只有用户明确确认后，才允许创建治理 Issue 或发表评论。**
**CRITICAL - 所有结论必须绑定证据；无法采集的数据必须标记为“数据缺失”，不得猜测。**

## 目标

本 Skill 帮助 Agent 把 GitLink 仓库数据转化为维护者可执行的驾驶舱：

- Maintainer Evidence Pack：采集仓库、Issue、PR、Commit、贡献者、Release、CI、README 等证据。
- Playbook Diagnosis：根据证据匹配维护剧本，解释风险来源。
- 30/60/90 天治理计划：把风险变成可执行任务。
- Governance Issue Draft：生成可确认落地的治理 Issue 草稿。

详细规则见：

- [`references/evidence-pack.md`](references/evidence-pack.md)
- [`references/playbooks.md`](references/playbooks.md)
- [`references/governance-issue-template.md`](references/governance-issue-template.md)

## 适用场景

当用户提出以下需求时使用本 Skill：

- “帮我分析这个 GitLink 仓库是否健康”
- “给项目做维护者驾驶舱”
- “找出 Issue/PR 堵塞点”
- “给开源项目生成治理计划”
- “帮我创建一个项目治理 Issue”
- “比赛演示需要一个可落地的 GitLink Skill”

不要用于：

- 单纯查看某个 Issue、PR、Release 的详情；改用对应领域 Skill。
- 直接批量修改 Issue、关闭 PR、删除资源。
- GitHub/GitLab 仓库；本 Skill 只面向 GitLink。

## 工作流

### 1. 建立上下文

1. 阅读 `gitlink-shared`。
2. 确认 `owner` 和 `repo`。如果当前目录是 GitLink 仓库，可以依赖 `gitlink-cli` 自动从 remote 解析。
3. 说明默认只读，并告诉用户写入治理 Issue 前会二次确认。

### 2. 采集证据

优先使用 Shortcuts，所有命令都加 `--format json`。

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --limit 100 --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --limit 100 --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --limit 100 --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --limit 100 --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state closed --limit 100 --format json
gitlink-cli release +list --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
```

Shortcuts 未覆盖时使用 Raw API：

```bash
gitlink-cli api GET /v1/<owner>/<repo>/commits --query 'page=1&limit=100' --format json
gitlink-cli api GET /v1/<owner>/<repo>/contributors/stat --format json
gitlink-cli api GET /<owner>/<repo>/languages --format json
gitlink-cli api GET /<owner>/<repo>/readme --format json
```

如果某个命令失败，不要中断整个诊断。记录：

- 命令
- 失败原因
- 该缺失会影响哪些判断

### 3. 生成 Evidence Pack

按 [`references/evidence-pack.md`](references/evidence-pack.md) 汇总证据。每条结论都必须能追溯到至少一个证据项。

证据引用格式：

```text
[E3: open_prs] open PR count = 8, oldest updated_at = 2026-03-10
```

### 4. 匹配治理剧本

按 [`references/playbooks.md`](references/playbooks.md) 选择 1-3 个剧本。优先选择证据最充分、对维护者最有行动价值的剧本。

输出时必须包含：

- 匹配剧本名称
- 触发证据
- 风险等级：High / Medium / Low
- 本周可执行动作
- 30/60/90 天治理计划

### 5. 生成驾驶舱报告

报告必须包含：

1. 项目维护状态总览
2. Evidence Pack 摘要
3. 3-5 个关键风险
4. 匹配到的治理剧本
5. 30/60/90 天治理计划
6. Governance Issue 草稿
7. 数据缺失和可信度说明

参考 [`examples/sample-dashboard-report.md`](examples/sample-dashboard-report.md)。

### 6. 写入确认闸门

默认不要写入 GitLink。只有当用户明确说“创建治理 Issue”“发布这个 Issue”“确认写入”等同义指令后，才可以执行写入。

写入前必须展示将执行的命令和完整 Issue 内容：

```bash
gitlink-cli issue +create --owner <owner> --repo <repo> \
  --title "<governance issue title>" \
  --body "<governance issue body>"
```

用户确认后只创建一个治理 Issue。不要批量创建 Issue，不要关闭 Issue，不要评论 PR，除非用户另行明确要求。

## 输出准则

- 用中文输出，保留 Evidence Pack、Playbook、Governance Issue 等少量英文术语。
- 风险判断要克制：有证据就判断，无证据就标注缺失。
- 建议必须能被维护者执行，避免“加强管理”“优化流程”这类空话。
- 治理任务要有验收标准，例如“关闭或回复 10 个超过 14 天未更新的 Issue”。
- 如果仓库是新项目或数据量很少，报告重点转为“治理基线补齐”，不要给出虚假的趋势分析。
