# Codex 验证记录：扫描前 40 条 open PR

这个示例记录了 `gitlink-pr-assessor` 在 Codex 中的一次实际验证，用来证明这个 skill 可以在真实仓库上完成 open PR 队列扫描、逐条评估和维护者报告整理。

## 验证环境

- Agent 平台：Codex
- 仓库：`Gitlink/gitlink-cli`
- 时间：2026-06-24
- 模式：只读扫描，不回写 PR

## 使用提示词

```text
$gitlink-pr-assessor 扫描 Gitlink/gitlink-cli 仓库当前前 40 条 open PR。只保留 pull_request_status == 0、且没有 approved / rejected、且没有 assessor 报告标记的 PR。对符合条件的 PR 生成一份队列总览，并为每条 PR 生成维护者报告，不要回写到 PR。
```

## 产出结果

- 这次验证已经成功生成队列总览和逐条评估结果，证明 skill 可以在真实 open PR 队列上完成批量筛查。
- 为避免把一次性运行日志和截图长期提交进仓库，详细报告与界面截图不再作为仓库内容保留；如需展示，可在 PR 描述、评审回复或单独的演示材料中引用。
- 仓库内保留这份验证说明，作为“已在真实项目执行过”的复核依据。

## 验证结论

- 成功扫描当前列表顺序下前 40 条 open PR。
- 按 `pull_request_status == 0`、review 状态和 assessor 标记完成过滤。
- 对纳入范围的 PR 生成了维护者队列总览和逐条报告。
- 全程没有向 PR 回写内容，也没有修改当前工作树中的既有文件。

## 结果摘要

- 纳入评估：39 条
- 跳过：1 条
- `origin/master` 基线上的 `go build ./...` 与 `go test ./...` 均通过
- 扫描结果能够区分可继续推进、需补测试、需拆分、建议拒绝等不同结论
