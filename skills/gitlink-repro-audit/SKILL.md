---
name: gitlink-repro-audit
version: 1.0.0
description: "科研项目复现性审计：对 GitLink 上的科研代码仓库做八维复现性检查（文档/许可证/引用/依赖/入口/数据说明/测试/版本固化），生成 0-100 评分卡与修复建议，并可回写改进 tracking issue。当用户需要评估科研仓库可复现性、准备论文代码发布、或做学术规范检查时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-repro-audit（科研复现性审计）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 审计本身只读；`--apply` 回写 tracking issue 前必须先向用户展示报告并确认。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

---

## 功能概述

论文代码「跑不起来」是科研复现危机的核心。本技能对科研代码仓库做系统化复现性评估：

| 检查项 | 分值 | 判据 |
|--------|------|------|
| README 与运行说明 | 15 | 存在 README 且含 How to run / 复现步骤章节（仅有 README 得 8 分） |
| 开源许可证 | 15 | LICENSE / COPYING 文件 |
| 引用信息 | 10 | CITATION.cff 或 README 中的 BibTeX/引用章节 |
| 依赖清单（环境固化） | 15 | requirements.txt / environment.yml / go.mod / package.json 等 |
| 运行入口 | 10 | Makefile / run.sh / main.py / Dockerfile / scripts 目录 |
| 数据可得性说明 | 10 | data 目录或 README 中的数据集来源说明 |
| 测试/验证代码 | 10 | tests 目录 |
| 版本发布（成果固化） | 15 | 至少一个 release |

总分映射等级：≥85 A（可复现性良好）/ ≥70 B / ≥50 C / <50 D（复现困难）。

## 使用方式

### 方式一：确定性脚本（推荐，可进 CI）

```bash
python3 examples/research/repro-audit/scripts/repro_audit.py \
  --owner <owner> --repo <repo> [--ref <branch>] [--apply]
```

- 默认 dry-run 只输出报告；`--apply` 用 `issue +create` 回写改进 tracking issue
- 退出码：`0` 总分 ≥70（基本可复现），`2` 存在明显缺口

### 方式二：AI Agent 手工执行命令链

```bash
# 1. 结构采集
gitlink-cli repo +tree --owner <owner> --repo <repo> --format json
# 2. README 内容
gitlink-cli file +view --owner <owner> --repo <repo> --path README.md --raw
# 3. 版本发布
gitlink-cli release +list --owner <owner> --repo <repo> --format json
# 4.（确认后）回写改进 issue
gitlink-cli issue +create --owner <owner> --repo <repo> -t "[repro-audit] 复现性审计报告" -b "<报告>"
```

AI 按上表逐项打分，报告须给出每项证据与修复建议，不允许笼统结论。

## 注意事项

- 判据基于仓库顶层结构与 README 文本，属于必要条件检查：高分不代表结果一定可复现，低分则一定存在工程缺口。
- 对论文笔记类仓库（纯 Markdown）等级普遍偏低，属预期行为；报告建议仅在其确为「实验代码仓库」时回写。
- 与 [`../gitlink-doc-sync/SKILL.md`](../gitlink-doc-sync/SKILL.md)（文档一致性）、[`../gitlink-license-compliance/SKILL.md`](../gitlink-license-compliance/SKILL.md)（许可证合规）互补：本技能聚焦复现性维度的整体评估。
