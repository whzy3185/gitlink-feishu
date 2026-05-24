# 验证记录

## 环境

- Windows PowerShell
- Python 3
- `@gitlink-ai/cli` 0.1.13

## 已验证的真实仓库

### `puygob236/gitlink-cli`

- `repo +info` 可访问
- `issue +list` 可访问
- `pr +list` 可访问
- `release +list` 可访问
- 已完成 Issue 摘要回写验证

### `Gitlink/gitlink-cli`

- `repo +info` 可访问
- `issue +list` 可访问
- `pr +list` 可访问
- `release +list` 可访问
- 当前可提取到的统计结果：
  - Issues: 15
  - PR: 20
  - Release: 11

## 本地输出

已生成的文件：

- `outputs/Gitlink_gitlink-cli_20260515_040153_report.md`
- `outputs/Gitlink_gitlink-cli_20260515_040153_summary.json`
- `outputs/Gitlink_gitlink-cli_20260515_121523_report.md`
- `outputs/Gitlink_gitlink-cli_20260515_121523_release_notes.md`
- `outputs/Gitlink_gitlink-cli_20260515_121523_summary.json`
- `outputs/puygob236_gitlink-cli_20260515_121544_report.md`
- `outputs/puygob236_gitlink-cli_20260515_121544_release_notes.md`
- `outputs/puygob236_gitlink-cli_20260515_121544_summary.json`
- `outputs/puygob236_gitlink-cli_20260515_121845_report.md`
- `outputs/puygob236_gitlink-cli_20260515_121845_release_notes.md`
- `outputs/puygob236_gitlink-cli_20260515_121845_summary.json`
- `outputs/Gitlink_gitlink-cli_20260520_140525_report.md`
- `outputs/Gitlink_gitlink-cli_20260520_140525_release_notes.md`
- `outputs/Gitlink_gitlink-cli_20260520_140525_summary.json`
- `outputs/puygob236_gitlink-cli_20260520_143224_report.md`
- `outputs/puygob236_gitlink-cli_20260520_143224_release_notes.md`
- `outputs/puygob236_gitlink-cli_20260520_143224_summary.json`

其中 `20260520_140525` 对应公开仓库数据分析验证，`20260520_143224` 对应参赛仓库采集与 Issue 回写验证。

## 示例产物

`outputs/` 是运行时目录，仓库交付中同时提供了轻量示例：

- `examples/demo_outputs/Gitlink_gitlink-cli_report.md`
- `examples/demo_outputs/Gitlink_gitlink-cli_release_notes.md`
- `examples/demo_outputs/puygob236_gitlink-cli_report.md`
- `examples/demo_outputs/puygob236_gitlink-cli_release_notes.md`

## 复现方式

```powershell
.\scripts\run_demo.ps1
```
