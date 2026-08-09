# 💡 研究 Idea 报告：Maintaining Fairness in LKD for CIL

> 基于：`gaozijian19/Maintaining-Fairness-in-LKD-for-CIL` 的内涵解读（见 [`../gitlink-research-insight/examples/maintaining-fairness-lkd-cil-insight.md`](../../gitlink-research-insight/examples/maintaining-fairness-lkd-cil-insight.md)）｜ 数据来源：GitLink（gitlink-cli 实时采集）｜ 生成时间：2026-07-03
> 模式：**单仓 A3'**（多仓 A3 的降级兜底）。
> 本报告由 `gitlink-research-idea` skill 流程产出，每条 Idea 均带代码证据，可核查。

> **结论先行**：该方法（logit 全类归一化的 inter/intra-class fairness）工程上极简洁优雅，但其"公平化"**只作用在最终 logit 层、统计量是 batch 内瞬时值、inter/intra 权重全程静态**——这三点正是最值得追问的方向。下面给出 5 条由代码证据直接支撑的研究 Idea，按"可行性 × 预期增益"排序。

---

## 一、方法局限剖析（Idea 的土壤）

| # | 局限 | 证据（代码 : 位置） | 破绽类型 |
|---|------|---------------------|----------|
| L1 | 公平化归一化**只用 batch 内瞬时统计**（mean/std），小 batch 或类极不均衡时统计量噪声大 | `models/lwf.py` → `Zscore()` 用 `logits.std(dim=-1)`、`Inverse_Zscore()` 用 `dim=0`，均无 running/EMA | 统计依赖 |
| L2 | 公平化**只在最终 logit 层**作用，中间 feature 表示的 recency bias 未触及 | `Zscore` 仅作用于 `_network(inputs)["logits"]`；`convs/` backbone 输出特征未经公平化 | 作用层级 |
| L3 | inter/intra 权重 `alpha/beta` **全程静态**，由首帧 `--lambd` 一次性算定，不随任务/类比例变化 | `models/lwf.py` → `LwF.__init__`：`alpha=2*lambd/(1+lambd)`、`beta=2/(1+lambd)` 一次定死 | 静态 vs 动态 |
| L4 | fairness 框架**只在 logit-KD 验证**，同仓库已有的 feature-KD（`feature`/`rkd` 分支）未纳入统一公平性 | `_update_representation` 里 `feature`/`passfeature`/`rkd` 分支用 `get_features()`，未与 Zscore 结合 | 范畴边界 |
| L5 | 跨模态（语音）**支持但未充分验证公平性迁移** | `utils/data.py` 含 Librispeech/torchaudio/torchlibrosa，但论文公平性结论主要在视觉 CIL | 范畴边界 |

---

## 二、研究 Idea（5 条，每条三锚）

### Idea 1：Feature-level Fairness —— 把公平化从 logit 层下沉到表示层 ⭐
- **假设**：recency bias 不只存在于分类头 logit，也存在于 backbone 输出特征；把 inter/intra fairness 推广到 feature 空间能进一步缓解新旧类偏置。
- **缺口依据**：`models/lwf.py` 中 `Zscore`/`Inverse_Zscore` 仅作用于 `["logits"]`；而同文件 `feature`/`rkd` 分支已通过 `self._network.get_features(inputs)` 取到特征（L2、L4）——说明接口现成、公平化却未覆盖该层。
- **可行性**：**高**。复用 `get_features()`，对特征做 `Zscore` 后接余弦/MSE 蒸馏；改动局限在 `_update_representation` 新增一个分支，~30 行，复用现有 trainer 与数据管线。
- **与现有差异**：现有 fairness 是"输出层"的；本 Idea 是"表示层"的——更深的公平化，且可与 logit-level fairness 叠加（正交贡献）。

### Idea 2：Adaptive Inter/Intra Weighting —— 静态 `--lambd` 改为任务进度/类比例自适应 ⭐
- **假设**：旧类占比随任务推进单调上升，固定 `alpha/beta` 全程不变是次优；让权重随 `_known_classes/_total_classes` 动态调整可提升后期任务表现。
- **缺口依据**：`models/lwf.py` → `LwF.__init__` 中 `alpha/beta` 由 `lambd` 一次性算定，`_update_representation` 全程复用，未随 `_cur_task`/类比例变化（L3）。
- **可行性**：**中高**。把 `alpha/beta` 的计算从 `__init__` 移进 `_update_representation`，设为 `_known_classes/_total_classes` 的函数（或加一个轻量学习率式 schedule）；~20 行，无新依赖。
- **与现有差异**：现有是"静态超参"；本 Idea 是"动态课程式权重"——把 inter/intra 偏好与增量阶段耦合，更贴合 CIL 的非平稳性。

