[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$DatabasePath,
    [string]$Installation = ""
)

$ErrorActionPreference = "Stop"
$inspector = Join-Path $PSScriptRoot "inspect-feishu-review-scope.ps1"
& $inspector -DatabasePath $DatabasePath -Resource "feishu_card" -Installation $Installation
if ($LASTEXITCODE -ne 0) {
    throw "Card mapping verification failed"
}
