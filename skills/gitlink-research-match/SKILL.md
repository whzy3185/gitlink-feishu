---
name: gitlink-research-match
version: 1.0.0
description: "科研协作智能匹配：分析科研仓库的技术栈、未解决 Issue 与研究缺口，构建需求画像，从 GitLink 平台匹配合适的候选学者/团队，生成《科研协作推荐方案》（含推荐理由、建议认领的 Issue、外联模板）。当课题组需要寻找跨团队协作者、为开放 Issue 招募贡献者、或评估潜在合作对象时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli search --help"
---

# gitlink-research-match（科研协作智能匹配）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能为只读分析型；任何外联/邀请动作只生成草稿，须由用户亲自发出。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能概述

本技能回答科研协作场景的一个核心问题：**「我这个项目缺人/缺能力，平台上谁最适合来帮忙或合作？」**

它从**仓库的真实需求**（技术栈、未解决 Issue、研究缺口）出发，构建**需求画像**，再到 GitLink 平台**匹配候选协作者/团队**，输出一份**《科研协作推荐方案》**：谁、为什么、可以从哪个 Issue 切入、以及一封可直接发出的外联草稿。

### 匹配三步法

```
需求侧画像  →  候选侧检索与画像  →  匹配打分与推荐
（我要什么）     （谁可能合适）        （谁最匹配 + 怎么对接）
```

---

## 一、需求侧画像：这个仓库到底缺什么

### 1.1 技术栈与领域

```bash
gitlink-cli repo +info      --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme    --owner <owner> --repo <repo> --format json
```

提取：主语言/框架、研究方向关键词（从 name/description/README）、领域标签。

### 1.2 未解决需求（开放 Issue / 招募信号）

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
```

识别**招募/协作信号**：
```
高价值匹配信号（Issue 标题/标签/正文）：
  - help wanted / 求助 / 招募 / 寻求合作
  - good first issue / 新手友好（适合引流新贡献者）
  - 标签含技术领域：cuda / nlp / 前端 / 文档 / 数据标注
  - 长期未被认领（无 assignee 且开放 > 14 天）
```

### 1.3 当前团队能力盲点

```bash
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

结合 1.1 的技术栈，推断**能力缺口**：例如仓库以算法（Python）为主、但有大量前端/文档/部署类未解决 Issue → 缺工程/文档协作者。

### 1.4 形成需求画像

```
需求画像示例：
  领域：医学影像分割（CV / 深度学习）
  主技术栈：Python / PyTorch
  紧缺能力：① CUDA 性能优化  ② 推理服务化（部署）  ③ 英文文档
  可切入 Issue：#42(help wanted, CUDA), #51(good first issue, docs)
```

---

## 二、候选侧检索与画像：平台上谁可能合适

### 2.1 按能力关键词检索候选

```bash
# 按研究方向/技术关键词找相关仓库与其作者
gitlink-cli search +repos --keyword "medical image segmentation" --format json
gitlink-cli search +repos --keyword "CUDA 优化" --format json

# 直接检索用户
gitlink-cli search +users --keyword "<领域/方向关键词>" --format json
```

### 2.2 候选画像（复用画像能力）

对每个候选用户/组织，采集其产出与方向，判断匹配度：

```bash
gitlink-cli user +info  --login <login>     --format json
gitlink-cli repo +list  --user  <login>     --format json
gitlink-cli org +members --id   <org_name>  --format json   # 候选是团队时
```

> 💡 如需更完整的候选科研画像，可联动 `gitlink-scholar-profile` 技能生成该候选的影响力画像，再回填到本匹配流程。

### 2.3 候选要点提取

```
对每个候选提取：
  - 方向吻合度：候选代表作方向 vs 需求领域
  - 技术吻合度：候选主语言 vs 需求技术栈
  - 能力证据：候选是否有需求紧缺能力的相关仓库（如 CUDA kernel、部署、文档站）
  - 活跃度：近 6 个月是否活跃（避免推荐"休眠"账号）
  - 协作开放度：是否参与过他人仓库（有外部贡献历史 = 更可能接受协作）
```

---

## 三、匹配打分与推荐

### 3.1 匹配度评分（每个候选 0–100）

```
匹配分 = 方向吻合 ×0.30 + 技术吻合 ×0.25 + 能力证据 ×0.25 + 活跃度 ×0.10 + 协作开放度 ×0.10

各项 0–100 评分指引：
  方向吻合：完全同方向=100，相邻方向=70，弱相关=40，无关=0
  技术吻合：主语言+框架一致=100，语言一致=70，可迁移=40
  能力证据：有紧缺能力的代表作=100，有相关但非代表=60，仅声明=30，无=0
  活跃度：  近1月活跃=100，近3月=80，近6月=60，更久=20
  协作开放度：常参与他人仓库=100，偶尔=60，仅自有仓库=30
```

