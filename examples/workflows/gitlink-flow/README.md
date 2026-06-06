# gitlink-flow：社区运营自动化端到端工作流

面向 GitLink 智能化服务开源项目贡献赛 **子赛题三（构建端到端自动化工作流）** 的参赛作品。

把开源社区运营中分散的日常工作——Issue 分拣、PR Review 跟进、Release Notes 整理、社区健康检查、贡献者致谢——**串联成一条可一键运行的自动化流水线**，对任意真实 GitLink 仓库生成一份完整的「社区运营周报」。

作者：Ct201314

## 这是什么

子赛题三要求「组合 gitlink-cli 已有命令和 Skills，完成一个可复现的端到端自动化场景，串联 ≥3 个调用」。本作品串联 **6 个步骤**，复用了我在子赛题二完成的 **5 个自研 Skill** 的能力，远超 3 个的要求：

```
                      gitlink-flow 端到端工作流
┌──────────────────────────────────────────────────────────────┐
│                                                                │
│   采集（GitLink 公开 API，只读）                                  │
│   仓库信息 / Issue / PR / 提交 / 贡献者 / 版本 / 文件树            │
│                          │                                     │
│                          ▼                                     │
│   ① Issue 自动分拣 ──────────  bug/feature/question/新手友好分类  │
│                          │      （对标官方 triage 参考工作流）     │
│                          ▼                                     │
│   ② PR Review 汇总 ─────────  开放/合并/关闭统计 + 待 Review 清单 │
│                          │      （对标官方 PR Review 参考工作流）  │
│                          ▼                                     │
│   ③ Release Notes 生成 ─────  按 conventional commits 归类       │
│                          │      （对标官方 Release Notes 参考）   │
│                          ▼                                     │
│   ④ 社区健康体检 ───────────  复用 gitlink-scaffold 能力          │
│                          │                                     │
│                          ▼                                     │
│   ⑤ 贡献者致谢 ─────────────  复用 gitlink-contributor 能力      │
│                          │                                     │
│                          ▼                                     │
│   ⑥ 社区运营周报 ───────────  汇总以上全部为一份 Markdown 周报     │
│                                                                │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
            outputs/<owner>_<repo>_flow.md（周报）
                  或 --format json（供 Agent 消费）
```

其中 ①②③ 三个子工作流**精准对标官方 `examples/workflows/` 列出的三个参考场景**（Issue 自动分拣、PR Review、Release Notes 生成），④⑤ 复用自研 Skill，⑥ 汇总输出。

## 与子赛题二的区别

子赛题二交付的是**独立可复用的 Skill**；本作品交付的是**串联多个步骤的完整解决方案**——它把这些能力编排成一条端到端流水线，一条命令产出社区运营全景。

## 快速开始

环境：Python 3.10+（采集与编排仅用标准库，无需登录）。

```powershell
# 一键运行（默认分析 Gitlink/gitlink-cli）
.\scripts\run_demo.ps1

# 指定仓库
.\scripts\run_demo.ps1 -Owner <owner> -Repo <repo>

# 或直接调用
python src/flow.py --owner Gitlink --repo gitlink-cli
python src/flow.py --owner Gitlink --repo gitlink-cli --format json
python src/flow.py --config examples/config.json   # 批量多仓库
```

产物输出到 `outputs/<owner>_<repo>_flow.md`。

## 目录结构

```
gitlink-flow/
├── src/
│   ├── glapi.py        GitLink 公开 API 客户端（采集层）
│   ├── steps.py        工作流步骤库（三个子工作流 + 复用 Skill 能力）
│   ├── flow.py         编排器主入口（串联 6 步）
│   └── report.py       社区运营周报生成
├── scripts/run_demo.ps1     一键复现脚本
├── examples/
│   ├── config.json          批量配置
│   └── demo_outputs/         真实运行产物（社区运营周报）
├── tests/test_flow.py       单元测试（16 个用例）
├── docs/                    架构、快速开始、验证记录
├── requirements.txt
└── LICENSE
```

## 真实验证

在真实仓库 `Gitlink/gitlink-cli` 上跑通，生成的周报准确反映了社区现状：17 个 Issue → 识别 5 个新手友好任务；20 个开放 PR（含真实的 fork PR）列入待 Review；社区健康度 50/100；贡献者致谢榜；从 127 条提交生成的分类 Release Notes。完整产物见 [`examples/demo_outputs/`](examples/demo_outputs/)，验证记录见 [`docs/verification.md`](docs/verification.md)。

单元测试：

```bash
python -m pytest tests/ -q   # 16 passed
```

## 设计要点

- **组合而非重写**：编排器只负责串联，每个步骤是独立可测的纯函数，复用已验证的 Skill 能力。
- **对标官方参考**：三个子工作流对齐官方 examples/workflows 的三个参考场景，便于理解与收录。
- **通用可复现**：对任意 GitLink 公开仓库都能跑，数据只读、零第三方依赖（核心）、一键复现。
- **Agent 友好**：`--format json` 输出供 AI Agent 消费，可嵌入更大的自动化链路。

## 许可证

[Mulan PSL v2](LICENSE)，与 gitlink-cli 主仓库保持一致。
