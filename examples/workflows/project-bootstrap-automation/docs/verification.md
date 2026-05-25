# 验证记录

## 本地验证

执行目录：

```text
examples/workflows/project-bootstrap-automation
```

单元测试：

```powershell
go test ./scripts
```

结果：

```text
ok  	github.com/gitlink-org/gitlink-cli/examples/workflows/project-bootstrap-automation/scripts
```

dry-run 复现：

```powershell
.\scripts\run_demo.ps1
```

结果：

```text
已生成初始化报告: outputs\puygob236_gitlink-bootstrap-demo_20260524_072107_bootstrap_report.md
已生成初始化摘要: outputs\puygob236_gitlink-bootstrap-demo_20260524_072107_summary.md
已生成文件清单: outputs\puygob236_gitlink-bootstrap-demo_20260524_072107_manifest.json
已生成命令日志: outputs\command_log_20260524_072107.json
模式: dry-run
计划/执行 gitlink-cli 调用: 7 个
```

## 真实仓库验证计划

目标仓库：

```text
puygob236/gitlink-bootstrap-demo
```

验证步骤：

1. 确认 GitLink 认证可用。
2. 创建或确认目标仓库存在。
3. 执行 `.\scripts\run_demo.ps1 -Apply`。
4. 检查分支、Issue 和输出报告。
5. 如需展示回写能力，执行 `.\scripts\run_demo.ps1 -Apply -PublishIssueNumber <number>`。

## 真实仓库验证结果

目标仓库：

```text
https://gitlink.org.cn/puygob236/gitlink-bootstrap-demo
```

已完成验证：

- `repo +info`：成功读取 `puygob236/gitlink-bootstrap-demo` 仓库信息。
- `branch +list`：成功读取 `master`、`develop`、`release/v0.1` 分支。
- `issue +create`：成功创建初始化 Issue，生成项目任务清单。
- `issue +comment`：成功将初始化摘要回写到 Issue。

回写验证命令：

```powershell
.\scripts\run_demo.ps1 -Config examples\verification_comment_config.json -Apply -PublishIssueNumber 4
```

回写验证结果：

```text
模式: apply
计划/执行 gitlink-cli 调用: 3 个
```

命令日志中 3 条调用状态均为 `ok`，无 stderr。
