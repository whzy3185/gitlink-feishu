---
name: gitlink-research-recommend
version: 1.0.0
description: "科研仓库推荐：根据研究方向关键词搜索 GitLink 相关仓库，用活跃度/影响力/成熟度/科研价值评分筛选，推荐最适合研究的 Top 仓库。当科研工作者需要寻找适合研究或复现的开源项目时触发。任务四创新场景（PDF 之外）。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli search --help"
---

# gitlink-research-recommend（科研仓库推荐 · 创新科研 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读采集 + 分析，不写入仓库。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务四**创新**科研 Skill（PDF 之外的原创场景）。科研工作者面对海量开源项目，难以判断哪个适合研究/复现。本 Skill 根据**研究方向**搜索 GitLink 仓库，用 `research-insight` 五维评分筛选，**推荐最适合研究的 Top 仓库**。体现任务四创新性。

---

## 推荐模型

```
研究方向（关键词）
    ↓
search +repos（搜索候选）
    ↓
对每个候选用 research-insight 五维评分（活跃/影响/成熟/协作/科研价值）
    ↓
按科研价值排序
    ↓
推荐 Top N + 推荐理由
```

---

## 工作流

### Step 1：按研究方向搜索候选
```bash
gitlink-cli search +repos -k "<研究方向关键词>" --format json
# 如：论文复现 / 深度学习 / 算法 / 数据集 / research
```

### Step 2：对每个候选用 research-insight 评分
对每个候选仓库调用 research-insight 的五维分析（活跃度/影响力/成熟度/协作健康/科研价值），取综合科研评分。

### Step 3：排序 + 推荐
按科研价值评分排序，取 Top N（如 Top 3）。

### Step 4：输出推荐报告
```markdown
## 🎯 科研仓库推荐 — 研究方向「<关键词>」

### Top 3 推荐
| 排名 | 仓库 | 科研评分 | 推荐理由 |
|:----:|------|:-------:|---------|
| 1 | xxx/yyy | ⭐4.5 | 活跃+成熟+文档全，适合复现 |
| 2 | ... | ⭐4.0 | ... |
| 3 | ... | ⭐3.5 | ... |

### 推荐详情
#### 🥇 xxx/yyy（⭐4.5）
- 活跃度：高（N issue/pr）
- 成熟度：高（N release，README 完整）
- 科研价值：适合作为研究对象/复现基础
- 建议：fork 复现 / 引用
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| search 结果质量参差 | 用五维评分过滤低质量仓库 |
| 候选过多 | 限制 Top N（如前 10 个候选评分后取 Top 3）|
| 中文仓库名编码 | 评分时优先英文名仓库 |
| search 关键词太泛 | 让用户细化研究方向 |

---

## 实测落地参考
**研究方向**：「论文复现」（科研典型场景）

search +repos "论文复现" 候选：
- songhui18/ICCV2021论文复现（18 forks，CV 顶会论文合集）
- informatik020/高级：论文复现
- songhui18 等

**推荐**：
1. **songhui18/ICCV2021论文复现** ⭐4.0 —— 18 forks 显示社区认可，CV 顶会论文复现合集，**适合作为计算机视觉研究/复现基础**
2. 其他候选（数据少，评分较低）

> 注：中文仓库名 API 采集有编码坑，推荐时优先展示 + 引导用户网页访问。本 Skill 的推荐逻辑（搜索→评分→排序）同样适用于任意研究方向。

详见 verification.md。
