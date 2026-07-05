---
name: gitlink-research-idea
version: 0.1.0
description: "科研 Idea 生成：基于对一个（或多个）科研代码仓库的内涵解读，产出『方法局限 + 可拓展研究 Idea』。单仓模式给一个仓库生成局限与≥3条 Idea；多仓模式对比同主题多仓、提炼共性缺口与跨仓 Idea。当用户说『这个方法还能怎么改进』『有什么研究Idea』『创新启发』『这个方向的下一步』『这几个仓库相比之下还能做什么』时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-idea（科研 Idea 生成）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

## 何时使用 / 何时跳过

**使用本 skill（生成研究 Idea / 创新启发）：**
- 用户说"这个方法还能怎么改进""有什么研究 Idea""创新启发""这个方向的下一步""这几个仓库相比之下还能做什么"
- 用户要在某课题上选题、找创新点、写 future work

**跳过本 skill（改用其他 skill）：**
- 想先读懂"这仓库做了什么" → 先 `gitlink-research-insight`（本 skill 的输入）
- 想查"能不能复现" → `gitlink-research-repro`
- 想看某领域热点/学者画像 → `gitlink-research-tracker` / `gitlink-scholar-profile`

> **与 `gitlink-research-tracker` 的本质区别（差异化自证）**：tracker 在**跨仓**粒度做"搜索 + 计数 + 成熟度评分"（哪个方向热、多少人在做）；本 skill 在**语义**粒度做"方法局限剖析 + 假设构造"（这个方法假设了什么、哪里可破、能提什么新假设）。一个回答"外面流行什么"，一个回答"这里能长出什么新研究"。

## ⚠️ 现实约束（驱动设计的前提）

1. GitLink 科研仓库多为**成品上传、无探索历史**——不能从 commit 挖"作者试过但放弃的方向"。Idea 必须**从内容（方法实现 + 实验配置）的内在矛盾/假设/边界**推导。
2. 单仓 Idea 易"想当然"——所以**每条 Idea 必须三锚**：①缺口依据（代码/配置里的具体证据）②可行性（改哪个文件、复用什么现有接口）③与现有方法的差异（为什么不是重复造轮子）。无证据的 Idea 不产出。

## 两种模式

| 模式 | 输入 | 输出 | 适用 |
|------|------|------|------|
| **单仓模式（A3'，默认）** | 一个仓库的 A1 内涵解读 | 该方法的**局限清单 + ≥3 条可拓展研究 Idea** | 时间紧/只盯一个仓库；多仓凑不齐时的降级兜底 |
| **多仓模式（A3，扩展）** | ≥3 个同主题仓库的 A1 内涵 | **创新点对比矩阵 + 共性缺口 + ≥3 条跨仓 Idea** | 核心差异化场景；需用户给 owner/repo 列表 |

> 单仓模式是本版默认交付；多仓模式复用同一"采集 + gap 合成"引擎，把单仓的"局限"升级为跨仓"共性缺口"。本 skill 的方法局限段即多仓对比的输入单元。

## 数据采集

**单仓模式**：本 skill 建立在 `gitlink-research-insight`（A1）产出之上。若用户尚未给内涵解读，**先跑 insight** 拿到"研究问题/方法/创新/基线/数据集"。再按需用 `sub_entries` 精读方法实现以找"内在矛盾"：

```bash
# 复用 insight 的采集 + 按需精读方法源码（正文在 data.entries.content）
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/<owner>/<repo>/sub_entries" --query "filepath=<方法文件>&ref=master" --format json
# 多仓模式：对每个仓库重复上述（或直接复用各自已生成的 insight 报告）
```

**多仓模式**：仓库来源 = 用户给定 `owner/repo` 列表（推荐，demo 可 curate 3–5 个同主题 CIL 仓）；或 `search +repos`（噪声大，需 AI 筛科研类）。

## Idea 生成流程（openair gap-analysis / hypothesis-construction 内核）

