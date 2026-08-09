<#
一键复现脚本：运行科研团队洞察工作流并生成示例输出。
用法: .\scripts\run_demo.ps1 [-Out <dir>] [-NoFetch]
  -NoFetch  使用 examples/demo_outputs/data 下已采集数据离线复现
#>
param(
  [string]$Out = "./examples/demo_outputs",
  [switch]$NoFetch
)

$ErrorActionPreference = "Stop"
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $here "..")

# 解析 gitlink-cli 可执行文件（Windows 上为 .CMD 包装）
$cliBin = "gitlink-cli"
$src = (Get-Command gitlink-cli -ErrorAction SilentlyContinue).Source
if ($src -and $src.ToLower().EndsWith(".ps1")) { $cliBin = ($src -replace '\.ps1$', '.CMD') }

$args = @("run", "./scripts",
  "--config", "config/team.example.json",
  "--out", $Out,
  "--cli-bin", $cliBin)
if ($NoFetch) { $args += @("--no-fetch", "--data-root", (Join-Path $Out "data")) }

Write-Host "运行：go $($args -join ' ')"
& go @args
