[CmdletBinding()]
param(
    [string]$OutputFile = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$started = [DateTimeOffset]::UtcNow
$results = @()

$checks = @(
    @{ Name = "round2-deployment"; Script = "test-round2-powershell.ps1"; Arguments = @() },
    @{ Name = "dual-chat"; Script = "test-feishu-review-dual-chat.ps1"; Arguments = @() },
    @{ Name = "webhook"; Script = "test-feishu-review-webhook.ps1"; Arguments = @() },
    @{ Name = "operation-outbox"; Script = "test-feishu-review-operation-outbox.ps1"; Arguments = @() },
    @{ Name = "operation-failures"; Script = "inject-feishu-review-operation-failures.ps1"; Arguments = @() },
    @{ Name = "coalescing"; Script = "verify-feishu-review-coalescing.ps1"; Arguments = @() },
    @{ Name = "operation-restart"; Script = "verify-feishu-review-operation-restart.ps1"; Arguments = @() },
    @{ Name = "reconciliation"; Script = "verify-feishu-review-reconciliation.ps1"; Arguments = @() },
    @{ Name = "migration-restart"; Script = "verify-feishu-review-restart.ps1"; Arguments = @() },
    @{ Name = "service-health"; Script = "test-feishu-review-service-health.ps1"; Arguments = @() },
    @{ Name = "admin-api"; Script = "verify-feishu-review-admin-api.ps1"; Arguments = @() },
    @{ Name = "backup-restore"; Script = "verify-feishu-review-backup-restore.ps1"; Arguments = @() },
    @{ Name = "configuration-revision"; Script = "verify-feishu-review-config-revision.ps1"; Arguments = @() },
    @{ Name = "graceful-shutdown"; Script = "verify-feishu-review-graceful-shutdown.ps1"; Arguments = @() },
    @{ Name = "retention-wal"; Script = "verify-feishu-review-retention-wal.ps1"; Arguments = @() }
)

Push-Location $root
try {
    foreach ($check in $checks) {
        $path = Join-Path $PSScriptRoot $check.Script
        $checkStarted = [DateTimeOffset]::UtcNow
        try {
            $arguments = @($check.Arguments)
            $output = @(& $path @arguments 2>&1)
            $results += [ordered]@{
                name        = $check.Name
                script      = $check.Script
                passed      = $true
                duration_ms = [int64]([DateTimeOffset]::UtcNow - $checkStarted).TotalMilliseconds
            }
            Write-Host ("PASS {0}" -f $check.Name)
            Write-Verbose ($output -join [Environment]::NewLine)
        } catch {
            $results += [ordered]@{
                name          = $check.Name
                script        = $check.Script
                passed        = $false
                duration_ms   = [int64]([DateTimeOffset]::UtcNow - $checkStarted).TotalMilliseconds
                error_summary = $_.Exception.Message
            }
            throw
        }
    }
} finally {
    Pop-Location
}

$summary = [ordered]@{
    schema_version  = "review.powershell-cross-platform/v1"
    validation_mode = "offline"
    platform        = [Environment]::OSVersion.Platform.ToString()
    powershell      = $PSVersionTable.PSVersion.ToString()
    started_at      = $started.ToString("O")
    finished_at     = [DateTimeOffset]::UtcNow.ToString("O")
    passed          = @($results | Where-Object { -not $_.passed }).Count -eq 0
    checks          = $results
    external_writes = 0
}
$json = $summary | ConvertTo-Json -Depth 8
if ($OutputFile) {
    $fullOutput = [IO.Path]::GetFullPath((Join-Path $root $OutputFile))
    [IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($fullOutput)) | Out-Null
    [IO.File]::WriteAllText($fullOutput, $json, [Text.UTF8Encoding]::new($false))
}
$json