### Step 1：拿到方法的"内涵 + 实现细节"（来自 insight，必要时补读源码）

### Step 2：做"假设—边界"剖析（gap-analysis）
对方法的每个核心机制，追问四个破绽问题（每问对应一类 Idea 来源）：
- **统计依赖**：方法是否依赖某统计量（batch 均值/方差、类比例）？该统计在什么条件下会失效？（→ 稳健化 Idea）
- **作用层级**：方法只在某一层（如 logit 层）作用？同一矛盾在别的层（feature/loss/data）是否也在？（→ 层级推广 Idea）
- **静态 vs 动态**：关键超参/权重是静态固定的？随任务/数据分布变化是否次优？（→ 自适应 Idea）
- **范畴边界**：方法只在某子类（如 logit-KD）验证？同一框架能否统一相邻范畴（feature-KD/跨模态）？（→ 范畴统一/迁移 Idea）

### Step 3：生成 ≥3 条研究 Idea（hypothesis-construction）
每条 Idea **必须含三锚**：
- **缺口依据**：代码/配置里的具体证据（`文件:函数`），不是空想
- **可行性**：改哪里、复用什么现有接口、大致工作量
- **与现有方法的差异**：为什么是新研究而非复现

### Step 4：生成中文「研究 Idea 报告」（按下述模板，Write 工具产出 Markdown）

## 输出模板：研究 Idea 报告（单仓模式）

```markdown
# 💡 研究 Idea 报告：<仓库/论文名>

> 基于：<owner>/<repo> 的内涵解读 ｜ 数据来源：GitLink ｜ 生成时间：{{date}}
> 结论先行：该方法最值得追问的 N 个方向 …

## 一、方法局限剖析（Idea 的土壤）
| # | 局限 | 证据（代码:位置） | 破绽类型 |
|---|------|-------------------|----------|
| 1 | … | `文件:函数` | 统计依赖/层级/静态/范畴 |

## 二、研究 Idea（≥3 条，每条三锚）
### Idea 1：<一句话标题>
- **假设**：…
- **缺口依据**：…（`文件:位置`）
- **可行性**：…（改哪个文件、复用什么、工作量）
- **与现有差异**：…
（重复 ≥3 条）

## 三、优先级建议
（按 可行性 × 预期增益 排序，给一个"先做哪个"的建议 + 一句话理由）
```

> **多仓模式**额外加：①"创新点对比矩阵"（每仓 1–3 条创新 + 代码证据）②"共性研究缺口 ≥2 条"③跨仓 Idea。模板同理扩展。

## 示例

- **单仓模式（A3'）**：[`examples/maintaining-fairness-lkd-cil-ideas.md`](examples/maintaining-fairness-lkd-cil-ideas.md)（靶子仓库：从 `Zscore()` 的 batch 统计、fairness 仅在 logit 层、`--lambd` 静态权重等真实代码证据，推出 feature-level fairness / 自适应权重 / 稳健归一化 等 Idea）。
- **多仓模式（A3）**：[`examples/cil-multi-repo-ideas.md`](examples/cil-multi-repo-ideas.md)（3 个同主题 CIL 仓的创新点对比矩阵 + 共性缺口 + 4 条跨仓 Idea + 差异化自证；含"GitLink 同主题仓稀缺、搜索非全文"的发现与穷举搜索记录）。

## 注意事项

- ✅ **每条 Idea 必须有代码证据**（`文件:函数`），无证据不产出——这是与"空想 future work"的分界。
- ✅ **先 insight 后 idea**：单仓模式依赖 A1 内涵；若用户直接要 Idea，先快速跑 insight。
- ✅ **单仓是兜底，多仓是目标**：时间允许时，把 ≥3 个同主题仓库的内涵喂进来做跨仓合成（差异化最强）。
- ⚠️ **Git Bash 路径转换**：raw `api` 带前导 `/` 必须 `MSYS_NO_PATHCONV=1`。
- ✅ **报告以中文 Markdown 产出**。
