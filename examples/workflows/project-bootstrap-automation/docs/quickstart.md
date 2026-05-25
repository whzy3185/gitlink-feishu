# 快速开始

## 1. 进入目录

```powershell
cd examples\workflows\project-bootstrap-automation
```

## 2. 运行 dry-run

```powershell
.\scripts\run_demo.ps1
```

该命令不会写入 GitLink，只生成初始化材料和命令计划。

## 3. 查看输出

```powershell
Get-ChildItem outputs
```

重点查看：

- `*_bootstrap_report.md`
- `*_summary.md`
- `*_manifest.json`
- `command_log_*.json`

## 4. 执行单元测试

```powershell
go test ./scripts
```

## 5. 执行真实写入

确认目标仓库和认证状态后执行：

```powershell
.\scripts\run_demo.ps1 -Apply
```

执行真实写入前，应先通过 `gitlink-cli auth login` 或当前环境已配置的认证方式完成 GitLink 登录。

如需创建目标仓库：

```powershell
.\scripts\run_demo.ps1 -Apply -CreateRepo
```

如需把摘要发布到指定 Issue：

```powershell
.\scripts\run_demo.ps1 -Apply -PublishIssueNumber 1
```