### 3.2 推荐分级

| 匹配分 | 推荐级别 | 建议动作 |
|--------|----------|----------|
| ≥80 | ⭐⭐⭐ 强烈推荐 | 优先发出协作邀请，附具体 Issue |
| 60–79 | ⭐⭐ 推荐 | 可邀请参与某个 good-first / 子任务试合作 |
| 40–59 | ⭐ 候选 | 观望/作为备选 |
| <40 | — | 不推荐 |

---

## 四、《科研协作推荐方案》报告模板

```markdown
## 🤝 科研协作推荐方案

**目标仓库：** <owner>/<repo>
**生成时间：** 2026-06-15
**数据来源：** GitLink 平台（gitlink-cli 只读采集）

---

### 一、需求画像

| 项目 | 内容 |
|------|------|
| 研究领域 | 医学影像分割（CV / 深度学习） |
| 主技术栈 | Python / PyTorch |
| 紧缺能力 | ① CUDA 性能优化　② 推理服务化　③ 英文文档 |
| 可切入 Issue | #42（help wanted, CUDA）、#51（good first issue, docs） |

### 二、推荐协作者 / 团队

#### ⭐⭐⭐ 强烈推荐：`@cuda_master`（匹配分 88）

| 维度 | 评估 |
|------|------|
| 方向吻合 | 相邻方向：医学影像加速（90） |
| 技术吻合 | Python/PyTorch + CUDA（95） |
| 能力证据 | 代表作 `fast-seg-cuda`（⭐120，CUDA kernel 优化）（95） |
| 活跃度 | 近 1 月活跃（100） |
| 协作开放度 | 常给他人仓库提 PR（80） |

**推荐理由：** 正好补齐"CUDA 性能优化"这一最紧缺能力，且方向相邻、近期活跃、有外部协作习惯。
**建议切入：** Issue #42（CUDA 推理加速）。

#### ⭐⭐ 推荐：`@docs-helper`（匹配分 72）
> 擅长英文技术文档站，匹配"英文文档"需求；建议从 #51（good first issue）切入。

### 三、外联草稿（请用户亲自发出）

> **致 @cuda_master：**
> 你好！我们在 GitLink 维护 `someorg/medseg`（医学影像分割）。注意到你的 `fast-seg-cuda`
> 在 CUDA 加速上很出色。我们有一个推理加速需求（Issue #42），不知是否有兴趣参与协作或交流？
> 期待你的回复，谢谢！

### 四、对接建议

1. 先用一个 good-first / 子任务建立轻量协作，再评估深度合作。
2. 给被邀请者准备好可复现环境（可联动 gitlink-research-reproducibility 评估自身仓库复现性）。
3. 在对应 Issue 上 @ 对方并说明上下文，降低参与门槛。
```

---

## 五、执行步骤总览

```bash
# Step 1：需求侧画像
gitlink-cli repo +info       --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages  --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme     --owner <owner> --repo <repo> --format json
gitlink-cli issue +list      --owner <owner> --repo <repo> --state open --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json

# Step 2：候选侧检索
gitlink-cli search +repos --keyword "<方向/能力关键词>" --format json
gitlink-cli search +users --keyword "<方向/能力关键词>" --format json

# Step 3：候选画像
gitlink-cli user +info --login <login> --format json
gitlink-cli repo +list --user <login>  --format json

# Step 4：AI 按第三节规则打分、第四节模板出推荐方案
```

---

## 注意事项

- ✅ **纯只读 + 人在回路**：本技能只生成推荐与外联草稿，**绝不替用户发出邀请或评论**。
- ⚠️ **隐私与尊重**：候选画像仅基于公开仓库数据；推荐用于"协作邀请"，不用于任何骚扰式批量触达。
- ⚠️ **登录名解析**：GitLink 显示名与登录名（login）可能不同；检索到候选后用 `search +users` 校正 login，避免 404（参考 gitlink-scholar-profile 的解析流程）。
- ✅ **可与其他科研 Skill 联动**：候选深度画像→`gitlink-scholar-profile`；邀请前自查复现性→`gitlink-research-reproducibility`。
- ✅ **最终产出为 Markdown 推荐方案**，含匹配分、推荐理由、可切入 Issue 与外联草稿。
