# GitLink 科研热点追踪器

基于 GitLink 平台的开源科研仓库多维度自动化分析流水线。

## 这是什么

`repos.txt` 列出一批仓库 → `run_all.sh` 一键执行数据采集、分析、可视化 → 输出 CSV 统计表、PNG 图表和 Markdown 报告，帮你快速了解一批开源科研仓库的代码规模、活跃度、技术栈和贡献者生态。

分析流水线适配所有科研领域，当前示例是蛋白质结构预测领域的 5 个仓库，换成任何其他领域只需改 `repos.txt`。

## 项目结构

```
├── repos.txt                              # 仓库列表（一行一个 owner/repo）
├── run_all.sh                             # 一键运行入口
├── research_application_report.md         # 科研场景应用报告（完整技术说明）
│
├── scripts/
│   ├── fetch_data_v2.sh                   # 批量数据采集（调用 gitlink-cli）
│   ├── analyze_v2.py                      # JSON → CSV 数据分析
│   └── visualize.py                       # 生成 4 张可视化图表
│
├── data/
│   ├── raw/                               # 每仓库 4 个 JSON：info、code_stats、contributors、languages
│   └── processed/                         # 分析后的 4 个 CSV 文件
│       ├── repo_summary.csv               # 仓库基本信息
│       ├── code_stats.csv                 # 代码规模 + Top3 贡献者
│       ├── contributor_network.csv        # 贡献者跨仓库分布
│       └── language_distribution.csv      # 语言分布
│
└── output/                                # 图表与报告
    ├── code_volume_comparison.png         # 代码新增/删除对比柱状图
    ├── activity_bubble.png                # 提交次数 vs 贡献者人数气泡图
    ├── language_distribution.png          # 语言分布堆叠条形图
    ├── cross_repo_contributors.png        # 跨仓库贡献者水平条形图
    ├── struct.svg                         # 系统架构图
    ├── research_trend_report.md           # 科研趋势分析报告
    └── research-application-report.md     # 科研场景应用报告
```

## 快速开始

```bash
# 1. 安装依赖
pip install pandas matplotlib

# 2. 编辑仓库列表（可选，默认分析了 5 个蛋白质结构预测仓库）
#    vim repos.txt

# 3. 一键运行
bash run_all.sh
```

运行后数据在 `data/processed/`，图表和报告在 `output/`。

## 流水线详解

`run_all.sh` 按顺序执行三步：

### 第一步：数据采集（`scripts/fetch_data_v2.sh`）

逐行读取 `repos.txt`，对每个仓库调用 `gitlink-cli` 四个子命令：

| 命令 | 输出 JSON | 内容 |
|------|-----------|------|
| `repo +info` | `{owner}_{repo}_info.json` | 仓库名、大小、描述、默认分支 |
| `repo +code-stats` | `{owner}_{repo}_code_stats.json` | 代码新增/删除行数、提交次数、按作者统计 |
| `repo +contributors` | `{owner}_{repo}_contributors.json` | 贡献者列表（含邮箱、贡献占比） |
| `repo +languages` | `{owner}_{repo}_languages.json` | 各编程语言代码占比 |

5 个仓库 → 20 个 JSON 文件，存入 `data/raw/`。

### 第二步：数据分析（`scripts/analyze_v2.py`）

解析 raw JSON，输出 4 个标准化 CSV：

- **repo_summary.csv** — 各仓库基本属性（大小、分支等）
- **code_stats.csv** — 代码新增/删除量、提交数、作者数、Top3 贡献者及提交数
- **contributor_network.csv** — 每个贡献者的邮箱、覆盖仓库数、仓库列表
- **language_distribution.csv** — 各仓库的编程语言百分比矩阵（直接可用于可视化）

### 第三步：可视化（`scripts/visualize.py`）

基于 CSV 数据生成 4 张 matplotlib 图表（dpi=150）：

1. **代码量对比柱状图** — 各仓库新增/删除行数对比
2. **活跃度气泡图** — x 轴作者数、y 轴提交数、气泡大小 = 新增代码量
3. **语言分布堆叠条形图** — 各仓库技术栈统一性一览
4. **跨仓库贡献者条形图** — 参与多个仓库的核心开发者（Top 15）

## 当前分析结论

基于 5 个蛋白质结构预测镜像仓库的真实数据：

| 指标 | AlphaFold3 | alphafold | AlphaFill | RoseTTAFold | OpenFold |
|------|-----------|-----------|-----------|-------------|----------|
| 提交次数 | 244 | 198 | 279 | 34 | **604** |
| 贡献者 | 14 | **28** | 3 | 2 | 9 |
| 新增代码 | 8.4 万行 | 5.5 万行 | 27.2 万行 | **42.3 万行** | 16.6 万行 |
| 巴士因子 | ≥3 | ≥3 | 1 | 1 | ≥3 |
| 技术栈 | Python 85% + C++ 14% | Python 97% | C++ 80% | Python 98% | Python 94% |

关键发现：
- **OpenFold** 提交 604 次，9 名贡献者高强度迭代，社区开放协作程度最高
- **AlphaFill** 和 **RoseTTAFold** 巴士因子仅 1，高度依赖单一核心开发者
- **Augustin Zidek**（DeepMind）是唯一跨 `AlphaFold3` 和 `alphafold` 的核心贡献者
- 全部仓库 Python 占主导，技术栈高度统一

详细分析见 `output/research_trend_report.md`。

## 扩展到其他领域

修改 `repos.txt` 中的仓库列表即可：

```txt
# 示例：气候模拟领域
some-lab/CESM
some-lab/WRF
some-lab/MPAS

# 示例：基因组学
broadinstitute/gatk
luntergroup/octopus
```

重新运行 `bash run_all.sh`，整套管道自动适配。领域不影响任何分析逻辑——指标全部来自 GitLink 通用 API，不依赖领域知识。

补充说明：分析仓库必须是 GitLink 平台上的仓库。对于 GitHub 原仓库，需要先在 GitLink 上创建镜像。

## 环境要求

- **Python** 3.9+
- **pip** 依赖：`pandas`, `matplotlib`
- **gitlink-cli**（用于数据采集阶段，需 GitLink 账号 token）
- 操作系统：Linux / macOS / WSL（脚本为 Bash）
