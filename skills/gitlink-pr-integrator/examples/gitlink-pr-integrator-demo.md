# gitlink-pr-integrator 使用示例

## 场景 1：单条 PR 集成检查

用户请求：

```text
使用 gitlink-pr-integrator 检查 Gitlink/gitlink-cli 的 PR #281 是否已经具备合并条件，执行合并态验证、冲突风险分析、发布影响判断和合并后动作清单整理，不要回写远端。
```

期望输出重点：

- 明确的结论：`ready_to_merge` / `ready_after_rebase` / `ready_after_followups` / `not_integration_ready`
- 是否能在最新 base 上 clean merge
- 官方构建和测试是否通过
- 与其他 open PR 的冲突风险
- 是否需要补文档、help、changelog 或 release notes

## 场景 2：批量扫描合并队列

用户请求：

```text
使用 gitlink-pr-integrator 扫描 Gitlink/gitlink-cli 最近 10 个 open PR，给出建议合并顺序、冲突热点、发布影响和需要优先处理的阻塞项，不要回写远端。
```

期望输出重点：

- 候选 PR 列表
- 每条 PR 的集成结论和冲突等级
- 推荐合并顺序
- 重叠最严重的文件或模块
- 只对高风险项进入本地合并验证

## 演示建议

在 Codex 中优先使用仓库源码构建的 CLI，而不是依赖全局安装版本。Windows 下如果 PowerShell 拦截 `gitlink-cli`，就改用 `gitlink-cli.cmd`，或直接在仓库根目录用：

```powershell
go run . pr --help
```
