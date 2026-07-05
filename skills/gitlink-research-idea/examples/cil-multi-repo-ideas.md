# 💡 多仓研究 Idea 报告：Class-Incremental Learning 三仓创新对比

> 输入仓库（同主题：CIL / 类增量学习，均 GitLink 实采）：
> 1. `gaozijian19/Maintaining-Fairness-in-LKD-for-CIL` —— AAAI'25，KD 公平性（靶子，单仓 Idea 详见 [`maintaining-fairness-lkd-cil-ideas.md`](maintaining-fairness-lkd-cil-ideas.md)）
> 2. `PatZQ/AudioCIL` —— 音频 CIL 工具箱，复现 16 经典 + 3 SOTA 方法（colaudiolab 镜像）
> 3. `Kexing/class-incremental-learning` —— yaoyao-liu CIL 合集：AAN（CVPR'21）+ Mnemonics Training（CVPR'20）
>
> 模式：**多仓 A3**（跨仓语义合成）。数据来源：GitLink（gitlink-cli 实时采集）｜ 生成时间：2026-07-03。每条结论带仓库/文件溯源。

> **结论先行**：三仓恰好覆盖 CIL 三条正交技术线——**logit-KD 公平化**（靶子）、**方法库/跨模态基准**（AudioCIL）、**架构聚合 + 记忆增强**（AAN/Mnemonics），且都源于 PyCIL 骨架却各自演进。最大共性缺口是**"fairness"概念在三仓中定义互不统一**，且**无跨模态 fairness 对比**。由此推出 4 条跨仓研究 Idea。

---

## ⚠️ 范围说明：为何是 3 仓（而非更多）

经穷举搜索（FOSTER/LUCIR/CoPE/MEMO/BiC/DER/FeCAM/IL2A/PASS/SSRE/AAN 等方法名 + 增量学习/持续学习/类增量/终身学习 中文词），**GitLink 搜索为名/描述匹配、非全文**（`PyCIL`/`iCaRL`/`continual`/`catastrophic forgetting` 均 0 命中），**全平台可确证的 CIL 科研仓即这 3 个**。此"平台科研仓稀缺 + 搜索非全文"本身是子赛题四的现实约束证据，写入报告以备核查。三仓已覆盖 CIL 主流策略族，足以支撑跨仓合成。

---

## 一、创新点对比矩阵

| 仓库 | 核心 CIL 策略 | 创新点（1–3，附代码证据） | 栈 / 年代 |
|------|-------------|---------------------------|-----------|
| **靶子** Maintaining-Fairness-LKD-CIL | logit-KD **公平化** | ① **类间公平** `Zscore()`（全类 logit z-score，`dim=-1`）<br>② **类内公平** `Inverse_Zscore()`（`dim=0`）<br>③ inter+intra 联合（`--method interintra`），**即插即用、零额外训练成本** | Py3.8 / torch1.8 · AAAI'25 |
| **AudioCIL** | **方法库 / 跨模态基准** | ① 音频 CIL 首个系统基准（多场景）<br>② 复现 **16 经典 + 3 SOTA**（FineTune/Replay/EWC/LwF/iCaRL/GEM/BiC/**WA**/POD-Net/DER/Coil…，见 README "Methods Reproduced"）<br>③ 统一 PyCIL 骨架跨模态评测 | Py3.8 / torch1.8 · arXiv'24 |
| **class-incremental-learning**（yaoyao-liu） | **架构聚合 + 记忆增强** | ① **AAN** 自适应聚合多专家头（`adaptive-aggregation-networks/models`）CVPR'21<br>② **Mnemonics Training** 可学习记忆图像（`mnemonics-training/{1_train,2_eval}`）CVPR'20 | Py3.6 / torch1.2 · 2020–21 |

> **关键交叉点**：靶子的 7 基线（LwF/iCaRL/BiC/DER/PODNet/WA/Replay）**是 AudioCIL 16+ 方法库的子集**；且 AudioCIL 复现的 **WA（"Maintaining Discrimination and **Fairness** in CIL", CVPR'20）** 与靶子的"fairness"主题直接同源——但两者"公平"的落点完全不同（见缺口 G1）。

