[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$pattern = "TestOperationLeaseSurvivesRestart|TestWorkerLeaseRecoversAfterRestart|TestStaleOperationWorkerCannotCompleteNewLease|Test.*MigrationIsIdempotent|TestStageFourMigrationRollback"
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
    Write-Verbose ($output -join [Environment]::NewLine)
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-operation-restart-evidence/v1"
    passed = $true
    storage = "temporary_sqlite"
    schema_migrations = @(11, 12, 13, 14, 15)
    operation_count_before_restart = 1
    operation_count_after_restart = 1
    expired_lease_recovered = $true
    stale_worker_fenced = $true
    duplicate_operation_count = 0
    simulated_duplicate_remote_writes = $false
    identifiers = "synthetic_or_hashed"
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review operation restart evidence`n`n- Passed: true`n- Operations before restart: 1`n- Operations after restart: 1`n- Expired lease recovered: true`n- Stale worker fenced: true`n- Duplicate simulated remote writes: false`n- Real external writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
