---
name: gitlink-research-insight
version: 0.1.0
description: "仓库级科研项目洞悉：深度解读一个科研/论文代码仓库的内涵——研究问题、方法、核心创新、对比基线、数据集、复现性，生成中文「科研内涵解读报告」。当用户想快速搞懂一个陌生科研代码仓库、做科研复盘、或为选题找参考时触发。常见表述：「这个仓库做了什么」「解读一下这个论文代码」「这个科研项目讲了啥」。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-insight（仓库级科研项目洞悉）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

## 何时使用 / 何时跳过

**使用本 skill（单仓深度内容解读）：**
- 用户说"这个仓库做了什么""解读这个论文代码""这个科研项目讲了啥"
- 用户拿到一个陌生科研/论文代码仓库，想 30 分钟内读懂其研究内涵

**跳过本 skill（改用其他 skill）：**
- 想看某学者/团队跨仓画像 → `gitlink-scholar-profile`
- 想做某领域技术调研/热点追踪（跨仓） → `gitlink-research-tracker`
- 想看某仓库被 fork 后的传播 → `gitlink-research-fork-impact`
- 只查许可证/依赖合规 → `gitlink-compliance` / `gitlink-license-compliance`

> **与上述 skill 的本质区别**：它们在**跨仓**粒度做指标计数/评分；本 skill 在**单仓**粒度做**内容级研究内涵理解**。

## ⚠️ 现实约束（驱动设计的前提）

GitLink 上的科研仓库**绝大多数是"成品上传"**——论文发表后整体上传，commit 少（常见 1–5 个）、**不含真实研究探索过程**（实测靶子仓库仅 3 个 commit 且全部在同一天）。

→ 因此本 skill **基于"内容"（README + 代码结构 + 实验配置 + 依赖）做解读，而非基于 commit/分支演进**。不要试图从 commit 历史提炼"技术演进脉络"——那在 GitLink 科研仓库上基本无数据。

## 数据采集（只用这些 gitlink-cli 命令）

```bash
# 0) 前置：认证
gitlink-cli auth status

# 1) 仓库元信息（名称/描述/大小/默认分支/mirror 标志/各 count）
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 2) README 全文 —— 【主要信息源】，Abstract/方法/数据集/运行命令/作者多在此
gitlink-cli repo +readme --owner <owner> --repo <repo> --format json
#   返回 data.content（base64）或 data.replace_content（已解码，图片地址已改绝对路径）

# 3) 文件结构 —— 暴露方法实现位置、基线矩阵、模块划分
gitlink-cli repo +tree --owner <owner> --repo <repo> --format json
#   逐层展开子目录：对 tree 结果中的 type=dir，可用 api 递归列出

# 4) 语言分布 / 贡献者 / 代码统计
gitlink-cli repo +languages  --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
gitlink-cli repo +code-stats --owner <owner> --repo <repo> --format json
```

> **镜像仓库注意**：`repo +info` 的 `contributor_users_count` 对 GitHub 镜像恒为 0（无 GitLink 注册用户）；要看真实贡献者，必须用 `repo +contributors`（按 commit 邮箱归因）。

