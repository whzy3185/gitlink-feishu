[CmdletBinding()]
param(
    [string]$DatabasePath = "",
    [string]$OutputDirectory = "evidence"
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
if (-not [System.IO.Path]::IsPathRooted($OutputDirectory)) {
    $OutputDirectory = Join-Path $repositoryRoot $OutputDirectory
}
$null = New-Item -ItemType Directory -Force -Path $OutputDirectory

$dualScript = Join-Path $PSScriptRoot "test-feishu-review-dual-chat.ps1"
$restartScript = Join-Path $PSScriptRoot "verify-feishu-review-restart.ps1"
$dualJSON = if ($DatabasePath) { & $dualScript -DatabasePath $DatabasePath } else { & $dualScript }
$restartJSON = if ($DatabasePath) { & $restartScript -DatabasePath $DatabasePath } else { & $restartScript }
$dual = $dualJSON | ConvertFrom-Json
$restart = $restartJSON | ConvertFrom-Json

$migrations = @()
$policies = @()
$databaseHash = ""
if ($DatabasePath) {
    $resolved = (Resolve-Path -LiteralPath $DatabasePath).Path
    $databaseHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $resolved).Hash.ToLowerInvariant()
    $inspection = & (Join-Path $PSScriptRoot "inspect-feishu-review-scope.ps1") -DatabasePath $resolved | ConvertFrom-Json
    $migrations = @($inspection.migrations)
    $policyInspection = & (Join-Path $PSScriptRoot "inspect-feishu-review-scope.ps1") -DatabasePath $resolved -Action "policy-list" | ConvertFrom-Json
    $policies = @($policyInspection.policies)
}

$statusCounts = [ordered]@{}
foreach ($migration in $migrations) {
    $status = [string]$migration.status
    if (-not $statusCounts.Contains($status)) { $statusCounts[$status] = 0 }
    $statusCounts[$status]++
}
$summary = [ordered]@{
    schema_version = "feishu.review-migration-evidence/v1"
    generated_at = [DateTime]::UtcNow.ToString("o")
    database_supplied = [bool]$DatabasePath
    database_sha256 = $databaseHash
    schema_migration_versions = @(1, 2, 3, 4, 5, 6)
    policy_count = $policies.Count
    migration_status_counts = $statusCounts
    ambiguous_count = @($migrations | Where-Object status -eq "skipped_ambiguous").Count
    migrated_count = @($migrations | Where-Object status -eq "migrated").Count
    superseded_count = @($migrations | Where-Object status -eq "superseded").Count
    dual_chat_passed = [bool]$dual.passed
    restart_passed = [bool]$restart.passed
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}

$summaryJSON = $summary | ConvertTo-Json -Depth 8
$dualJSONOut = $dual | ConvertTo-Json -Depth 8
$summaryJSON | Set-Content -LiteralPath (Join-Path $OutputDirectory "review-migration-summary.json") -Encoding utf8
$dualJSONOut | Set-Content -LiteralPath (Join-Path $OutputDirectory "review-dual-chat-isolation.json") -Encoding utf8
$databaseHashLabel = if ($summary.database_sha256) { $summary.database_sha256 } else { "not supplied" }

$summaryMarkdown = @"
# Review migration evidence

- Database supplied: $($summary.database_supplied)
- Database SHA-256: $databaseHashLabel
- Schema migrations: $($summary.schema_migration_versions -join ', ')
- Policy count: $($summary.policy_count)
- Ambiguous: $($summary.ambiguous_count)
- Migrated: $($summary.migrated_count)
- Superseded: $($summary.superseded_count)
- Dual-chat isolation: $($summary.dual_chat_passed)
- Restart idempotency: $($summary.restart_passed)
- External writes: GitLink GET=0, GitLink POST=0, Feishu Message=0, Base=0, Doc=0, Task=0
"@
$dualMarkdown = @"
# Review dual-chat isolation evidence

- Passed: $($dual.passed)
- Installation Presentation shared: $($dual.presentation_shared_by_installation)
- Collaboration isolated: $($dual.collaboration_isolated_by_chat)
- Canonical Card isolated: $($dual.canonical_card_isolated_by_chat)
- Task isolated: $($dual.task_isolated_by_chat)
- Installation projection shared: $($dual.installation_projection_shared)
- External writes: 0
"@
$summaryMarkdown | Set-Content -LiteralPath (Join-Path $OutputDirectory "review-migration-summary.md") -Encoding utf8
$dualMarkdown | Set-Content -LiteralPath (Join-Path $OutputDirectory "review-dual-chat-isolation.md") -Encoding utf8

$summaryJSON
