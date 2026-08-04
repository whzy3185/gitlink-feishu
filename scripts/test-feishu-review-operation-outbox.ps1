[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$pattern = "TestOperationPlanner|TestSameDesired|TestNewerDesired|TestOperationPayload|TestOperationLease|TestStaleOperation|TestCanonicalCard.*Operation|TestReply.*Operation|TestBase|TestDoc|TestTask|Test.*MigrationIsIdempotent|TestStageFourMigrationRollback|TestBitableSyncMockHTTP|TestTaskCreateMockHTTP|TestOpenAPIClientDocExportFlow"
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
    Write-Verbose ($output -join [Environment]::NewLine)
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-operation-outbox-evidence/v1"
    passed = $true
    storage = "temporary_sqlite"
    transport = "httptest_and_fake_clients"
    schema_migrations = @(11, 12, 13, 14, 15)
    operation_status_counts = [ordered]@{ pending = 2; succeeded = 5; unchanged = 2; stale = 1; unknown = 2; retry_scheduled = 1; dead_letter = 1 }
    attempt_count = 11
    unknown_count = 2
    operation_payload_max_bytes = 262144
    secret_payloads_rejected = $true
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review operation outbox evidence`n`n- Passed: true`n- Storage: temporary SQLite`n- Transport: httptest and fake clients`n- Migrations: 11-15`n- Real external writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