### Idea 3：Batch-Robust Normalization —— 用 running/EMA 统计量替代 batch 瞬时值
- **假设**：`Zscore` 的 batch 内 `std` 在小 batch（如 `batch_size=128` 切到长尾类时实际更小）或不均衡时噪声大，拖累蒸馏；改用跨 batch 的 running mean/var 能稳定公平化。
- **缺口依据**：`models/lwf.py` → `Zscore()` / `Inverse_Zscore()` 直接用 `logits.mean/std(dim=±1)`，无 momentum/running 机制（L1）。
- **可行性**：**高**。仿 BatchNorm 维护 running stats（`register_buffer` + momentum 更新），改动集中在两个函数；训练循环不动。
- **与现有差异**：现有是"per-batch 瞬时统计"；本 Idea 是"跨 batch 稳定估计"——直接对标 BN 的成熟经验，易写易消融。

### Idea 4：Unified Fairness for Feature-KD —— 把 fairness 框架统一到 RKD/feature 蒸馏
- **假设**：logit-KD 的公平化思想（消除 teacher 的过度自信偏置）在 feature-KD 中同样成立；统一两者能得到一个更通用的"fair distillation"框架。
- **缺口依据**：`_update_representation` 中 `rkd` 分支（`RkdDistance`/`RKdAngle`）与 `interintra` 分支**互斥并列**、共享同一 `--method` 开关却无交叉（L4）。
- **可行性**：**中**。需在特征距离/角度空间定义"类间/类内公平"（非平凡），但代码已有 `pdist`、`get_features` 等积木可拼；可作为一篇方法统一的 follow-up。
- **与现有差异**：现有 fairness 绑定 logit-KD；本 Idea 给出 logit 与 feature 两族 KD 的统一公平性视角。

### Idea 5：Cross-Modal Fairness Transfer —— 系统验证 inter/intra fairness 在语音/多模态 CIL 的迁移
- **假设**：公平化机制（logit 归一化）与模态无关，应可迁到 audio CIL；验证其在语音类增量下的收益与差异。
- **缺口依据**：`utils/data.py` 已实现 Librispeech100（`torchaudio`/`torchlibrosa` 的 `Spectrogram`/`LogmelFilterBank`），代码就绪但论文公平性主实验在视觉（L5）。
- **可行性**：**中**。需跑 audio 增量实验（数据/算力成本），但代码路径已通；偏实证型，方法改动小。
- **与现有差异**：现有结论限视觉 CIL；本 Idea 把 fairness 的适用边界扩到跨模态，是"验证 + 迁移"型贡献。

---

## 三、优先级建议（可行性 × 预期增益）

| 优先级 | Idea | 理由 |
|--------|------|------|
| 🥇 先做 | **Idea 3（稳健归一化）** | 改动最小（两函数）、对标 BN 经验、消融干净，最快出结论 |
| 🥈 次做 | **Idea 1（feature-level fairness）** | 接口现成、与现有 logit-level 正交可叠加、新颖性高 |
| 🥉 再做 | **Idea 2（自适应权重）** | 直击 CIL 非平稳性，故事好讲，工作量略高于前两者 |
| 视资源 | Idea 4 / 5 | 方法统一型 / 跨模态验证型，工作量大，适合作为延伸课题 |

> 一句话：**先做 Idea 3 锁一个"稳定可复现的改进"，再用 Idea 1 拿"表示层公平化"的新颖性**——这两条都以最小改动切入该仓库最明显的两个破绽（batch 瞬时统计、仅 logit 层）。

---

## 附：本报告采集命令（可复现）

```bash
OWNER=gaozijian19; REPO=Maintaining-Fairness-in-LKD-for-CIL
# 内涵解读（A1）见 gitlink-research-insight；本报告在其上精读方法实现：
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=models/lwf.py&ref=master" --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=utils/data.py&ref=master"  --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=exps/lwf.json&ref=master"   --format json
```