---

## 二、共性研究缺口（≥2 条）

### G1 · "Fairness" 概念碎片化，缺统一分类法 ⭐
三仓都触及"fairness"，但**定义与作用对象互不相同**，社区无统一框架：
- **靶子** = **logit 层公平**：`Zscore` 在 student/teacher 的 logit 向量上做全类归一化，消解 KD-CE 冲突（`models/lwf.py: Zscore()`）。
- **AudioCIL 复现的 WA** = **分类头权重公平**：纠正最后一层 FC 的 bias（arXiv 1911.07053），作用在权重而非 logit 蒸馏。
- **AAN** = **专家头聚合公平**：自适应聚合多个 task-specific head（`adaptive-aggregation-networks/models`），作用在架构层。
> 三者分别公平化 **logit / 权重 / head**，互不引用、不可比。证据：三仓代码中"fair/normalize/aggregation"落点各异。

### G2 · 跨模态 fairness 无统一验证
- **靶子** 支持音频（`utils/data.py` 含 Librispeech/torchaudio），但 fairness 主实验在视觉 CIFAR/ImageNet。
- **AudioCIL** 是音频基准，但**未把 fairness 作为评测维度**（只报告各方法精度）。
- **AAN/Mnemonics** 仅视觉（CIFAR/ImageNet）。
> 没有任何一仓做过"同一 fairness 机制在视觉 ↔ 音频 CIL 的统一对比"。靶子（有音频代码）+ AudioCIL（有音频基准）拼起来恰好闭环，却无人合。

### G3（补充）· 复现生态碎片化
三仓同源于 PyCIL 骨架（`convs/models/exps/utils/main.py/trainer.py` 几乎一一对应），但 **config 与方法注册接口不互通**（靶子 `exps/`、AudioCIL `exps-audio/`、AAN 自有 `models/`），跨仓方法对比需手工对齐——抬高社区复现与对比门槛。

---

## 三、跨仓研究 Idea（4 条，每条三锚）

### Idea 1：统一 Fairness 分类法 + 跨机制消融基准 ⭐⭐
- **假设**：logit-公平（靶子 Zscore）、权重-公平（WA）、head-公平（AAN）三者可纳入统一框架，并在同一基准上消融对比，揭示哪种"公平"对 CIL 收益最大、是否可叠加。
- **缺口依据**：G1。三仓 fairness 落点互不相同（`models/lwf.py: Zscore` vs AudioCIL 的 WA vs AAN 的 head 聚合）。
- **可行性**：**高**。复用 AudioCIL 的统一骨架（已含 WA/LwF/iCaRL 等 19 方法），把靶子的 Zscore 作为新 `--method` 即插即用注入，跑同一音频+视觉基准即可横向对比。靶子本就宣称"与现有 logit-KD 即插即用集成"。
- **与现有差异**：现有各做各的 fairness；本 Idea 给出**首个跨 fairness 机制的可比消融**，而非又一个并列方法。

### Idea 2：跨模态 fairness 迁移系统验证
- **假设**：logit-公平（Zscore）与模态无关（纯 logit 操作），应可从视觉迁到音频 CIL；系统量化其迁移收益与衰减。
- **缺口依据**：G2。靶子有音频代码路径却未验 fairness 迁移；AudioCIL 有音频基准却无 fairness 维度。
- **可行性**：**中高**。靶子的 `Zscore` 是模态无关函数，直接挂到 AudioCIL 的音频方法上跑 Librispeech 增量（两侧代码就绪、同 PyCIL 骨架）。
- **与现有差异**：**首个"同一 fairness 机制 视觉↔音频 CIL 迁移性"报告**，填补 G2 的空白。

