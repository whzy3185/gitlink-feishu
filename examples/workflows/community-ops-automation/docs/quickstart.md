# 快速开始

## 一键运行

直接运行一键脚本：

```powershell
.\scripts\run_demo.ps1
```

脚本会自动通过 `npm exec` 找到 `@gitlink-ai/cli`，把 `gitlink-cli` 放到临时 PATH 里，再执行：

- 仓库信息采集
- Issue 列表采集
- PR 列表采集
- Release 列表采集
- 周报生成
- Release Notes 草稿生成

## 配置切换

- `examples/demo_active_config.json`：公开仓库验证配置，默认指向 `Gitlink/gitlink-cli`
- `examples/sample_config.json`：参赛仓库验证配置，默认指向 `puygob236/gitlink-cli`

## 输出

- `outputs/*_report.md`
- `outputs/*_release_notes.md`
- `outputs/*_summary.json`

## 已验证事实

- `puygob236/gitlink-cli` 已完成采集、报告生成和 Issue 摘要回写验证
- `Gitlink/gitlink-cli` 可生成带统计内容的周报和 Release Notes
