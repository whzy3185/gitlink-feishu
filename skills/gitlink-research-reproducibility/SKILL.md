---
name: gitlink-research-reproducibility
version: 1.0.0
description: "科研复现性评估：扫描科研代码仓库的环境依赖、数据可得性、运行说明、结果产物、文档与代码质量，生成可量化的《复现性评分报告》（0-100 分 + A/B/C/D 等级 + 风险等级 + 修复清单）。当用户需要评估科研项目能否被他人复现、排查复现障碍、准备论文配套代码开源时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-reproducibility（科研复现性评估）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 本技能为只读分析型，不修改任何仓库内容；如需创建修复 Issue，须先确认用户意图。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 功能概述

本技能回答科研场景的一个核心问题：**「这个科研仓库，别人能照着把它跑出来吗？」**

学术界长期存在「可复现性危机」——论文发表了，配套代码仓库却因为缺依赖说明、缺数据、缺运行脚本、缺随机种子而无法复现。本技能通过组合 gitlink-cli 的只读命令，对一个科研仓库进行**六维复现性体检**，输出**可量化的复现性评分（0–100）**、等级（A/B/C/D）、复现风险等级，以及**按优先级排序的修复清单**。

### 六维复现性评估模型

| 维度 | 权重 | 核心问题 |
|------|------|----------|
| 1. 环境可复现性（Environment） | 25 | 依赖与运行环境是否被明确锁定？ |
| 2. 数据可得性（Data） | 20 | 实验所需数据能否获取？ |
| 3. 运行可复现性（Execution） | 20 | 是否有清晰的运行入口与步骤？ |
| 4. 结果可复现性（Result） | 15 | 能否验证跑出的结果与论文一致？ |
| 5. 文档完整性（Documentation） | 10 | README/引用/许可是否齐备？ |
| 6. 代码可维护性（Maintainability） | 10 | 是否有测试/CI/合理结构？ |

> **总分 = 各维度得分加权求和（满分 100）。** 评分规则见第三节。

---

## 一、确定评估目标并采集元数据

### 1.1 解析 owner/repo

```bash
# 在科研仓库目录下，自动解析 owner/repo
gitlink-cli repo +info --format json

# 或显式指定
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

从返回中提取并记录：`name`、`description`、`language`、`license`、`default_branch`、
`updated_at`、`created_at`、`forked_from_project_id`（用于排除纯 fork）。

### 1.2 采集主语言与发行物

```bash
# 语言分布（判断技术栈，决定后续要找哪些依赖清单）
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json

# 发行版（是否有打 tag / checkpoint / 结果产物）
gitlink-cli release +list --owner <owner> --repo <repo> --format json

