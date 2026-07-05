# 🔧 复现性体检报告：Maintaining Fairness in Logit-based Knowledge Distillation for Class-Incremental Learning

> 体检对象：`gaozijian19/Maintaining-Fairness-in-LKD-for-CIL` ｜ 数据来源：GitLink（gitlink-cli 实时采集）｜ 体检时间：2026-07-03
> 本报告由 `gitlink-research-repro` skill 流程产出，所有结论可经 gitlink-cli 命令复现核查。
>
> **结论先行：复现难度 = 低偏中。** 主路径（CIFAR-100 + interintra 方法）能跑通——入口清晰、依赖全 pin、有 seed 与 cudnn 确定性控制、CIFAR 自动下载、还带 CPU 回退；主要卡点是依赖栈偏旧（torch 1.8.1+cu111，2021 年）、`requirements.txt` 实为 conda 导出却按 pip 命名（直接 `pip install -r` 会失败）、数据加载器初始化 seed 半硬编码、无任何测试/CI。**值得投入复现**，按本报告第二节操作即可。

---

## 一、体检总表（8 维度）

| # | 维度 | 状态 | 一句依据（带文件来源） |
|---|------|------|------------------------|
| 1 | 环境锁定 | ✅ | `requirements.txt` 为 conda 全量导出，56 个包逐个 pin（`python=3.8.20`、`numpy=1.24.4`、`scipy=1.10.1`…） |
| 2 | 可一键安装 | ⚠️ | 同文件**实为 conda `--file` 格式**（`name=version=build`，如 `torch=1.8.1+cu111=pypi_0`），文件头自注 `conda create --name <env> --file <this file>`；**命名 `requirements.txt` 易被误用 `pip install -r`，后者会失败** |
| 3 | 运行入口 | ✅ | `main.py` 为干净 argparse 入口 → `trainer.train(args)`；README 给出完整运行命令；`samples.sh` 提供大量样例（**但整文件每行皆被 `#` 注释，是"菜单"非可执行脚本**） |
| 4 | 数据集获取 | ⚠️ | `utils/data.py`：CIFAR-10/100 走 `datasets.cifar.CIFAR100("./data", download=True)` **自动下载** ✅；但 ImageNet-Subset / Tiny-ImageNet / Librispeech 为 `use_path=True`/torchaudio，**需手动准备并填写路径**；无统一 `--datapath` 配置项 |
| 5 | 随机性受控 | ✅（带小瑕疵） | `exps/lwf.json` 有 `"seed": [1993]`；`trainer.py` 设 `cudnn.deterministic=True`、`benchmark=False` ✅。**瑕疵**：`trainer.py:170-172` 数据加载器初始化处 `torch.manual_seed(1)` 为**硬编码**，未取 `args["seed"]`，与 config 的 1993 不一致（见第三节） |
| 6 | 预训练权重 | ⚠️ | `main.py` 有 `--loadpre`(默认 0) 与 `--path`(默认 `temp.pth`) 开关；README 未提供任何 checkpoint 下载链接。**默认 `--loadpre 0` 可不依赖权重跑通**，故非阻断 |
| 7 | 硬件 / 兼容 | ⚠️ | `trainer.py:159-160` **有 CPU 回退**（`device_type == -1 → torch.device("cpu")`）✅；但依赖 `torch=1.8.1+cu111`（2021-03），**在新 CUDA driver / 新架构 GPU（如 40 系）上直装易失败**，需借旧镜像或源码编译 |
| 8 | 自动化测试 | ❌ | 无 `tests/`、无 `pytest.ini`、无 `.github/workflows/`、无 `Makefile`；成品上传（3 commit 同日），无 CI 兜底，复现正确性只能靠人工核对 |

**总览**：✅ ×3 ｜ ⚠️ ×4 ｜ ❌ ×1 → **复现难度：低偏中**。无阻断项（最大风险是旧版 torch 的安装兼容，而非代码本身）。

---

## 二、复现路径（按此顺序操作即可复现 CIFAR-100 主实验）

1. **环境准备**（用 conda，不要用 pip）：
   ```bash
   # 正确方式：按文件头注释，用 conda --file 消费
   conda create --name fairlkd --file requirements.txt
   conda activate fairlkd
   # 若新机器装不上 torch=1.8.1+cu111，退而求其次：
   #   conda create -n fairlkd python=3.8.20 && conda activate fairlkd
   #   pip install torch==1.8.1+cu111 torchvision==0.9.1+cu111 \
   #     -f https://download.pytorch.org/whl/torch_stable.html
   #   再按 requirements.txt 补 numpy==1.24.4 scipy==1.10.1 pillow==10.4.0 等
   ```
