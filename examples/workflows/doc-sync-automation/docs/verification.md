# 真实平台验证记录

验证日期：2026-07-05；平台：生产环境 gitlink.org.cn；CLI：包含 `file` 命令组（PR #330）的本地构建。

## 1. 文档对自动发现 + 漂移检测（只读，dry-run）

```bash
python3 scripts/doc_sync_workflow.py --owner Taoyouce --repo gitlink-cli --output-dir outputs
```

结果（完整报告见 [`../examples/demo-outputs/doc-sync-Taoyouce-gitlink-cli.md`](../examples/demo-outputs/doc-sync-Taoyouce-gitlink-cli.md)）：

- `repo +tree` 拉取根目录后按命名约定**自动识别** `README.md ⇄ README.zh-CN.md`
- `file +view --raw` 拉取两版全文后检出**真实存在的漂移**：
  - 🟡 代码块数不一致：英文 32 个 vs 中文 29 个
  - 🟡 表格行数不一致：英文 43 行 vs 中文 40 行（功能表滞后）
- 退出码 `0`（无严重漂移）

该漂移与本仓库实际情况一致：英文 README 的部分示例段落未同步到中文版。

## 2. `--apply` 真实回写 tracking issue

```bash
python3 scripts/doc_sync_workflow.py --owner Taoyouce --repo gitlink-cli --apply
```

- `issue +create` 成功在 fork 仓库创建 tracking issue：
  `[doc-sync] 中英文档漂移报告`（issue #1）
- 通过 `issue +list` API 回执确认：`project_issues_index=1`，正文为完整漂移报告

## 3. 确定性回归护栏

```bash
python3 tests/test_drift.py
# Ran 9 tests ... OK
```

覆盖：结构解析（标题/代码块/表格/版本号）、代码围栏内标题不误判、四类漂移全部检出、相同文档零发现、同输入同输出确定性、文档对发现约定、报告渲染。