# 标签（是否有版本快照，复现需要可定位的版本）
gitlink-cli repo +tags --owner <owner> --repo <repo> --format json
```

---

## 二、六维证据采集

### 2.1 维度一 · 环境可复现性（Environment，25 分）

**目标：判断依赖与运行环境是否被明确锁定。**

```bash
# 列出仓库根目录文件树，识别依赖清单与容器文件
gitlink-cli repo +tree --owner <owner> --repo <repo> --format json
```

在文件树中检测以下**依赖/环境证据**（按语言）：

```
Python：     requirements.txt / environment.yml / Pipfile / pyproject.toml / setup.py / poetry.lock
Conda：      environment.yml / conda.yaml
容器：        Dockerfile / docker-compose.yml / .devcontainer/
Go：         go.mod / go.sum
Node：        package.json / package-lock.json / yarn.lock
Java：        pom.xml / build.gradle
R：          renv.lock / DESCRIPTION
Julia：       Project.toml / Manifest.toml
通用环境：     Makefile / .python-version / runtime.txt
```

对找到的关键清单，进一步检查**是否锁定版本**：

```bash
# 读取依赖清单内容，判断是否 pin 版本（如 numpy==1.24.3 vs numpy）
gitlink-cli api GET /:owner/:repo/raw/requirements.txt --format json
# 或通过 contents 接口
gitlink-cli api GET "/:owner/:repo/contents/requirements.txt" --format json
```

> ⚠️ `:owner` / `:repo` 为占位符，调用时替换为真实值。若 raw 路径取不到，回退用 `repo +files` 或 contents 接口。

**评分细则（满分 25）：**

```
+8  存在至少一个依赖清单（requirements/environment/go.mod/package.json…）
+6  依赖清单锁定了具体版本（== / 锁文件存在）
+6  存在容器化定义（Dockerfile / docker-compose / devcontainer）
+3  存在一键环境搭建脚本或 Makefile install 目标
+2  README 明确写出语言/框架/硬件（GPU/CUDA）版本要求
（无任何依赖清单 → 本维度 0 分）
```

### 2.2 维度二 · 数据可得性（Data，20 分）

**目标：判断复现实验所需数据能否获取。GitLink 的数据集能力是本维度的关键加分项。**

```bash
# 查询本仓库绑定的 GitLink 科研数据集（复用 dataset 能力）
gitlink-cli dataset +view --owner <owner> --repo <repo> --format json
```

> ⚠️ **实测提示：** 若仓库未绑定数据集，该命令会返回 `404`（如 `[404] 您访问的页面不存在或已被删除`）。
> 这是**正常信号**，应判定为「无绑定数据集」（本维度该项不加分），**不要当作执行失败而中断**。

并在文件树中检测：`data/`、`datasets/`、`*.csv/*.json/*.npz`（小样例数据）、
数据下载脚本（`download_data.sh`、`get_data.py`、`data/README.md`）、
README 中的数据链接（含 `huggingface.co`、`zenodo.org`、`figshare`、网盘、`dataset` 字样）。

**评分细则（满分 20）：**

```
+8  绑定了 GitLink 数据集 或 README/脚本给出可访问的数据获取方式
+5  提供数据下载/预处理脚本（download/preprocess）
+4  仓库内含小规模样例数据，便于快速冒烟测试
+3  数据附带说明（来源、规模、格式、许可/隐私声明）
（无任何数据获取线索 → 本维度 0 分；纯算法/无数据类项目按"不适用"记满分并在报告中标注）
```

### 2.3 维度三 · 运行可复现性（Execution，20 分）

**目标：判断是否有清晰的运行入口与步骤。**

```bash
# 读取 README，提取"运行/使用/快速开始"章节
gitlink-cli repo +readme --owner <owner> --repo <repo> --format json
```

在 README 与文件树中检测：
- 运行入口脚本：`run.sh` / `train.py` / `main.py` / `Makefile`（run/train 目标）/ `scripts/`
- README 是否含 **可复制的命令块**（```bash ... ```）覆盖 安装 → 运行 → 评估
- 配置管理：`config/`、`*.yaml/*.json` 配置、命令行参数说明

**评分细则（满分 20）：**

```
+8  README 含可直接复制运行的命令（从安装到出结果）
+6  存在明确的运行入口（脚本/Makefile/CLI），无需读源码猜
+4  关键超参/路径通过配置文件或命令行参数暴露，而非硬编码
+2  区分了训练 / 推理 / 评估的不同入口
```

### 2.4 维度四 · 结果可复现性（Result，15 分）

**目标：判断能否验证复现结果与论文/声明一致。**

在 README、文件树与发行物中检测：
- 随机种子设置（README 提及 `seed`，或代码中 `set_seed`/`random_state`）
- 预期结果：README 含指标表格 / 期望精度 / 基线对比
- 结果产物：`results/`、`outputs/`、`logs/`、预训练权重（`*.pt/*.ckpt/*.h5` 或 release 附件）
- 评估脚本：`eval.py` / `evaluate.sh` / `test/` 基准

```bash
# 发行版常用于发布权重/结果产物
gitlink-cli release +list --owner <owner> --repo <repo> --format json
```

**评分细则（满分 15）：**

```
+5  README 给出可对照的预期结果（指标/表格/曲线）
+4  提供独立的评估/复现脚本
+3  明确设置并说明随机种子，保证可重复
+3  提供预训练权重/checkpoint 或结果产物（含 release 附件）
```

### 2.5 维度五 · 文档完整性（Documentation，10 分）

```bash
gitlink-cli repo +readme --owner <owner> --repo <repo> --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json   # 含 license 字段
```

**评分细则（满分 10）：**

```
+4  README 结构完整（简介/安装/使用/结果/引用 至少覆盖 4 项）
+3  含 LICENSE（开源许可，复现/复用的法律前提）
+2  含论文/项目引用信息（arXiv / DOI / BibTeX / 顶会顶刊名）
+1  提供中英文双语 README 之一以上的清晰说明
```

### 2.6 维度六 · 代码可维护性（Maintainability，10 分）

```bash
gitlink-cli repo +tree --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributor-stats --owner <owner> --repo <repo> --format json
```

**评分细则（满分 10）：**

```
+4  存在测试（tests/ / *_test.* / test_*.py）
+3  存在 CI 配置（.gitea/workflows、.github/workflows、.gitlink）
+2  目录结构清晰（源码/脚本/配置/文档分离，而非全堆根目录）
+1  近 6 个月有维护更新（updated_at），降低"年久失修跑不通"风险
```

---

## 三、评分汇总与分级

### 3.1 计算总分

```
总分 = E(环境/25) + D(数据/20) + X(运行/20) + R(结果/15) + Doc(文档/10) + M(可维护/10)
```

> 若项目天然「不适用」某维度（如纯理论/无数据项目的"数据可得性"），将该维度按比例剔除后归一化到 100，并在报告中显式标注「数据维度不适用」。

### 3.2 等级与风险映射

| 总分 | 等级 | 复现风险 | 含义 |
|------|------|----------|------|
| 85–100 | **A** | 🟢 低 | 他人可顺利复现，开源就绪 |
| 70–84 | **B** | 🟡 中低 | 基本可复现，少量补充即可 |
| 50–69 | **C** | 🟠 中高 | 需补关键材料才能复现 |
| 0–49 | **D** | 🔴 高 | 当前几乎无法被他人复现 |

### 3.3 复现性雷达（ASCII）

```
              环境
            ★★★★★★★★☆☆ 20/25
             ╲         ╱
   可维护    ╲       ╱   数据
 ★★★★★★★☆☆☆  ╲   ╱  ★★★★★★☆☆☆☆ 12/20
   7/10       ╲ ╱
              ★
            ╱   ╲
          ╱       ╲
  文档   ╱           ╲  运行
★★★★★★★★☆☆ 8/10    ★★★★★★★☆☆☆ 14/20
              结果 ★★★★★★★★☆☆ 12/15
```

---

## 四、完整复现性评估报告模板

```markdown
## 🔬 科研复现性评估报告

**评估对象：** <owner>/<repo>
**评估时间：** 2026-06-15
**主语言：** Python　**许可证：** MIT　**最近更新：** 2026-05-30
**数据来源：** GitLink 平台（gitlink-cli 只读采集）

---

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
| 环境可复现性 | 20 / 25 | ✅ requirements.txt 已 pin 版本；✅ Dockerfile；❌ 未说明 CUDA 版本 |
| 数据可得性 | 12 / 20 | ✅ README 给出 Zenodo 链接；❌ 无下载脚本；❌ 无样例数据 |
| 运行可复现性 | 14 / 20 | ✅ README 有完整命令；✅ train.py 入口；❌ 超参硬编码在脚本里 |
| 结果可复现性 | 12 / 15 | ✅ 指标表格；✅ eval.py；✅ 设种子；❌ 未提供 checkpoint |
| 文档完整性 | 8 / 10 | ✅ 结构完整；✅ LICENSE；✅ arXiv 引用；❌ 仅中文 README |
| 代码可维护性 | 8 / 10 | ✅ tests/；✅ CI；✅ 结构清晰；✅ 近期维护 |

### 三、复现性雷达

（插入第 3.3 节 ASCII 雷达图，数值替换为实际得分）

### 四、修复清单（按优先级 / 投入产出比排序）

| 优先级 | 行动项 | 预计提分 | 维度 |
|--------|--------|----------|------|
| 🔴 P0 | 提供数据下载脚本 `download_data.sh` 并在 README 标注数据规模 | +5 | 数据 |
| 🔴 P0 | 将训练超参从脚本抽到 `config.yaml`，命令行可覆盖 | +4 | 运行 |
| 🟠 P1 | 通过 GitLink Release 发布预训练 checkpoint | +3 | 结果 |
| 🟠 P1 | 在依赖说明中补充 CUDA / 驱动版本 | +2 | 环境 |
| 🟢 P2 | 补充英文 README，扩大国际可复现受众 | +1 | 文档 |

**预计修复后总分：74 → 89（B → A，复现风险 🟡 → 🟢）**

### 五、复现就绪度结论

> 该仓库工程化基础良好，核心障碍集中在「数据获取」与「结果对照」。
> 完成 P0 两项（约半天工作量）即可跨越「可被他人独立复现」的门槛，建议在论文正式开源前完成。
```

---

## 五、执行步骤总览

```bash
# Step 1：采集仓库元数据与技术栈
gitlink-cli repo +info     --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json

# Step 2：取文件树（六维证据的主要来源）
gitlink-cli repo +tree     --owner <owner> --repo <repo> --format json

# Step 3：读 README（运行/数据/结果章节）
gitlink-cli repo +readme   --owner <owner> --repo <repo> --format json

# Step 4：数据可得性（复用数据集能力）
gitlink-cli dataset +view  --owner <owner> --repo <repo> --format json

# Step 5：结果产物 / 版本快照
gitlink-cli release +list  --owner <owner> --repo <repo> --format json
gitlink-cli repo +tags     --owner <owner> --repo <repo> --format json

# Step 6：按需读取关键依赖清单内容判断版本锁定
gitlink-cli api GET "/:owner/:repo/contents/requirements.txt" --format json

# Step 7：可维护性信号
gitlink-cli repo +contributor-stats --owner <owner> --repo <repo> --format json

# Step 8：AI 按第三节规则评分、第四节模板出报告
```

---

## 注意事项

- ✅ **纯只读**：本技能不创建/修改/删除任何仓库内容，对被评估仓库零副作用。
- ✅ **"不适用"维度需归一化**：纯理论/无数据项目应剔除数据维度后重新归一到 100，并在报告标注。
- ⚠️ **文件树可能分页或较大**：大仓库可只取根目录与关键子目录（如 `data/`、`scripts/`、`tests/`），避免全量拉取。
- ⚠️ **raw/contents 取文件**：若 `repo +readme` 已返回 README，则无需再单独取；依赖清单内容用 contents/raw 接口按需读取，失败则降级为"仅凭文件存在性评分"。
- ✅ **评分透明可追溯**：报告中每一维度都要列出"加了哪几分、因为看到什么证据"，便于科研人员据此整改。
- ✅ **最终产出为 Markdown 报告**，可直接粘贴进项目 Wiki 或论文附录。