2. **数据准备**：CIFAR-100 首次运行时自动下载到 `./data`（无需操作）；若跑 ImageNet-Subset / Tiny-ImageNet / Librispeech，需自备数据并按 `utils/data.py` 中对应类的路径约定放置。
3. **运行**（README 真实命令，等价于 `samples.sh` 中任一 interintra 行去掉前导 `#`）：
   ```bash
   python main.py --config './exps/lwf.json' --init_cls 20 --increment 20 \
     --device "0" --method "interintra" --dataset "cifar100" --loadpre 0
   # 无 GPU 时改 --device "-1" 走 CPU（trainer.py:159-160）
   ```
4. **预期产出**：训练日志与各阶段精度（`prefix=reproduce`，见 `exps/lwf.json`），可与论文 Table 对比。

---

## 三、卡点与修复建议（逐条对应 ⚠️/❌）

| 卡点 | 现状（依据） | 修复建议（可执行） | 严重度 |
|------|--------------|---------------------|--------|
| 依赖清单命名误导 | `requirements.txt` 实为 conda `--file` 格式，`pip install -r` 失败（`requirements.txt` 文件头 + `torch=1.8.1+cu111=pypi_0` 行） | 改名 `environment.txt` 或补一份真正的 `environment.yml`（`conda env export -f environment.yml`）；README 明确写"用 `conda create --file`，勿用 pip" | 影响首次安装 |
| 数据集路径分散 | ImageNet/Tiny/Librispeech 需手动准备，路径散落在 `utils/data.py` 各类里，无统一配置（`utils/data.py`） | 加一个 `--datapath` 全局参数，或在 `exps/*.json` 增 `data_root` 字段，`utils/data.py` 统一读取 | 影响非 CIFAR 实验 |
| seed 半硬编码 | `trainer.py:170-172` 数据加载器初始化用字面量 `torch.manual_seed(1)`，未用 `args["seed"]`(config 为 1993)；主循环 seed 走 config 但数据侧不一致 | 把 `1` 改为 `args["seed"]`：`torch.manual_seed(args["seed"])`（含 `cuda.manual_seed_all(args["seed"])`） | 影响严格可复现 |
| 预训练权重无链接 | `--loadpre` 开关存在但 README 无 checkpoint 下载地址（`main.py` 默认 `--loadpre 0`） | 默认路径不阻断；若作者希望支持 phase-1 续训，在 README 补 checkpoint 下载链接与 `--path` 用法 | 仅整洁（非阻断） |
| 旧版依赖栈 | `torch=1.8.1+cu111`(2021-03) 在新 CUDA/新 GPU 上难直装（`requirements.txt`） | 提供一份现代版可选依赖（如 `torch>=2.0`）或在 README 列出已知可用的 docker/镜像；说明最低算力 | 影响新机器安装 |
| 无自动化测试 | 无 `tests/`/CI/`Makefile`（`+tree` 核查） | 至少补一个 smoke test：固定 seed 跑 1 个 epoch 的小规模 CIFAR-10，断言精度 > 随机基线，放 `.github/workflows/test.yml` | 影响长期可信 |

---

## 四、一句话评价

工程整洁度中上的科研仓库：**入口、配置、seed、确定性控制、CPU 回退俱全**，CIFAR 主路径基本"装好环境就能跑"；真实卡点集中在**依赖清单的 conda/pip 命名误导**与**旧版 torch 的安装兼容**，都是"半天内可补"的工程债，不涉及算法正确性。适合作为 CIL/KD 方向的复现与扩展起点；建议作者至少补一份 `environment.yml` 和一个 smoke test，即可把复现难度从中压到低。

---

## 附：本报告采集命令（可复现）

```bash
OWNER=gaozijian19
REPO=Maintaining-Fairness-in-LKD-for-CIL

# 元信息 / README / 文件结构
gitlink-cli repo +info   --owner $OWNER --repo $REPO --format json
gitlink-cli repo +readme --owner $OWNER --repo $REPO --format json
gitlink-cli repo +tree   --owner $OWNER --repo $REPO --format json

# 读取复现性证据文件（sub_entries，正文在 data.entries.content）
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=requirements.txt&ref=master" --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=main.py&ref=master"        --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=trainer.py&ref=master"      --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=utils/data.py&ref=master"   --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=exps/lwf.json&ref=master"    --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET "/$OWNER/$REPO/sub_entries" --query "filepath=samples.sh&ref=master"      --format json
```
