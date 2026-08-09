---
name: gitlink-research-repro
version: 0.1.0
description: "科研仓库复现性体检：扫描一个科研/论文代码仓库，判断『这套实验代码能否被他人复现、卡在哪』——逐维度给出 ✅/⚠️/❌ 与可执行修复建议。当用户问『这个仓库能复现吗』『复现这个实验要准备什么』『这份论文代码卡在哪』『可复现性怎么样』时触发。区别于许可证/依赖合规（gitlink-compliance），本 skill 专查『实验能否跑通、结果能否重现』。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-repro（科研仓库复现性体检）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

## 何时使用 / 何时跳过

**使用本 skill（实验复现性体检）：**
- 用户说"这个仓库能复现吗""复现这个实验要准备什么""这份论文代码卡在哪""可复现性如何"
- 用户想决定是否值得投入算力/时间去复现某科研仓库的实验

**跳过本 skill（改用其他 skill）：**
- 想读懂"这仓库做了什么/方法是什么" → `gitlink-research-insight`（内涵解读）
- 只查许可证 / 依赖合规（LICENSE、开源协议、依赖漏洞） → `gitlink-compliance` / `gitlink-license-compliance`
- 想生成研究 Idea → `gitlink-research-idea`

> **与 `gitlink-compliance` 的本质区别（边界自证）**：合规 skill 查"法律/安全"——LICENSE 是否存在、依赖有无已知漏洞；本 skill 查"科学/工程"——环境能否装上、数据能否拿到、入口能否跑通、随机性是否受控、结果能否重现。一个问"能不能合法用"，一个问"能不能复现出论文数字"。

## ⚠️ 现实约束（驱动设计的前提）

GitLink 上的科研仓库**绝大多数是"成品上传"**——论文发表后整体上传，commit 少（常见 1–5 个）、**不含真实调试/探索过程**（实测靶子仓库仅 3 个 commit 同日）。

→ 因此本 skill **不依赖 commit/CI 历史**（多数仓库根本没有 CI），而是**直接读取仓库内容**（依赖清单、入口脚本、数据加载、配置文件、训练循环里的随机性控制）做静态体检。把"无 CI"本身也作为一条体检结论（❌ 自动化测试缺失）。

## 数据采集（只用这些 gitlink-cli 命令）

```bash
# 0) 前置：认证
gitlink-cli auth status

# 1) 元信息（默认分支 / 规模 / 是否镜像）
gitlink-cli repo +info --owner <owner> --repo <repo> --format json

# 2) README —— 运行说明 / 数据获取 / 预训练权重链接多在此
gitlink-cli repo +readme --owner <owner> --repo <repo> --format json

# 3) 文件结构 —— 定位依赖清单 / 入口 / 配置 / 数据加载文件
gitlink-cli repo +tree --owner <owner> --repo <repo> --format json

# 4)【关键】读取任意文件内容 —— 用 sub_entries 端点（不是 /contents/）
#    Git Bash 下必须加 MSYS_NO_PATHCONV=1，否则前导 / 被转成 Windows 路径 → 404 → SPA HTML
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/<owner>/<repo>/sub_entries" \
  --query "filepath=<path>&ref=master" --format json
#    文件正文在返回的 data.entries.content（已解码纯文本）
```

**按需读取的"复现性证据文件"清单**（先看 tree，存在哪个读哪个）：

| 证据文件 | 回答的体检维度 |
|----------|----------------|
| `requirements.txt` / `environment.yml` / `Pipfile.lock` / `poetry.lock` | 环境锁定、依赖版本、是否可一键安装 |
| `main.py` / `train.py` / `run.py`（入口） | 运行入口是否清晰、参数是否暴露 |
| `Makefile` / `Justfile` / `run.sh` / `samples.sh` | 是否有"一键复现"脚本 |
| `Dockerfile` / `.devcontainer/` | 环境是否容器化（复现性最强证据） |
| `utils/data.py` / `data_loader.py` / `datasets/` | 数据集获取方式（自动下载 vs 手填路径） |
| `exps/*.json` / `configs/*.yaml`（实验配置） | 随机种子、超参、模型名是否落盘 |
| `trainer.py` / `engine.py` / 训练循环 | 随机性控制（seed / cudnn.deterministic） |
| `.github/workflows/` / `tests/` | 是否有自动化测试/CI 兜底 |

> **镜像仓库注意**：`repo +info` 的 `contributor_users_count` 对 GitHub 镜像恒为 0，无关复现性体检，可忽略。
> **读文件路径**：`/raw/` 端点对 API token 常返回 403，**用 `/sub_entries`**（所有现有 skill 的统一做法）。

## 体检流程

### Step 1：采集（执行上述命令；先 `+info`/`+tree` 定位，再 `sub_entries` 逐个读证据文件）

### Step 2：逐维度判定（每维给 ✅ / ⚠️ / ❌ + 一句依据）

按下表 **8 个维度**（≥6 即达标）逐项检查。每项必须**引用具体文件/行/字段**作依据，可核查：

