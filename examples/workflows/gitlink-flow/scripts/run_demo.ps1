# gitlink-flow 一键复现脚本
#
# 运行社区运营自动化端到端工作流，对真实 GitLink 仓库生成社区运营周报。
#
# 用法：
#   .\scripts\run_demo.ps1                                  # 默认 Gitlink/gitlink-cli
#   .\scripts\run_demo.ps1 -Owner <owner> -Repo <repo>
#   .\scripts\run_demo.ps1 -Format json                    # JSON 输出

param(
    [string]$Owner = "Gitlink",
    [string]$Repo = "gitlink-cli",
    [string]$Format = "markdown"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# 说明：本脚本以 UTF-8 BOM 保存，确保 Windows PowerShell 5.1 正确解析中文。
# 不修改控制台代码页，避免在中文路径下破坏向 python 传递的参数。

Write-Host "=== gitlink-flow 社区运营自动化工作流 ==="
Write-Host "目标仓库：$Owner/$Repo"

New-Item -ItemType Directory -Force "outputs" | Out-Null
$ext = if ($Format -eq "json") { "json" } else { "md" }
$out = "outputs/${Owner}_${Repo}_flow.$ext"

python src/flow.py --owner $Owner --repo $Repo --format $Format --output $out
$code = $LASTEXITCODE

if ($code -eq 0) {
    Write-Host "`n工作流完成，周报已输出到 $out" -ForegroundColor Green
} else {
    Write-Host "`n工作流失败，退出码 $code" -ForegroundColor Red
}
exit $code
