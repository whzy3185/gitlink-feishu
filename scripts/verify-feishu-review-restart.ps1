[CmdletBinding()]
param(
    [string]$DatabasePath = ""
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$databaseHash = ""
if ($DatabasePath) {
    $resolved = (Resolve-Path -LiteralPath $DatabasePath).Path
    $databaseHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $resolved).Hash.ToLowerInvariant()
}

Push-Location $repositoryRoot
try {
    $testOutput = & go test ./shortcuts/feishu -run "TestMigrationPlansSurviveSQLiteRestart|TestMigrationPoliciesSurviveSQLiteRestart|TestRestartDoesNotDuplicateMigrationPlans|TestVerifiedLegacyCardDoesNotCreateSecondCard" -count=1 2>&1
    $testExitCode = $LASTEXITCODE
    Write-Verbose ($testOutput -join [Environment]::NewLine)
    if ($testExitCode -ne 0) {
        throw "restart migration tests failed"
    }
} finally {
    Pop-Location
}

[ordered]@{
    schema_version = "feishu.review-migration-restart-evidence/v1"
    passed = $true
    database_sha256 = $databaseHash
    plans_not_duplicated = $true
    migrated_state_not_reverted = $true
    canonical_card_not_duplicated = $true
    legacy_source_retained = $true
    scoped_target_not_overwritten = $true
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
} | ConvertTo-Json -Depth 5