| # | 维度 | 判 ✅ 的条件 | 判 ❌/⚠️ 的典型情形 |
|---|------|-------------|---------------------|
| 1 | **环境锁定** | 有锁定文件且全量 pin 版本 | 仅 `requirements.txt` 无版本；无任何锁定文件 |
| 2 | **可一键安装** | 锁定文件能被标准工具直接消费（pip/env.yml/Docker） | 是 conda `--file` 导出却命名 `requirements.txt`（pip 装不了）；缺 python 版本 |
| 3 | **运行入口** | 有明确 `main.py`/`train.py` + 参数说明 | 无入口脚本；参数全靠硬编码；README 无运行命令 |
| 4 | **数据集获取** | 数据自动下载或有明确下载脚本+路径配置 | 需手填本地绝对路径；私有数据无获取说明 |
| 5 | **随机性受控** | config 有 seed + 训练循环设 `cudnn.deterministic=True` | 无 seed；seed 未传到训练；未关 cudnn.benchmark |
| 6 | **预训练权重** | 提供 checkpoint 下载链接或 `--loadpre=0` 即可跑 | 有 `--loadpre` 开关却无下载链接；强依赖未公开权重 |
| 7 | **硬件/兼容** | 有 CPU 回退或声明最低算力；依赖栈非远古 | 强制特定 GPU 无回退；torch 为多年前旧版（装不上新卡） |
| 8 | **自动化测试** | 有 `tests/` 或 CI workflow | 无任何测试 / 无 CI（成品仓库常见） |

> 维度可按仓库语言增减（如 Go 仓库看 `go.mod`+`go.sum`，Node 看 `package-lock.json`），但**不少于 6 维**，且必须覆盖：环境、数据、入口、随机性、权重。

### Step 3：给修复建议（每个 ⚠️/❌ 都要可执行）

对每个未达标项，给一条**具体、可粘贴**的修复建议（如"补一个 `environment.yml`""在 README 加 `wget <权重链接>`""把 `torch.manual_seed(1)` 改成 `torch.manual_seed(args['seed'])`"）。

### Step 4：生成中文「复现性体检报告」（按下述模板，Write 工具产出 Markdown）

## 输出模板：复现性体检报告

```markdown
# 🔧 复现性体检报告：<仓库标题/论文名>

> 体检对象：<owner>/<repo>  ｜  数据来源：GitLink（gitlink-cli 实时采集）  ｜  体检时间：{{date}}
> 结论先行：**复现难度 = 低 / 中 / 高**（一句话：能不能复现、卡点在哪、值不值得投入）

## 一、体检总表（8 维度）
| # | 维度 | 状态 | 一句依据（带文件来源） |
|---|------|------|------------------------|
| 1 | 环境锁定 | ✅/⚠️/❌ | …（如 `requirements.txt` 56 包全 pin） |
| 2 | 可一键安装 | ✅/⚠️/❌ | … |
| ... | ... | ... | ... |

**总览**：✅ ×n  ⚠️ ×n  ❌ ×n  →  复现难度评级 + 理由

## 二、复现路径（按这个顺序操作即可复现）
1. 环境准备：…（具体命令）
2. 数据准备：…
3. 运行：…（从 README/samples.sh 提取的真实命令）
4. 预期产出：…

## 三、卡点与修复建议（逐条对应 ⚠️/❌）
| 卡点 | 现状（依据） | 修复建议（可执行） | 严重度 |
|------|--------------|---------------------|--------|
| … | … | … | 阻断/影响精度/仅整洁 |

## 四、一句话评价
（这套实验代码复现性整体如何、最该先补什么、适合谁复现）
```

> **溯源要求**：每条 ✅/⚠️/❌ 必须引用具体文件/字段/行，可核查。
> **边界声明**：本报告**不评判 LICENSE 合规**（那是 `gitlink-compliance` 的职责）；如发现无 LICENSE，仅提示"请交合规 skill 处理"，不计入复现性评分。

## 示例

完整范例见 [`examples/maintaining-fairness-lkd-cil-repro.md`](examples/maintaining-fairness-lkd-cil-repro.md)（靶子仓库 `gaozijian19/Maintaining-Fairness-in-LKD-for-CIL` 的真实体检，含 conda --file 误命名、seed 半硬编码等真实卡点）。

## 注意事项

- ✅ **sub_entries 是读文件正解**：`/raw/` 对 token 常 403、`/contents/` 在 GitLink 不存在；统一用 `/sub_entries?filepath=<path>&ref=<branch>`，正文在 `data.entries.content`。
- ✅ **静态体检为主**：成品仓库无 CI 历史，靠读"依赖/入口/数据/配置/训练循环"做静态判定，不依赖 commit。
- ⚠️ **Git Bash 路径转换**：raw `api` 带前导 `/` 的参数会被 MSYS 转成 Windows 路径，必须 `MSYS_NO_PATHCONV=1`。
- ✅ **每条结论带文件来源**，修复建议要具体到可粘贴的命令/代码改动。
- ✅ **报告以中文 Markdown 产出**。
