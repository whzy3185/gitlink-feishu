# 真实平台验证记录

验证日期：2026-07-05；平台：生产环境 gitlink.org.cn；CLI：包含 `file` 命令组（PR #330）的本地构建。

## 1. 论文笔记仓库（预期低分）

```bash
python3 scripts/repro_audit.py --owner GengzhaoWang --repo UAV-Paper
```

- 结果：**8/100 — D（复现困难）**，退出码 `2`
- 证据：仅有 README（无运行说明章节），无 LICENSE / 引用 / 依赖清单 / 入口 / 数据说明 / 测试 / release
- 完整报告：[`../examples/demo-outputs/repro-audit-GengzhaoWang-UAV-Paper.md`](../examples/demo-outputs/repro-audit-GengzhaoWang-UAV-Paper.md)

## 2. 科研工具代码仓库（中等分，建议明确）

```bash
python3 scripts/repro_audit.py --owner momomym --repo paper-summarizer
```

- 结果：**58/100 — C（存在明显缺口）**，退出码 `2`
- 检出：LICENSE ✅、requirements.txt ✅、scripts 入口 ✅、tests ✅
- 缺口：README 无运行章节（部分分 8/15）、无引用信息、无数据说明、无 release
- 完整报告：[`../examples/demo-outputs/repro-audit-momomym-paper-summarizer.md`](../examples/demo-outputs/repro-audit-momomym-paper-summarizer.md)

两个真实仓库得分区分度显著（8 vs 58），每项证据均可在仓库网页人工复核。

## 3. 确定性回归护栏

```bash
python3 tests/test_audit.py
# Ran 7 tests ... OK
```

覆盖：满分样例（100/A）、空仓库（0/D）、README 无运行说明的部分分、README 内引用识别、scripts 目录作为入口、同输入同输出确定性、失败项必附修复建议。
