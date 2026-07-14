# 真实仓库验证：`opensci/pDeep` 复现性评估

> 本文件是 `gitlink-research-reproducibility` 技能在**真实 GitLink 科研仓库**上的端到端验证记录。
> 目标仓库：[`opensci/pDeep`](https://www.gitlink.org.cn/opensci/pDeep)（pDeep：基于深度学习的肽段 MS/MS 谱图预测系统，质谱蛋白质组学领域）。
> 采集时间：2026-06-15，全程只读，未对仓库做任何写操作。
>
> **重要声明：** 本评估针对「仓库内随附的复现性材料完备度」，**并非**对该科研工作学术价值的评价；pDeep 是真实发表的科研工具。该 GitLink 仓库为镜像，部分完整资料可能存在于上游。受**只读 API 无法递归遍历子目录**的限制，环境/运行维度按「可见证据的保守下限」评分，真实情况可能更高，建议 clone 后复核。

---

## 一、采集命令（技能驱动的真实只读调用）

```bash
gitlink-cli repo +info     --owner opensci --repo pDeep --format json
gitlink-cli repo +tree     --owner opensci --repo pDeep --format json   # 根目录文件树
gitlink-cli repo +readme   --owner opensci --repo pDeep --format json
gitlink-cli repo +languages --owner opensci --repo pDeep --format json
gitlink-cli dataset +view  --owner opensci --repo pDeep --format json
gitlink-cli release +list  --owner opensci --repo pDeep --format json
gitlink-cli repo +tags     --owner opensci --repo pDeep --format json
```

## 二、采集到的真实证据

**根目录文件树（`repo +tree`）：**

```
dir  | batch_pLabel
dir  | model
dir  | pDeep2
file | AUTHORS
file | CHANGELOG
file | LICENSE
file | README.md
```

**README.md 全文（`repo +readme`，仅 180 字节）：**

```
# pDeep
Predicting MS/MS Spectra of Peptides with Deep Learning

Please visit https://github.com/pFindStudio/pDeep/tree/master/pDeep2 for the improved version of pDeep --- pDeep2.
```

**其他事实：**
- ✅ 含 `LICENSE`、`AUTHORS`、`CHANGELOG`、`model/`（疑似预训练模型目录）。
- ❌ `dataset +view`：未发现绑定的 GitLink 科研数据集。
- ❌ `release +list` / `repo +tags`：无发行版/版本快照。
- ⚠️ 根目录未见任何依赖清单（requirements.txt / environment.yml / Dockerfile）；README 未含安装、运行、数据获取说明，并将"改进版"指向站外 GitHub。
- ⚠️ 受只读 API 限制，`pDeep2/`、`model/`、`batch_pLabel/` 子目录未能遍历，深层可能存在依赖/脚本文件。

## 三、按六维模型评分（保守下限）

| 维度 | 得分 | 评分依据（仅基于可见证据） |
|------|------|---------------------------|
| 环境可复现性 /25 | **3** | 根目录无依赖清单/容器文件；README 未声明环境要求。⚠️ 子目录未遍历，可能低估 |
| 数据可得性 /20 | **2** | 无数据集绑定、无下载脚本、无样例数据；README 未给数据获取方式 |
| 运行可复现性 /20 | **2** | README 无任何可复制运行命令；根目录无明确入口脚本；指向站外 |
| 结果可复现性 /15 | **4** | `model/` 疑似含预训练模型(+3)；无预期结果表、无评估脚本、未提随机种子 |
| 文档完整性 /10 | **5** | ✅ LICENSE(+3)；✅ 英文 README(+1)；✅ AUTHORS/CHANGELOG(+1)；结构仅含简介，未覆盖安装/使用/结果 |
| 代码可维护性 /10 | **3** | 代码按 `pDeep2`/`batch_pLabel`/`model` 分目录(+2)；根目录无 CI 配置；近期维护状态未知(+1 结构) |

**复现性总分 = 3 + 2 + 2 + 4 + 5 + 3 = 19 / 100**

## 四、生成报告（样例输出）

```markdown
## 🔬 科研复现性评估报告

**评估对象：** opensci/pDeep（肽段 MS/MS 谱图预测 · 深度学习）
**评估时间：** 2026-06-15
**数据来源：** GitLink 平台（gitlink-cli 只读采集）

### 一、复现性总评
| 项目 | 结果 |
|------|------|
| **复现性总分（保守下限）** | **19 / 100** |
| **等级** | **D** |
| **复现风险** | 🔴 高 |
| 一句话结论 | 仓库随附复现材料严重不足：缺依赖说明、缺数据获取、缺运行命令；当前难以被他人独立复现 |

### 二、六维雷达
              环境
            ★★☆☆☆☆☆☆☆☆ 3/25
             ╲         ╱
   可维护    ╲       ╱   数据
 ★★★☆☆☆☆☆☆☆  ╲   ╱  ★☆☆☆☆☆☆☆☆☆ 2/20
   3/10       ╲ ╱
              ★
            ╱   ╲
  文档   ╱           ╲  运行
★★★★★☆☆☆☆☆ 5/10    ★☆☆☆☆☆☆☆☆☆ 2/20
              结果 ★★★☆☆☆☆☆☆☆ 4/15

### 三、修复清单（按投入产出比排序）
| 优先级 | 行动项 | 预计提分 |
|--------|--------|----------|
| 🔴 P0 | 增补 `requirements.txt`（pin 版本）或 `environment.yml`，写明 Python/框架版本 | +8 |
| 🔴 P0 | README 补「安装 → 训练/预测 → 评估」可复制命令块与明确入口脚本 | +10 |
| 🔴 P0 | 说明训练/测试数据来源与获取方式（链接或下载脚本），或绑定 GitLink 数据集 | +8 |
| 🟠 P1 | 通过 Release 正式发布预训练模型，并附预期指标/示例输出 | +5 |
| 🟠 P1 | README 补论文引用（BibTeX/DOI），将站外资料同步进仓库或 Wiki | +3 |
| 🟢 P2 | 增补最小可跑的样例数据与冒烟测试脚本 | +3 |

**预计修复后：19 → 56+（D → C/B，🔴 → 🟠）**

> 注：受只读 API 限制，子目录未遍历，环境/运行维度为保守下限；若 `pDeep2/` 内已含依赖与脚本，请据实上调，并把这些信息「前置」到根 README，让复现者无需翻找。
```

## 五、验证结论

- 技能在真实科研仓库上**端到端跑通**：从只读采集 → 六维证据归集 → 量化评分 → 雷达图 → 分优先级修复清单，全链路可复现。
- 该案例恰好印证了技能的价值场景：**有价值的科研工具，因仓库随附复现材料不足而难以被他人复现**——这正是「可复现性危机」的典型表现，也是本技能要量化并推动改善的问题。
- 评估保持**证据可追溯、限制透明**：每一分都标注了依据，并显式说明只读 API 的遍历限制与"保守下限"口径，避免对真实仓库的误判。