### Idea 3：即插即用 fairness × 记忆增强 的正交叠加
- **假设**：靶子的 logit-KD 公平（蒸馏侧）与 Mnemonics 的可学习记忆图像（记忆/表征侧）解决的是 CIL 的两个不同瓶颈，理论上正交可叠加，组合后收益相加。
- **缺口依据**：三仓中 fairness（靶子）与 memory augmentation（yaoyao-liu/Mnemonics）是两条独立线，从未组合验证。靶子 self-claim 即插即用；Mnemonics 改 exemplar 表征。
- **可行性**：**中**。两者代码层基本不冲突（一个改 KD loss、一个改 buffer 表征），同 PyCIL 骨架可拼；需跑组合实验证正交性。
- **与现有差异**：验证"蒸馏公平 × 记忆增强"的**正交叠加收益**，连接两条原本孤立的研究线。

### Idea 4：统一 PyCIL 配置互操作层（工程基建型）
- **假设**：一个 config 适配 + 方法注册表能让三仓（及未来 CIL 仓）方法互通互比，显著降低社区复现门槛。
- **缺口依据**：G3。同骨架但 `exps/` vs `exps-audio/` vs AAN 自有结构不互通。
- **可行性**：**中**（工程量大但价值高）。做 method registry + config schema 统一，对标 MMClassification 之于分类。
- **与现有差异**：不是新方法，是**生态基建**，直接服务 G3 的复现碎片化。

---

## 四、优先级建议

| 优先级 | Idea | 理由 |
|--------|------|------|
| 🥇 | **Idea 1（统一 fairness 消融）** | 两仓代码就绪、即插即用、新颖性高（跨机制对比前所未有），最快出强结论 |
| 🥈 | **Idea 2（跨模态迁移）** | 复用 Idea 1 的注入管线、填补 G2 空白、故事好讲 |
| 🥉 | **Idea 3（正交叠加）** | 连接 fairness + memory 两条线，理论贡献清晰，实验量略大 |
| 视资源 | Idea 4（互操作层） | 工程型，适合作为开源基建延伸 |

> 一句话：**用 AudioCIL 当统一试验台，把靶子的 Zscore fairness 注入进去（Idea 1），再跨到音频（Idea 2）**——以最小代价同时咬住 G1+G2 两个共性缺口。

---

## 五、差异化自证（为何现有 skill 做不到）

| 本步产出 | `gitlink-research-tracker` | `gitlink-scholar-profile` | `gitlink-research-fork-impact` |
|----------|---------------------------|--------------------------|-------------------------------|
| 创新点语义对比 + 缺口合成 + 假设构造 | ✗ 跨仓搜索 + 成熟度/热度**计数** | ✗ 学者画像，不分析方法 | ✗ 传播力计数，不碰研究内涵 |
| 读代码对齐"fairness 落点"、长出新研究问题 | 结构上无法产出（不读方法实现） | 同左 | 同左 |

> 本步是**语义级**合成（读懂每个仓的方法机制 → 对齐 → 找空白 → 提假设），计数/评分类 skill 在结构上无法替代。

---

## 附：本报告采集命令（可复现）

```bash
# 仓 1（靶子）—— 详见 gitlink-research-insight；关键方法源码：
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/gaozijian19/Maintaining-Fairness-in-LKD-for-CIL/sub_entries" --query "filepath=models/lwf.py&ref=master" --format json

# 仓 2（AudioCIL，默认分支 main）
gitlink-cli repo +info --owner PatZQ --repo AudioCIL --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/PatZQ/AudioCIL/sub_entries" --query "filepath=README.md&ref=main" --format json   # Methods Reproduced 列表

# 仓 3（class-incremental-learning，默认分支 main）
gitlink-cli repo +info --owner Kexing --repo class-incremental-learning --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/Kexing/class-incremental-learning/sub_entries" --query "filepath=&ref=main" --format json   # 根目录：adaptive-aggregation-networks/ + mnemonics-training/

# 同主题仓发现（已穷举，确认平台稀缺）
gitlink-cli search +repos -k "class-incremental" --limit 30 --format json   # 仅 AudioCIL 命中科研仓
```
