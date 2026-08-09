---
name: gitlink-research-insight
version: 1.0.0
description: "科研项目洞悉：采集 GitLink 仓库的 issue/pr/contributors/release/commit 数据，AI 多维分析活跃度/影响力/成熟度/协作健康度/科研价值，输出科研项目洞悉报告。当科研工作者需要评估一个开源仓库是否适合作为研究对象或复现基础、课题组需要了解项目科研价值时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-insight（科研项目洞悉 · 科研辅助 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)（认证、权限、API 注意事项）。**
**CRITICAL — 本 Skill 为只读采集 + 分析，不写入任何仓库。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 gh（GitHub CLI）操作 GitLink 资源。**

> **定位**：任务四科研辅助 Skill。面向科研工作者 / 课题组，将 GitLink 仓库的协作数据转化为**科研洞悉**——评估一个开源项目作为"科研项目 / 科研工具"的活跃度、影响力、成熟度、协作健康度和科研价值，辅助科研选题、复现选型、协作评估。

---

## 科研洞悉五维模型

| 维度 | 采集指标 | 科研含义 |
|------|---------|---------|
| 🔥 活跃度 | issue/pr 频率、最近更新时间 | 项目是否持续维护（科研**可复现性**前提）|
| 📈 影响力 | fork / star / 贡献者数、PR 合并率 | 社区认可度（科研**引用价值**）|
| 🏗 成熟度 | release 版本数、文档完整性、LICENSE | 项目是否稳定可用（科研**可靠性**）|
| 🤝 协作健康 | 贡献者分布、issue 响应、社区参与 | 社区是否活跃（科研**可持续性**）|
| 🎓 科研价值 | 综合上述 + 技术栈适配 | 是否适合作为**研究对象 / 复现基础 / 工具引用** |

---

## 工作流

### Step 1：采集仓库元数据（活跃度 + 影响力 + 成熟度）

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
# 提取：forked_count / stars_count / watchers_count / issues_count / pull_requests_count
#       contributor_users_count / created_at / updated_at / license

gitlink-cli repo +readme --owner <owner> --repo <repo>
# 评估文档质量（README 是否完整：安装/使用/示例）

MSYS_NO_PATHCONV=1 gitlink-cli api GET /<owner>/<repo>/languages.json --format json
# 技术栈（判断科研适配：Python/AI 框架 → 适合 ML 研究）
```

### Step 2：采集协作数据（协作健康度）

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json    # 活跃 issue
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json  # 历史响应
gitlink-cli pr +list --owner <owner> --repo <repo> --format json                    # PR 协作活跃度

# 贡献者（contributors endpoint 可能返回 HTML，降级方案）：
MSYS_NO_PATHCONV=1 gitlink-cli api GET /<owner>/<repo>/contributors.json --format json
# 若返回 HTML，改用 commit 作者聚合（git log --format="%an" | sort | uniq -c）
```

### Step 3：AI 多维评分（0-100 / 5星）

综合采集数据，按五维模型评分，每维给出**依据 + 科研含义**：

| 维度 | 评分依据（示例）|
|------|---------------|
| 🔥 活跃度 | 最近更新 < 30天 + 有 open issue → 高；> 1年无更新 → 低 |
| 📈 影响力 | fork ≥ 20 + 贡献者 ≥ 10 → 高；PR 合并率高 → 社区认可 |
| 🏗 成熟度 | 有 release + README 完整 + LICENSE → 高；无文档 → 低 |
| 🤝 协作健康 | 贡献者分布均匀 + issue 响应及时 → 高；单人项目 → 中 |
| 🎓 科研价值 | 综合四维 + 技术栈适配科研方向 → 给出科研使用建议 |

### Step 4：输出科研洞悉报告

```markdown
## 🔬 科研项目洞悉报告 — <owner>/<repo>

📅 评估时间：<YYYY-MM-DD>
🎯 项目定位：<AI 工具 / 算法实现 / 数据集 / 论文复现 / ...>

### 综合科研评分：⭐x.x / 5（xx / 100）

| 维度 | 评分 | 依据 | 科研含义 |
|------|:----:|------|---------|
| 🔥 活跃度 | xx | N issue/PR，最近更新 X 天前 | <持续维护/已停滞> |
| 📈 影响力 | xx | N forks, N 贡献者 | <社区认可/小众> |
| 🏗 成熟度 | xx | N release, README 完整度 | <稳定/实验性> |
| 🤝 协作健康 | xx | 贡献者分布, 响应 | <活跃社区/个人项目> |
| 🎓 科研价值 | xx | 综合 + 技术栈 | <高/中/低> |

### 🔍 关键发现
1. <最突出的优势/风险>
2. <次要发现>

### 🎓 科研使用建议
- **适合作为**：研究对象 / 复现基础 / 工具引用 / 数据来源
- **注意事项**：<复现风险、依赖、文档缺口等>
- **建议动作**：< fork 复现 / 引用 / 关注 / 谨慎>
```

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| `contributors` API 返回 HTML 非 JSON | 降级用 `git log --format="%an" \| sort \| uniq -c` 聚合作者 |
| 中文仓库名 URL 编码失败（如"论文复现"）| 优先选英文 repo 名；或 `api GET` 传 URL 编码路径 |
| `api GET` 路径在 Git Bash 被转成 Windows 路径 | 加 `MSYS_NO_PATHCONV=1` |
| `pr +list --state` 过滤不精确 | 客户端按 `pull_request_status` 二次判断（0=open,1=merged,2=closed）|
| fork 仓库 PR/Release 为 0 | 如实反映（fork 无独立 PR/发版），不影响活跃度判断 |
| `issue +list` 返回数组含已关闭 | 客户端按 `status.id` 二次过滤（1=开放）|

---

## 实测落地参考

**验证仓库**：`Gitlink/gitlink-cli`（GitLink 官方 AI Agent CLI 工具——任务四背景明确其"连接开发者、科研工作者与智能化工作流，为科研团队提供辅助"）

| 维度 | 实测结果 |
|------|---------|
| 🔥 活跃度 | 高（322 PR、19 issue，持续更新）|
| 📈 影响力 | 中上（41 forks、13 watchers、29 贡献者）|
| 🏗 成熟度 | 高（MulanPSL-2.0 LICENSE、README 完整、有发版）|
| 🤝 协作健康 | 高（29 贡献者协作，322 PR 显示活跃 review 流）|
| 🎓 科研价值 | 高（AI Agent 工具，支撑科研智能化的研究对象）|

**结论**：gitlink-cli 作为"AI 辅助科研工具"的协作生态，是研究"开源 AI 工具如何支撑科研"的典型样本。

> 说明：纯科研类仓库（如论文复现）中文名常有 API 编码坑，故选用数据丰富且贴合任务四背景的官方 AI 工具仓库验证。科研洞悉方法同样适用于任意 GitLink 仓库。
