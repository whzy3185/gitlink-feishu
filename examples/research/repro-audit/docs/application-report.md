# 科研场景应用报告：基于 gitlink-cli 的科研复现性审计

## 1. 科研问题背景

「复现危机」是学术界公认的系统性问题：大量论文附带的代码仓库缺少依赖清单、运行说明、数据来源与版本固化，导致同行无法复现结果。科研工作者需要一个**低成本、可自动化、判据透明**的工具，在论文投稿与代码开源前完成复现性体检。

## 2. 技术实现

- **数据获取层**：全部通过 `gitlink-cli` 完成——`repo +tree`（仓库结构）、`file +view --raw`（README 内容）、`release +list`（版本发布），无任何直接 HTTP 调用。
- **评估层**：八维确定性检查（README 运行说明 / 许可证 / 引用信息 / 依赖固化 / 运行入口 / 数据可得性 / 测试 / 版本发布），每项有明确判据、分值与证据输出；同输入必同输出，可进 CI。
- **输出层**：0-100 评分卡 + A/B/C/D 等级 + 逐项修复建议；`--apply` 时用 `issue +create` 把报告回写为改进 tracking issue，形成可追踪的整改闭环。
- **工程质量**：纯 Python 标准库（≥3.9）零第三方依赖；7 个确定性单测护栏；默认 dry-run，写操作显式开启。

## 3. 科研赋能价值

| 使用者 | 场景 | 价值 |
|--------|------|------|
| 课题组 | 论文投稿/开源发布前自查 | 逐项补齐复现要件，提升论文可信度 |
| 实验室管理者 | 批量审计组内科研仓库（`--repos-file` 清单模式，输出得分排名汇总表） | 统一学术规范（许可证/引用/数据说明） |
| 期刊/会议 artifact 评审 | 快速初筛 | 评分卡作为客观初审依据 |
| CI 门禁 | 科研仓库发布流程 | 退出码 2 阻断复现缺口明显的发布 |

## 4. 落地效果（生产环境实测）

在 gitlink.org.cn 生产环境对真实科研类仓库验证：

| 仓库 | 类型 | 得分 | 等级 | 主要缺口 |
|------|------|------|------|----------|
| GengzhaoWang/UAV-Paper | 论文笔记仓库 | 8/100 | D | 全维度缺失（符合预期） |
| momomym/paper-summarizer | 科研工具代码仓库 | 58/100 | C | 引用信息、数据说明、release |

区分度显著且每项证据可在仓库页面人工复核；`--apply` 回写 tracking issue 已在自有 fork 实测成功。完整证据见 [`verification.md`](verification.md)，演示输出见 [`../examples/demo-outputs/`](../examples/demo-outputs/)。

## 5. 与 Skills 生态的关系

- 判据与命令链沉淀为 [`gitlink-repro-audit` Skill](../../../../skills/gitlink-repro-audit/SKILL.md)，兼容 Claude Code / Cursor 等主流 AI Agent（标准 SKILL.md 规范）。
- 与 `gitlink-doc-sync`（文档一致性）、`gitlink-license-compliance`（许可证合规）互补，共同覆盖科研仓库的学术规范维度。
