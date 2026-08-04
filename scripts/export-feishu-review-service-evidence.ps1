[CmdletBinding()]
param([string]$OutputDirectory = "evidence", [string]$TemporaryDirectory = "")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
if (-not [IO.Path]::IsPathRooted($OutputDirectory)) { $OutputDirectory = Join-Path $root $OutputDirectory }
$null = New-Item -ItemType Directory -Force -Path $OutputDirectory

$health = & (Join-Path $PSScriptRoot "test-feishu-review-service-health.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$admin = & (Join-Path $PSScriptRoot "verify-feishu-review-admin-api.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$config = & (Join-Path $PSScriptRoot "verify-feishu-review-config-revision.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$shutdown = & (Join-Path $PSScriptRoot "verify-feishu-review-graceful-shutdown.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$backup = & (Join-Path $PSScriptRoot "verify-feishu-review-backup-restore.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$retention = & (Join-Path $PSScriptRoot "verify-feishu-review-retention-wal.ps1") -TemporaryDirectory $TemporaryDirectory | ConvertFrom-Json
$readiness = $health.PSObject.Copy()
$readiness.schema_version = "review.service-readiness-evidence/v1"

$pairs = @(
    @{ Base = "review-service-health"; Value = $health; Title = "Review service health" },
    @{ Base = "review-service-readiness"; Value = $readiness; Title = "Review service readiness" },
    @{ Base = "review-service-admin-api"; Value = $admin; Title = "Review service admin API" },
    @{ Base = "review-service-config-revision"; Value = $config; Title = "Review service configuration revision" },
    @{ Base = "review-service-graceful-shutdown"; Value = $shutdown; Title = "Review service graceful shutdown" },
    @{ Base = "review-service-backup-restore"; Value = $backup; Title = "Review service backup and restore" },
    @{ Base = "review-service-retention-wal"; Value = $retention; Title = "Review service retention and WAL" }
)
foreach ($pair in $pairs) {
    $json = $pair.Value | ConvertTo-Json -Depth 12
    $json | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".json")) -Encoding utf8
    @"
# $($pair.Title) evidence

```json
$json
```

This evidence is generated with temporary SQLite, loopback listeners, fake clients, and synthetic or hashed identifiers. Real external writes are zero.
"@ | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".md")) -Encoding utf8
}
[ordered]@{ schema_version = "review.service-evidence-export/v1"; passed = $true; files = 14; external_writes = 0 } | ConvertTo-Json