> **读取任意文件（如 requirements.txt / exps/*.json / main.py）**：用 GitLink 的 `sub_entries` 端点（**不是 `/contents/`**——该路径在 GitLink 不存在，会 404 返回 SPA HTML）。正文在返回的 `data.entries.content`（已解码纯文本）。**Git Bash 下必须加 `MSYS_NO_PATHCONV=1`**（否则前导 `/` 被转成 `A:/...` 导致 404）：
> ```bash
> MSYS_NO_PATHCONV=1 gitlink-cli api GET "/<owner>/<repo>/sub_entries" --query "filepath=<path>&ref=master" --format json
> ```
> （`/raw/` 端点对 API token 常返回 403，勿用。所有现有 skill 统一用 `sub_entries`。）

## 解读流程

### Step 1：采集（执行上述命令，--format json）

### Step 2：识别"科研信号"
从 README + tree 判断这是否为科研仓库及方向：
- README 是否含 Abstract / 论文引用 / arXiv / 顶会名（AAAI/NeurIPS/ICML…）
- tree 是否含典型科研结构：`exps/`|`configs/`（实验配置）、`models/`|`convs/`（模型实现）、`data/`|`utils/data.py`（数据集）、`requirements.txt`、训练入口（`main.py`/`train.py`）
- 是否声明基于某已有框架（如 "builds upon PyCIL"）

### Step 3：提炼六维（AI 综合 README + tree + stats）

| 维度 | 提炼来源 | 要点 |
|------|----------|------|
| 研究问题与动机 | README Abstract | 解决什么问题、为何重要 |
| 方法与核心创新 | README 方法段 + tree 模型文件 | 核心机制、创新点（每条尽量映射到代码位置） |
| 对比基线与实验矩阵 | `exps/`/`configs/` 文件名 + README | 作为对比/集成的基线方法清单 |
| 数据集与运行入口 | README + `utils/data.py` + 入口脚本 | 支持的数据集、训练/推理命令 |
| 复现性速览 | `requirements.txt` + README 运行说明 + 预训练权重链接 | 环境是否锁定、数据是否可得、入口是否清晰 |
| 仓库画像 | repo +info / +languages / +contributors / +code-stats | 规模、语言、贡献者、是否成品上传 |

### Step 4：生成中文「科研内涵解读报告」（按下述模板，Write 工具产出 Markdown）

## 输出模板：科研内涵解读报告

```markdown
# 🔬 科研内涵解读报告：<仓库标题/论文名>

> 解读对象：<owner>/<repo>  ｜  数据来源：GitLink（gitlink-cli 实时采集）  ｜  解读时间：{{date}}

## 一、仓库画像
| 项 | 内容 |
|----|------|
| 仓库 | owner/repo（如为镜像，标注 mirror + 原址） |
| 论文/会议 | （从 README 引用提取，如 AAAI'25） |
| 规模 / 语言 | size ｜ Python xx% / Shell xx% |
| 贡献者 | n 人（主力：xxx，占比） |
| 上传形态 | 成品上传 / 持续演进（commit 数与时间跨度判断） |

## 二、研究问题与动机
（一段话：解决什么问题、为什么重要、现有方法不足）

## 三、方法与核心创新
1. **创新点一**：…（→ 代码位置：<file>）
2. **创新点二**：…（→ 代码位置：<file>）
（每条创新尽量标注对应代码文件/目录）

## 四、对比基线与实验矩阵
（从 exps/configs 提取基线清单；说明本方法与它们的关系：对比 / 即插即用集成 / 扩展）

## 五、数据集与运行入口
- 支持数据集：…
- 运行命令：（从 README 提取真实命令）
- 数据/权重获取方式：…

## 六、复现性速览 + 一句话评价
| 维度 | 状态 | 依据 |
|------|------|------|
| 环境锁定 | ✅/⚠️/❌ | requirements.txt 是否全 pin |
| 数据可得 | ✅/⚠️/❌ | 是否需手填路径/公开下载 |
| 运行入口 | ✅/⚠️/❌ | 是否有明确 main.py + config |
| 预训练权重 | ✅/⚠️/❌ | 是否提供下载链接 |

**一句话评价**：…（这个仓库是什么、强在哪、适合谁）
```

> **溯源要求**：报告中每条结论尽量标注来源（README 某段 / 某文件 / 某命令输出），可核查。

## 示例

完整范例见 [`examples/maintaining-fairness-lkd-cil-insight.md`](examples/maintaining-fairness-lkd-cil-insight.md)（靶子仓库 `gaozijian19/Maintaining-Fairness-in-LKD-for-CIL` 的真实解读）。

## 模式二：方法—代码实现剖析（A2）

> 在 A1 内涵解读之上，回答"论文方法如何落到代码、想改某模块该看哪个文件"。**不依赖 commit 历史**，纯从代码结构重建。当用户说"这个方法的代码在哪""论文公式对应哪段代码""我想改 X 该看哪个文件"时进入本模式。

### A2 流程

1. **复用 A1 的采集**（`+info`/`+readme`/`+tree`），再用 `sub_entries` 精读关键源码：入口（`main.py`/`train.py`）、训练循环（`trainer.py`/`engine.py`）、模型/方法实现（`models/*.py`）、网络结构（`utils/inc_net.py`、`convs/`）、配置（`exps/*.json`）。
2. **建立"论文概念 ↔ 代码位置"映射**：从 README 的方法/公式描述出发，在源码里定位其实现（函数名/分支/类），每条标注 `文件:函数/行`。
3. **画架构 Mermaid 图**：数据流 = 入口 → 训练编排 → 数据加载 → 模型前向 → 损失计算 → 反传；标注关键分叉（如 `--method`/`--mode` 一类开关如何在代码里切换算法变体）。
4. **补"想改 X 看哪里"快速定位表**（实战价值，帮接手者秒级定位）。

### A2 输出模板

```markdown
# 🧬 方法—代码实现剖析：<仓库/论文名>
## 一、架构总览（Mermaid）   ← 数据流图，标注算法开关分叉
## 二、方法—代码映射表        ← ≥5 行：论文概念 / 对应文件:位置 / 一句话作用
## 三、核心机制代码定位       ← 论文关键创新点的确切代码段（引用真实片段）
## 四、想改 X，看哪里         ← 快速定位表
```

### A2 示例

完整范例见 [`examples/maintaining-fairness-lkd-cil-anatomy.md`](examples/maintaining-fairness-lkd-cil-anatomy.md)（靶子仓库真实剖析：把论文 inter/intra-class fairness 两个创新点精确定位到 `Zscore()`/`Inverse_Zscore()` 两个函数 + `--method interintra` 分支）。

---

## 后续扩展：A3 跨多仓创新点对比 + 研究 Idea 生成

对多个同主题仓库各跑本 skill 的内涵提取（A1 的"方法与核心创新"段即 A3 的输入单元），再用 gap-analysis/lead-discovery 跨仓合成"创新点对比矩阵 + 共性缺口 + 潜在研究 Idea"。该能力由配套 skill `gitlink-research-idea` 承载（单仓模式 = 基于 A1 内涵生成"局限 + 可拓展 Idea"，作为多仓的降级兜底）。

## 注意事项

- ✅ **README 是主信息源**：GitLink 科研仓库的 Abstract/方法/数据/命令几乎都在 README，优先深读 README。
- ✅ **成品上传不丢分**：commit 少不影响解读，靠"内容"而非"历史"。
- ⚠️ **镜像仓的 count 不可信**：`contributor_users_count`/`issues_count` 等对镜像常为 0，用 `+contributors` 等带归因的命令。
- ⚠️ **Git Bash 路径转换**：raw `api` 命令带前导 `/` 的路径参数会被 MSYS 转成 Windows 路径，必须 `MSYS_NO_PATHCONV=1`。
- ✅ **报告以中文 Markdown 产出**。
