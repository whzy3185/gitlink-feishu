# 示例：评估一个科研仓库的复现性

本示例演示用 `gitlink-research-reproducibility` 技能对一个真实 GitLink 科研类仓库做复现性体检，并生成评分报告。命令均为 gitlink-cli 真实命令，可直接复现。

> 说明：下文以 `Gitlink/gitlink-cli` 作为占位演示仓库说明流程；实际使用时替换为目标科研仓库（如某课题组的 `paper-code` 仓库）。报告中的具体数值以实际采集结果为准。

---

## 一、Agent 对话片段

```
用户：帮我评估一下 someorg/paper-code 这个论文代码仓库别人能不能复现出来。

Agent：好的，我将用 gitlink-research-reproducibility 技能做六维复现性体检（环境/数据/运行/结果/文档/可维护性），全程只读、不改动仓库。开始采集……
```

---

## 二、采集命令序列（实际执行）

```bash
# 1) 元数据与技术栈
gitlink-cli repo +info      --owner someorg --repo paper-code --format json
gitlink-cli repo +languages --owner someorg --repo paper-code --format json

# 2) 文件树（六维证据主来源）
gitlink-cli repo +tree      --owner someorg --repo paper-code --format json

# 3) README
gitlink-cli repo +readme    --owner someorg --repo paper-code --format json

# 4) 数据可得性（复用 GitLink 数据集能力）
gitlink-cli dataset +view   --owner someorg --repo paper-code --format json

# 5) 结果产物 / 版本快照
gitlink-cli release +list   --owner someorg --repo paper-code --format json
gitlink-cli repo +tags      --owner someorg --repo paper-code --format json

# 6) 依赖锁定情况
gitlink-cli api GET "/someorg/paper-code/contents/requirements.txt" --format json

# 7) 可维护性
gitlink-cli repo +contributor-stats --owner someorg --repo paper-code --format json
```

---

## 三、证据归集（Agent 内部记录）

| 维度 | 检测到的证据 | 命中评分项 |
|------|--------------|-----------|
| 环境 | `requirements.txt`(已 pin)、`Dockerfile` | +8 +6 +6 |
| 数据 | README 含 Zenodo 链接；无下载脚本；无样例数据 | +8 +3 |
| 运行 | README 有完整命令块；`train.py`;超参硬编码 | +8 +6 |
| 结果 | 指标表格；`eval.py`；`set_seed()`；无 checkpoint | +5 +4 +3 |
| 文档 | README 结构完整；LICENSE(MIT)；arXiv 引用；仅中文 | +4 +3 +2 |
| 可维护 | `tests/`；`.gitea/workflows`；结构清晰；近期更新 | +4 +3 +2 +1 |

---

## 四、生成的复现性评估报告（样例输出）

```markdown
## 🔬 科研复现性评估报告

**评估对象：** someorg/paper-code
**评估时间：** 2026-06-15
**主语言：** Python　**许可证：** MIT　**最近更新：** 2026-05-30

### 一、复现性总评

| 项目 | 结果 |
|------|------|
| **复现性总分** | **74 / 100** |
| **等级** | **B** |
| **复现风险** | 🟡 中低 |
| 一句话结论 | 环境与运行说明完善，主要短板在"数据可得性"与"结果可对照性" |

### 二、六维明细

| 维度 | 得分 | 关键发现 |
|------|------|----------|
| 环境可复现性 | 20 / 25 | ✅ requirements.txt 已 pin；✅ Dockerfile；❌ 未说明 CUDA 版本 |
| 数据可得性 | 11 / 20 | ✅ README 给出 Zenodo 链接；❌ 无下载脚本；❌ 无样例数据 |
| 运行可复现性 | 14 / 20 | ✅ README 命令完整；✅ train.py 入口；❌ 超参硬编码 |
| 结果可复现性 | 12 / 15 | ✅ 指标表格；✅ eval.py；✅ 设种子；❌ 无 checkpoint |
| 文档完整性 | 9 / 10 | ✅ 结构完整；✅ LICENSE；✅ arXiv 引用；❌ 仅中文 README |
| 代码可维护性 | 10 / 10 | ✅ tests/；✅ CI；✅ 结构清晰；✅ 近期维护 |

### 三、复现性雷达

              环境
            ★★★★★★★★☆☆ 20/25
             ╲         ╱
   可维护    ╲       ╱   数据
 ★★★★★★★★★★  ╲   ╱  ★★★★★★☆☆☆☆ 11/20
   10/10      ╲ ╱
              ★
            ╱   ╲
  文档   ╱           ╲  运行
★★★★★★★★★☆ 9/10    ★★★★★★★☆☆☆ 14/20
              结果 ★★★★★★★★☆☆ 12/15

### 四、修复清单（按投入产出比排序）

| 优先级 | 行动项 | 预计提分 |
|--------|--------|----------|
| 🔴 P0 | 补 `download_data.sh` 并标注数据规模 | +5 |
| 🔴 P0 | 超参抽到 `config.yaml`，命令行可覆盖 | +4 |
| 🟠 P1 | 通过 Release 发布预训练 checkpoint | +3 |
| 🟠 P1 | 依赖说明补 CUDA / 驱动版本 | +2 |
| 🟢 P2 | 补英文 README | +1 |

**预计修复后：74 → 89（B → A，🟡 → 🟢）**
```

---

## 五、可选：把修复清单落成 Issue（需用户确认）

```bash
# 仅在用户明确同意后执行（写操作）
gitlink-cli issue +create --owner someorg --repo paper-code \
  --title "提升复现性：补充数据下载脚本与配置化超参" \
  --body "由 gitlink-research-reproducibility 评估生成，详见复现性报告 P0 项。"
```
