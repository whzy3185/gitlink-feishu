# GitLink CLI 一键安装脚本 (Windows PowerShell)
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  GitLink CLI 安装脚本 (Windows)" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

$binary = "gitlink-cli-windows-amd64.exe"
$url = "https://gitlink.org.cn/Gitlink/gitlink-cli/releases/download/latest/$binary"
$installDir = "$env:LOCALAPPDATA\gitlink-cli"
$dest = "$installDir\gitlink-cli.exe"

Write-Host "下载地址: $url"

# 创建安装目录
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# 下载
try {
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
    Write-Host "下载完成" -ForegroundColor Green
} catch {
    Write-Host "下载失败: $_" -ForegroundColor Red
    exit 1
}

# 添加到 PATH（用户级别）
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$installDir;$userPath",
        "User"
    )
    Write-Host "已添加到用户 PATH" -ForegroundColor Green
}

Write-Host ""
Write-Host "安装完成！" -ForegroundColor Green
Write-Host "请重新打开终端，运行 gitlink-cli --help 验证安装" -ForegroundColor Yellow
