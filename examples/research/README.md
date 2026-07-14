# 科研辅助开发模板（examples/research）

对应起步资源包「科研辅助开发模板」：科研场景脚本、分析工具、报告生成模板三件套。
开箱即用（自带一个可运行的示例场景），也可按占位符改造成你自己的科研辅助工具（子赛题四）。

## 内容

| 文件 | 作用 |
|------|------|
| `scripts/research_tool_template.py` | 科研场景脚本模板：采集（gitlink-cli，自动翻页）→ 指标计算（确定性纯函数）→ 报告渲染 三段式骨架，纯标准库 |
| `report-template.md` | 报告生成模板：`{placeholder}` 占位符由脚本填充，评审可复核的四元组结构（结论/指标/证据/复现命令） |
| `tests/test_metrics.py` | 指标纯函数单测样例（不访问网络） |

## 快速运行（自带示例：仓库科研活跃度快照）

```bash
gitlink-cli auth login

python3 scripts/research_tool_template.py \
  --owner <owner> --repo <repo> --output report.md
```

产出 `report.md`：开放/关闭 issue 数、open PR 数、分支数、近 30 天活跃判定，以及每项指标的复现命令。

## 改造为你自己的科研工具（3 步）

1. **采集**：改 `collect()`——用 `gitlink-cli <命令> --format json` 拉你需要的数据（数据多时记得翻页合并）
2. **指标**：改 `compute_metrics()`——保持**纯函数 + 确定性**（同输入同输出），这样单测和评审复核才可行
3. **报告**：改 `report-template.md` 的占位符——保留「复现命令」一节，科研结论必须可复现

## 设计约定（子赛题四评审要点）

- **可复现**：报告内嵌每项指标的采集命令，任何人可独立复核
- **确定性**：指标计算与阈值判定为纯函数，禁止依赖随机/时序副作用（时间基准由 `--now` 注入，便于测试）
- **零依赖**：纯 Python 标准库，无需装包即可在评审机运行
- 参考实现：更完整的科研复现性审计见 [`skills/gitlink-repro-audit`](../../skills/gitlink-repro-audit/)（若已收录）
