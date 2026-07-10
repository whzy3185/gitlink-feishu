# 科研仓库活跃度快照：{owner}/{repo}

> 生成时间（UTC）：{generated_at} ｜ 工具：examples/research 科研辅助模板

## 结论

{conclusion}

## 指标

| 指标 | 值 |
|------|-----|
| 开放 issue | {open_issues} |
| 已关闭 issue | {closed_issues} |
| issue 关闭率 | {close_rate} |
| 开放 PR | {open_prs} |
| 分支数 | {branches} |
| 近 {active_days} 天内有 issue 更新 | {recently_active} |

## 证据与复现命令

任何人可用以下命令独立复核上表数据：

```bash
gitlink-cli issue +list --owner {owner} --repo {repo} --state open --format json
gitlink-cli issue +list --owner {owner} --repo {repo} --state closed --format json
gitlink-cli pr +list --owner {owner} --repo {repo} --format json
gitlink-cli branch +list --owner {owner} --repo {repo} --format json
```

## 方法说明

- 指标计算为确定性纯函数（同输入同输出），实现见 `scripts/research_tool_template.py` 的 `compute_metrics()`
- 活跃判定阈值：最近 {active_days} 天内存在 issue 更新
