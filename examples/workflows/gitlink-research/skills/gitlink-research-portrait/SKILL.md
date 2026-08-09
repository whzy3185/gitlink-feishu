---
name: gitlink-research-portrait
version: 1.0.0
description: "科研贡献者深度画像：六维度刻画每位贡献者的科研画像（技能/活跃度/影响力/协作偏好/贡献模式/科研角色），输出个人画像卡。当需要深度了解某贡献者的科研能力与协作风格、或为科研团队做人才盘点时触发。任务四创新场景（PDF 之外）。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-portrait（科研贡献者深度画像 · 创新科研 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读分析，不写入仓库。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务四**创新**科研 Skill（PDF 之外的原创场景）。比 research-matching 的"技能画像"和 research-graph 的"协作网络"更深——为**每位贡献者生成一份完整科研画像**（六维度），用于人才盘点、角色识别、协作风格分析。体现任务四的创新性。

---

## 画像六维模型

| 维度 | 分析 | 科研含义 |
|------|------|---------|
| 🛠 技能领域 | commit/PR 涉及模块 | 擅长什么（核心/API/前端/CI）|
| 🔥 活跃度 | commit 频率/时间跨度 | 投入程度 |
| 📈 影响力 | PR 合并率/被 review/被引用 | 社区认可 |
| 🤝 协作偏好 | 独立/协作/审查型 | 协作风格 |
| 🎯 贡献模式 | 提交/审查/提问/统筹 | 工作类型 |
| 🎓 科研角色 | 核心/活跃/边缘/导师 | 团队定位 |

---

## 工作流

### Step 1：采集个人活动
```bash
git log --author="<贡献者>" --format="%ad|%s" --date=short   # 该人的 commit 历史
git log --author="<贡献者>" --name-only                       # 涉及的模块/文件
gitlink-cli pr +list --owner <o> --repo <r> --format json     # 该人的 PR
gitlink-cli issue +list --owner <o> --repo <r> --format json  # 该人的 Issue
```

### Step 2：AI 六维分析
综合该人的 commit/PR/Issue，按六维画像。

### Step 3：输出个人画像卡
```markdown
## 🧑‍🔬 科研贡献者画像 — <贡献者>

### 🛠 技能领域
Go核心架构 / CI/CD / Skill设计（涉及 internal/cmd/.github/skills）

### 🔥 活跃度
高（42 commits，跨度 X 月，持续贡献）

### 📈 影响力
核心枢纽（PR 合并率高，被多人 review，统筹合并）

### 🤝 协作偏好
统筹审查型（review/merge 他人 PR 为主）

### 🎯 贡献模式
统筹型（合并 + 跨模块协调）

### 🎓 科研角色
⭐ 核心 + 导师（项目维护者，引导多人协作）

### 画像总结
<一句话概括该人在科研团队中的定位>
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| 单人 commit 少画像模糊 | 标注"贡献数据不足，画像置信度低" |
| 技能推断主观 | 结合 commit message + 文件路径 + PR 分支名 |
| 时间跨度需多个 commit | 取首末 commit 日期算跨度 |
| 影响力需 review 数据 | `pr +reviews` 采样 |

---

## 实测落地参考
**Gitlink/gitlink-cli · wbtiger 画像**（核心贡献者）：
- 🛠 技能：Go核心 + CI + 统筹（涉及 internal/cmd/.github，跨所有模块）
- 🔥 活跃：高（42 commits，项目全程参与）
- 📈 影响：核心枢纽（合并大量 PR，被广泛 review）
- 🤝 协作：统筹审查型（主导合并 + 协调多人）
- 🎯 模式：统筹型
- 🎓 角色：⭐ 核心 + 导师（项目维护者）
- **总结**：gitlink-cli 的核心维护者，承担架构+CI+统筹+导师角色。

详见 verification.md。
