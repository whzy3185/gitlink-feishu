# 示例：为科研仓库匹配协作者

本示例演示用 `gitlink-research-match` 技能，为一个缺人手的科研仓库匹配协作者并生成推荐方案。命令均为 gitlink-cli 真实命令，可复现。

> 实际使用时替换为目标仓库与真实关键词；候选与数值以实际采集为准。

---

## 一、Agent 对话片段

```
用户：我们的 someorg/medseg 项目有几个 CUDA 加速和文档的 Issue 一直没人接，
      能在 GitLink 上帮我找找合适的协作者吗？

Agent：好的，我用 gitlink-research-match 技能：先画"需求画像"，再检索候选并打分，
       最后给推荐方案和外联草稿（不会替你发出邀请）。开始……
```

---

## 二、需求侧采集

```bash
gitlink-cli repo +info       --owner someorg --repo medseg --format json
gitlink-cli repo +languages  --owner someorg --repo medseg --format json
gitlink-cli repo +readme     --owner someorg --repo medseg --format json
gitlink-cli issue +list      --owner someorg --repo medseg --state open --format json
gitlink-cli repo +contributors --owner someorg --repo medseg --format json
```

**需求画像结果：**
- 领域：医学影像分割（CV / 深度学习）
- 技术栈：Python / PyTorch
- 紧缺能力：CUDA 性能优化、英文文档
- 可切入 Issue：#42（help wanted, CUDA）、#51（good first issue, docs）

---

## 三、候选侧检索与画像

```bash
gitlink-cli search +repos --keyword "CUDA 加速" --format json
gitlink-cli search +users --keyword "medical image" --format json

# 对检索到的候选逐个画像
gitlink-cli user +info --login cuda_master --format json
gitlink-cli repo +list --user cuda_master  --format json
```

**候选打分（按第三节规则）：**

| 候选 | 方向 | 技术 | 能力证据 | 活跃 | 协作 | 匹配分 |
|------|------|------|----------|------|------|--------|
| @cuda_master | 90 | 95 | 95 | 100 | 80 | **88** ⭐⭐⭐ |
| @docs-helper | 70 | 70 | 80 | 80 | 60 | **72** ⭐⭐ |
| @random_user | 40 | 30 | 0 | 20 | 30 | 28 — 不推荐 |

---

## 四、生成的推荐方案（样例输出）

```markdown
## 🤝 科研协作推荐方案

**目标仓库：** someorg/medseg
**生成时间：** 2026-06-15

### 一、需求画像
| 项目 | 内容 |
|------|------|
| 研究领域 | 医学影像分割（CV / 深度学习） |
| 主技术栈 | Python / PyTorch |
| 紧缺能力 | CUDA 性能优化、英文文档 |
| 可切入 Issue | #42（CUDA）、#51（docs） |

### 二、推荐协作者
#### ⭐⭐⭐ 强烈推荐：@cuda_master（88）
- 补齐最紧缺的 CUDA 优化能力；代表作 `fast-seg-cuda`（⭐120）。
- 建议切入：Issue #42。

#### ⭐⭐ 推荐：@docs-helper（72）
- 擅长英文文档站；建议从 #51（good first issue）切入。

### 三、外联草稿（请用户亲自发出）
> 致 @cuda_master：你好！我们维护 someorg/medseg（医学影像分割），
> 注意到你的 fast-seg-cuda 在 CUDA 加速上很出色，我们有推理加速需求（#42），
> 不知是否有兴趣参与或交流？期待回复，谢谢！

### 四、对接建议
1. 先以 good-first 子任务轻量试合作。
2. 邀请前自查仓库复现性（可联动 gitlink-research-reproducibility）。
3. 在 #42 上 @ 对方并说明上下文。
```

---

## 五、合规说明

- 全程只读；推荐方案与外联草稿均由用户审阅后亲自发出。
- 候选数据仅来自公开仓库，禁止用于批量骚扰式触达。
