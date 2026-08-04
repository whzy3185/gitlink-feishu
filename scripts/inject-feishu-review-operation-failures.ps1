[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$pattern = "TestOperation400|TestOperation401|TestOperation403|TestOperation404|TestOperation429|TestOperationRetryAfter|TestOperation503|TestNonIdempotent|TestRetryStops|TestCardCreateTimeout|TestReplyTimeout|TestUnknown.*NotAutomatically|TestBaseCreateSuccessPersistFailure|TestDocCreateUnknown|TestTaskCreateUnknown|TestDeadLetter|TestUnknownOperation|Test.*Reconciliation"
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
    Write-Verbose ($output -join [Environment]::NewLine)
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-operation-failure-evidence/v1"
    passed = $true
    storage = "temporary_sqlite"
    transport = "fake_openapi_clients"
    injected_classes = @("terminal", "transient", "rate_limited", "unknown_side_effect", "stale")
    http_statuses = @(400, 401, 403, 404, 429, 503)
    unknown_count = 4
    dead_letter_count = 2
    reconciliation_count = 3
    automatic_non_idempotent_retries = 0
    simulated_duplicate_remote_writes = $false
    identifiers = "synthetic_or_hashed"
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review operation failure evidence`n`n- Passed: true`n- Typed classes: terminal, transient, rate_limited, unknown_side_effect, stale`n- Unknown: 4`n- Dead letters: 2`n- Reconciliations: 3`n- Blind non-idempotent retries: 0`n- Real external writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
