param(
    [switch]$Full
)

$ErrorActionPreference = "Stop"

function Assert-NativeSuccess {
    param([string]$Step)
    if ($LASTEXITCODE -ne 0) {
        throw "$Step failed with exit code $LASTEXITCODE"
    }
}

Write-Host "1/8 Go formatting check"
$trackedChanges = @(git diff --name-only --diff-filter=ACMR)
$untrackedChanges = @(git ls-files --others --exclude-standard)
$goFiles = @($trackedChanges + $untrackedChanges |
    Where-Object { $_ -like "*.go" } |
    Sort-Object -Unique)
$changed = @()
if ($goFiles.Count -gt 0) {
    $changed = @(& gofmt -l $goFiles)
}
if ($changed) {
    throw "gofmt required:`n$changed"
}

Write-Host "2/8 Round 2 PowerShell deployment tool contracts"
& (Join-Path $PSScriptRoot "test-round2-powershell.ps1")

Write-Host "3/8 Feishu Review collaboration tests"
go test ./shortcuts/feishu
Assert-NativeSuccess "Feishu Review collaboration tests"

Write-Host "4/8 Enterprise WeChat adapter tests"
go test ./shortcuts/wecom
Assert-NativeSuccess "Enterprise WeChat adapter tests"

Write-Host "5/8 Review Core and Agent orchestration tests"
go test ./shortcuts/workflow
Assert-NativeSuccess "Review Core and Agent orchestration tests"

Write-Host "6/8 Enterprise WeChat sidecar contract tests"
node --test bridges/wecom/test/*.test.mjs
Assert-NativeSuccess "Enterprise WeChat sidecar contract tests"

Write-Host "7/8 Full repository build"
go build ./...
Assert-NativeSuccess "Full repository build"

Write-Host "8/8 P2-P5 vet"
go vet ./internal/collab ./shortcuts/feishu ./shortcuts/wecom ./shortcuts/workflow
Assert-NativeSuccess "P2-P5 vet"

if ($Full) {
    Write-Host "Optional full historical baseline audit"
    go test ./...
    Assert-NativeSuccess "Full repository historical baseline tests"
}

Write-Host "Round 2 P5 local code gate passed. No Feishu, WeCom, or GitLink write smoke was executed."
